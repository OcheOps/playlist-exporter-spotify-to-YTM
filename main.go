package main

import (
	"fmt"
	"log"
	"os"
	"playlist-exporter/auth"
	"playlist-exporter/config"
	"playlist-exporter/exporter"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	dryRun bool
	cfg    *config.Config
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg = config.NewConfig()

	rootCmd := &cobra.Command{
		Use:   "playlist-exporter",
		Short: "Export playlists from Spotify to YouTube Music",
		Run:   run,
	}

	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Perform a dry run without actually creating playlists")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	spotifyClient, err := auth.AuthenticateSpotify(cfg)
	if err != nil {
		log.Fatalf("Failed to authenticate with Spotify: %v", err)
	}

	youtubeService, err := auth.AuthenticateYouTube(cfg)
	if err != nil {
		log.Fatalf("Failed to authenticate with YouTube: %v", err)
	}

	exp := exporter.NewExporter(spotifyClient, youtubeService, cfg, dryRun)
	if err := exp.Run(); err != nil {
		log.Fatalf("Export failed: %v", err)
	}
}