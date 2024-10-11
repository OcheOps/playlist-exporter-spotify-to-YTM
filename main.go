package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/spotify"
	"google.golang.org/api/youtube/v3"
	"google.golang.org/api/option"
)

type Config struct {
	SpotifyToken    string `json:"spotify_token"`
	YouTubeMusicToken string `json:"youtube_music_token"`
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "playlist-exporter",
		Short: "Export playlists between music streaming services",
	}

	var loginCmd = &cobra.Command{
		Use:   "login [service]",
		Short: "Login to a streaming service",
		Args:  cobra.ExactArgs(1),
		Run:   handleLogin,
	}

	var exportCmd = &cobra.Command{
		Use:   "export",
		Short: "Export playlists from Spotify to YouTube Music",
		Run:   handleExport,
	}

	rootCmd.AddCommand(loginCmd, exportCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func handleLogin(cmd *cobra.Command, args []string) {
	service := args[0]
	
	switch service {
	case "spotify":
		handleSpotifyLogin()
	case "youtube":
		handleYouTubeLogin()
	default:
		fmt.Printf("Unsupported service: %s\n", service)
	}
}

func handleSpotifyLogin() {
	config := &oauth2.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8080/callback",
		Scopes: []string{
			"playlist-read-private",
			"playlist-read-collaborative",
		},
		Endpoint: spotify.Endpoint,
	}

	// Here you would implement the OAuth2 flow for Spotify
	// This is a simplified version
	fmt.Println("Please visit:", config.AuthCodeURL("state"))
	fmt.Print("Enter the authorization code: ")
	
	var authCode string
	fmt.Scanln(&authCode)

	token, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		log.Fatalf("Unable to exchange auth code: %v", err)
	}

	saveToken("spotify", token)
}

func handleYouTubeLogin() {
	config := &oauth2.Config{
		ClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
		ClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8080/callback",
		Scopes: []string{
			youtube.YoutubeScope,
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	// Similar OAuth2 flow for YouTube Music
	fmt.Println("Please visit:", config.AuthCodeURL("state"))
	fmt.Print("Enter the authorization code: ")
	
	var authCode string
	fmt.Scanln(&authCode)

	token, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		log.Fatalf("Unable to exchange auth code: %v", err)
	}

	saveToken("youtube", token)
}

func handleExport(cmd *cobra.Command, args []string) {
	// Load tokens
	spotifyToken := loadToken("spotify")
	youtubeToken := loadToken("youtube")

	if spotifyToken == nil || youtubeToken == nil {
		log.Fatal("Please login to both services first")
	}

	// Here you would:
	// 1. Fetch playlists from Spotify
	// 2. For each playlist:
	//    - Search for each song on YouTube Music
	//    - Create a new playlist on YouTube Music
	//    - Add found songs to the new playlist
	fmt.Println("Exporting playlists...")
}

func saveToken(service string, token *oauth2.Token) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	tokenPath := filepath.Join(configDir, "playlist-exporter", fmt.Sprintf("%s_token.json", service))
	os.MkdirAll(filepath.Dir(tokenPath), 0700)

	f, err := os.OpenFile(tokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func loadToken(service string) *oauth2.Token {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	tokenPath := filepath.Join(configDir, "playlist-exporter", fmt.Sprintf("%s_token.json", service))
	f, err := os.Open(tokenPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	token := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(token)
	if err != nil {
		return nil
	}
	return token
}