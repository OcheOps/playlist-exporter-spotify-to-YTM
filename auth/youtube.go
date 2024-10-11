package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	//"net/http"
	"os"
	"playlist-exporter/config"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

func AuthenticateYouTube(cfg *config.Config) (*youtube.Service, error) {
	ctx := context.Background()
	config := &oauth2.Config{
		ClientID:     cfg.YouTubeClientID,
		ClientSecret: cfg.YouTubeClientSecret,
		RedirectURL:  "http://localhost:8080/youtube-callback",
		Scopes: []string{
			youtube.YoutubeScope,
			youtube.YoutubeForceSslScope,
		},
		Endpoint: google.Endpoint,
	}

	// Check if we have a token file
	tokFile := "youtube-token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok, err = getTokenFromWeb(config)
		if err != nil {
			return nil, fmt.Errorf("error getting token from web: %v", err)
		}
		saveToken(tokFile, tok)
	}

	return youtube.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, tok)))
}

func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser: \n%v\n", authURL)

	if err := browser.OpenURL(authURL); err != nil {
		log.Println("Failed to open browser automatically. Please open the URL manually.")
	}

	fmt.Println("Enter the authorization code:")
	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %v", err)
	}
	return tok, nil
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
