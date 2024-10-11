package exporter

import (
	"context"
	"fmt"
	"log"
	"playlist-exporter/config"
	"playlist-exporter/utils"
	"sync"
	"time"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"github.com/zmb3/spotify/v2"
	"golang.org/x/time/rate"
	"google.golang.org/api/youtube/v3"
)

type Exporter struct {
	spotifyClient  *spotify.Client
	youtubeService *youtube.Service
	cfg            *config.Config
	dryRun         bool
	rateLimiter    *rate.Limiter
}

func NewExporter(spotifyClient *spotify.Client, youtubeService *youtube.Service, cfg *config.Config, dryRun bool) *Exporter {
	return &Exporter{
		spotifyClient:  spotifyClient,
		youtubeService: youtubeService,
		cfg:            cfg,
		dryRun:         dryRun,
		rateLimiter:    rate.NewLimiter(rate.Every(time.Second/5), 1), // 5 requests per second
	}
}

func (e *Exporter) Run() error {
	playlists, err := e.fetchSpotifyPlaylists()
	if err != nil {
		return fmt.Errorf("error fetching Spotify playlists: %v", err)
	}

	selectedPlaylists := e.selectPlaylists(playlists)

	for _, playlist := range selectedPlaylists {
		if err := e.exportPlaylist(playlist); err != nil {
			log.Printf("Error exporting playlist %s: %v", playlist.Name, err)
		}
	}

	return nil
}

func (e *Exporter) fetchSpotifyPlaylists() ([]spotify.SimplePlaylist, error) {
	playlists, err := e.spotifyClient.CurrentUsersPlaylists(context.Background())
	if err != nil {
		return nil, err
	}
	return playlists.Playlists, nil
}

func (e *Exporter) selectPlaylists(playlists []spotify.SimplePlaylist) []spotify.SimplePlaylist {
	fmt.Println("Select playlists to export (enter numbers separated by spaces, or 'all' for all playlists):")
	for i, playlist := range playlists {
		fmt.Printf("%d. %s\n", i+1, playlist.Name)
	}

	var input string
	fmt.Scanln(&input)

	if input == "all" {
		return playlists
	}

	selectedIndices := utils.ParseIndices(input)
	var selectedPlaylists []spotify.SimplePlaylist
	for _, index := range selectedIndices {
		if index > 0 && index <= len(playlists) {
			selectedPlaylists = append(selectedPlaylists, playlists[index-1])
		}
	}

	return selectedPlaylists
}

func (e *Exporter) exportPlaylist(playlist spotify.SimplePlaylist) error {
	fmt.Printf("Exporting playlist: %s\n", playlist.Name)

	tracks, err := e.fetchPlaylistTracks(playlist.ID)
	if err != nil {
		return fmt.Errorf("error fetching tracks for playlist %s: %v", playlist.Name, err)
	}

	youtubePlaylist, err := e.createYouTubePlaylist(playlist.Name, playlist.Description)
	if err != nil {
		return fmt.Errorf("error creating YouTube playlist %s: %v", playlist.Name, err)
	}

	p := mpb.New(mpb.WithWidth(64))
	bar := p.AddBar(int64(len(tracks)),
		mpb.PrependDecorators(
			decor.Name("Exporting: "),
			decor.CountersNoUnit("%d / %d"),
		),
		mpb.AppendDecorators(
			decor.Percentage(),
		),
	)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // Limit concurrent goroutines

	for _, track := range tracks {
		wg.Add(1)
		sem <- struct{}{}
		go func(track spotify.PlaylistTrack) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := e.addTrackToYouTubePlaylist(youtubePlaylist, track); err != nil {
				log.Printf("Error adding track %s to YouTube playlist: %v", track.Track.Name, err)
			}
			bar.Increment()
		}(track)
	}

	wg.Wait()
	p.Wait()

	fmt.Printf("Finished exporting playlist: %s\n", playlist.Name)
	return nil
}

func (e *Exporter) fetchPlaylistTracks(playlistID spotify.ID) ([]spotify.PlaylistTrack, error) {
	var allTracks []spotify.PlaylistTrack
	limit := 100
	offset := 0

	for {
		tracks, err := e.spotifyClient.GetPlaylistTracks(context.Background(), playlistID, spotify.Limit(limit), spotify.Offset(offset))
		if err != nil {
			return nil, err
		}

		allTracks = append(allTracks, tracks.Tracks...)

		if len(tracks.Tracks) < limit {
			break
		}
		offset += limit
	}

	return allTracks, nil
}

func (e *Exporter) createYouTubePlaylist(name, description string) (*youtube.Playlist, error) {
	if e.dryRun {
		fmt.Printf("Dry run: Would create YouTube playlist '%s'\n", name)
		return &youtube.Playlist{Id: "dry-run-id"}, nil
	}

	playlist := &youtube.Playlist{
		Snippet: &youtube.PlaylistSnippet{
			Title:       name,
			Description: description,
		},
		Status: &youtube.PlaylistStatus{
			PrivacyStatus: "private",
		},
	}

	if err := e.rateLimiter.Wait(context.Background()); err != nil {
		return nil, err
	}

	call := e.youtubeService.Playlists.Insert([]string{"snippet", "status"}, playlist)
		response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("error creating YouTube playlist: %v", err)
	}

	return response, nil
}

func (e *Exporter) addTrackToYouTubePlaylist(playlist *youtube.Playlist, track spotify.PlaylistTrack) error {
	query := fmt.Sprintf("%s %s", track.Track.Name, track.Track.Artists[0].Name)
	videoID, err := e.searchYouTubeVideo(query)
	if err != nil {
		return fmt.Errorf("error searching for YouTube video: %v", err)
	}

	if e.dryRun {
		fmt.Printf("Dry run: Would add track '%s' to playlist\n", track.Track.Name)
		return nil
	}

	playlistItem := &youtube.PlaylistItem{
		Snippet: &youtube.PlaylistItemSnippet{
			PlaylistId: playlist.Id,
			ResourceId: &youtube.ResourceId{
				Kind:    "youtube#video",
				VideoId: videoID,
			},
		},
	}

	if err := e.rateLimiter.Wait(context.Background()); err != nil {
		return err
	}

	call := e.youtubeService.PlaylistItems.Insert([]string{"snippet"}, playlistItem)
	_, err = call.Do()
	if err != nil {
		return fmt.Errorf("error adding track to YouTube playlist: %v", err)
	}

	return nil
}

func (e *Exporter) searchYouTubeVideo(query string) (string, error) {
	if err := e.rateLimiter.Wait(context.Background()); err != nil {
		return "", err
	}

	call := e.youtubeService.Search.List([]string{"id"}).
		Q(query).
		Type("video").
		MaxResults(1)

	response, err := call.Do()
	if err != nil {
		return "", fmt.Errorf("error searching YouTube video: %v", err)
	}

	if len(response.Items) == 0 {
		return "", fmt.Errorf("no video found for query: %s", query)
	}

	return response.Items[0].Id.VideoId, nil
}