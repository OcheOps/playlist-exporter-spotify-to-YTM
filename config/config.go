package config

import "os"

type Config struct {
	SpotifyClientID     string
	SpotifyClientSecret string
	YouTubeClientID     string
	YouTubeClientSecret string
}

func NewConfig() *Config {
	return &Config{
		SpotifyClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		SpotifyClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		YouTubeClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
		YouTubeClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
	}
}