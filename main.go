package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

var (
	spotifyRedirectURI = "http://localhost:8080/spotify-callback"
	googleRedirectURI  = "http://localhost:8090/ytcallback"
	spotifyAuth        = spotify.NewAuthenticator(spotifyRedirectURI, spotify.ScopePlaylistReadPrivate, spotify.ScopeUserLibraryRead)
	spotifyCh          = make(chan *spotify.Client)
	trackList          = make(chan []spotify.FullTrack)
	playlistName       string
	spotifyState       = "spotify-state"
	googleState        = "google-state"
)

func main() {
	_ = godotenv.Load()

	spotifyClientID := os.Getenv("SPOTIFY_ID")
	spotifyClientSecret := os.Getenv("SPOTIFY_SECRET")
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	if spotifyClientID == "" || spotifyClientSecret == "" || googleClientID == "" || googleClientSecret == "" {
		log.Fatal("Missing environment variables for Spotify or Google")
	}

	spotifyAuth.SetAuthInfo(spotifyClientID, spotifyClientSecret)

	http.HandleFunc("/spotify-callback", completeSpotifyAuth)
	go http.ListenAndServe(":8080", nil)

	http.HandleFunc("/ytcallback", func(w http.ResponseWriter, r *http.Request) {
		completeGoogleAuth(w, r, googleClientID, googleClientSecret)
	})
	go http.ListenAndServe(":8090", nil)

	fmt.Println("Please log in to Spotify:", spotifyAuth.AuthURL(spotifyState))

	googleConf := &oauth2.Config{
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		RedirectURL:  googleRedirectURI,
		Scopes:       []string{"https://www.googleapis.com/auth/youtube"},
		Endpoint:     google.Endpoint,
	}
	fmt.Println("Please log in to YouTube:", googleConf.AuthCodeURL(googleState, oauth2.AccessTypeOffline))

	spotifyClient := <-spotifyCh

	user, err := spotifyClient.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Spotify login successful for:", user.ID)

	playlists, err := spotifyClient.CurrentUsersPlaylists()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Select a playlist to migrate:")
	for i, p := range playlists.Playlists {
		fmt.Printf("[%d] %s\n", i+1, p.Name)
	}
	fmt.Println("[0] Liked Songs")
	fmt.Print("Enter your choice: ")

	var choice int
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		choice, _ = strconv.Atoi(scanner.Text())
	}

	var tracks []spotify.FullTrack
	if choice == 0 {
		limit := 50
		offset := 0
		for {
			page, err := spotifyClient.CurrentUsersTracksOpt(&spotify.Options{
				Limit:  &limit,
				Offset: &offset,
			})
			if err != nil {
				log.Fatal(err)
			}
			for _, saved := range page.Tracks {
				tracks = append(tracks, saved.FullTrack)
			}
			if len(page.Tracks) < limit {
				break
			}
			offset += limit
		}
		playlistName = "Liked Songs"
	} else if choice > 0 && choice <= len(playlists.Playlists) {
		selected := playlists.Playlists[choice-1]
		playlistName = selected.Name
		trackResp, err := spotifyClient.GetPlaylistTracks(selected.ID)
		if err != nil {
			log.Fatal(err)
		}
		for _, t := range trackResp.Tracks {
			tracks = append(tracks, t.Track)
		}
	} else {
		log.Fatal("Invalid selection")
	}

	fmt.Printf("Found %d tracks in '%s'.\n", len(tracks), playlistName)
	trackList <- tracks

	fmt.Println("Waiting for YouTube login at http://localhost:8090/ytcallback ...")
	select {}
}

func completeSpotifyAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := spotifyAuth.Token(spotifyState, r)
	if err != nil {
		http.Error(w, "Spotify token error", http.StatusForbidden)
		log.Fatal(err)
	}
	if r.FormValue("state") != spotifyState {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: expected %s, got %s", spotifyState, r.FormValue("state"))
	}
	client := spotifyAuth.NewClient(tok)
	fmt.Fprintln(w, "Spotify login successful. You can close this tab.")
	spotifyCh <- &client
}

