// auth/spotify.go

package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"playlist-exporter/config"

	"github.com/pkg/browser"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

const redirectURI = "http://localhost:8080/callback"

var (
	auth  *spotifyauth.Authenticator
	ch    = make(chan *spotify.Client)
	state = "abc123"
)

func AuthenticateSpotify(cfg *config.Config) (*spotify.Client, error) {
	auth = spotifyauth.New(
		spotifyauth.WithClientID(cfg.SpotifyClientID),
		spotifyauth.WithClientSecret(cfg.SpotifyClientSecret),
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(spotifyauth.ScopePlaylistReadPrivate),
	)

	// Set up a simple HTTP server to handle the callback
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go http.ListenAndServe(":8080", nil)

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

	if err := browser.OpenURL(url); err != nil {
		log.Println("Failed to open browser automatically. Please open the URL manually.")
	}

	// wait for auth to complete
	client := <-ch

	// use the client to make calls that require authorization
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error getting current user: %v", err)
	}
	fmt.Println("You are logged in as:", user.ID)

	return client, nil
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}

	// use the token to get an authenticated client
	client := spotify.New(auth.Client(r.Context(), tok))
	fmt.Fprintf(w, "Login Completed!")
	ch <- client
}