func completeGoogleAuth(w http.ResponseWriter, r *http.Request, clientID, clientSecret string) {
	if r.URL.Query().Get("state") != googleState {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  googleRedirectURI,
		Scopes:       []string{"https://www.googleapis.com/auth/youtube"},
		Endpoint:     google.Endpoint,
	}
	tok, err := conf.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		log.Fatal(err)
	}
	client := conf.Client(context.Background(), tok)
	ytService, err := youtube.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		http.Error(w, "YouTube service error", http.StatusInternalServerError)
		log.Fatal(err)
	}
	fmt.Fprintln(w, "YouTube login successful. You can close this tab.")

	fmt.Println("Transferring tracks to YouTube...")
	tracks := <-trackList

	maxToTransfer := 100
	if len(tracks) > maxToTransfer {
		fmt.Printf("⚠️  Limiting to %d tracks to avoid quota issues.\n", maxToTransfer)
		tracks = tracks[:maxToTransfer]
	}

	ytPlaylist := &youtube.Playlist{
		Snippet: &youtube.PlaylistSnippet{
			Title:       playlistName + " (Migrated)",
			Description: "Migrated from Spotify",
		},
		Status: &youtube.PlaylistStatus{PrivacyStatus: "private"},
	}
	created, err := ytService.Playlists.Insert([]string{"snippet,status"}, ytPlaylist).Do()
	if err != nil {
		log.Fatal("Failed to create playlist:", err)
	}

	migrated := make(map[string]bool)
	migratedFile := "migrated.txt"
	if data, err := os.ReadFile(migratedFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			migrated[line] = true
		}
	}

	file, err := os.OpenFile(migratedFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open migrated.txt: %v", err)
	}
	defer file.Close()

	for i, track := range tracks {
		trackID := fmt.Sprintf("%s - %s", track.Name, track.Artists[0].Name)
		if migrated[trackID] {
			fmt.Printf("[%d/%d] ⏭ Skipping (already migrated): %s\n", i+1, len(tracks), trackID)
			continue
		}
		query := fmt.Sprintf("%s %s topic", track.Name, track.Artists[0].Name)
		res, err := ytService.Search.List([]string{"snippet"}).Q(query).Type("video").MaxResults(5).Do()

		if err != nil || len(res.Items) == 0 {
			log.Printf("❌ Not found (with topic): %s. Trying fallback...", query)
			query = fmt.Sprintf("%s %s", track.Name, track.Artists[0].Name)
			res, err = ytService.Search.List([]string{"snippet"}).Q(query).Type("video").MaxResults(5).Do()
		}

		if err != nil || len(res.Items) == 0 {
			log.Printf("❌ Not found: %s", query)
			continue
		}
		var videoID string
		for _, item := range res.Items {
			if item.Snippet != nil && item.Snippet.ChannelTitle != "" && strings.Contains(item.Snippet.ChannelTitle, "- Topic") {
				videoID = item.Id.VideoId
				break
			}
		}
		if videoID == "" {
			videoID = res.Items[0].Id.VideoId
		}
		item := &youtube.PlaylistItem{
			Snippet: &youtube.PlaylistItemSnippet{
				PlaylistId: created.Id,
				ResourceId: &youtube.ResourceId{
					Kind:    "youtube#video",
					VideoId: videoID,
				},
			},
		}
		_, err = ytService.PlaylistItems.Insert([]string{"snippet"}, item).Do()
		if err != nil {
			log.Printf("⚠️  Failed to add: %s\n", query)
			continue
		}
		fmt.Printf("[%d/%d] ✔ %s\n", i+1, len(tracks), query)
		_, _ = file.WriteString(trackID + "\n")
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("🎉 Transfer complete!")
}
