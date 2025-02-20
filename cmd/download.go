package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/Ceralex/spotydw/internal/parser"
	"github.com/Ceralex/spotydw/internal/spotify"
	"github.com/spf13/cobra"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download <URL> [URL...]",
	Short: "Download a track, album, or playlist from Spotify",
	Long: `Download audio from Spotify by providing a track, album, or playlist URL.
	You can provide multiple URLs at once to download multiple resources.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Initialize Spotify client once for all downloads
		client, err := spotify.NewClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to initialize Spotify client: %w", err)
		}

		// Process each URL
		for _, arg := range args {
			if err := processURL(ctx, client, arg); err != nil {
				log.Printf("Error processing %s: %v", arg, err)
			}
		}

		return nil
	},
}

func processURL(ctx context.Context, client *spotify.Client, url string) error {
	typeUrl, err := parser.GetTypeUrl(url)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	if typeUrl == parser.UrlTypeUnknown {
		return fmt.Errorf("unknown URL type: %s", url)
	}

	id := parser.ExtractSpotifyID(url)
	if id == "" {
		return fmt.Errorf("failed to extract ID from URL: %s", url)
	}

	switch typeUrl {
	case parser.UrlTypeSpotifyTrack:
		if err := client.DownloadTrack(ctx, id); err != nil {
			return fmt.Errorf("failed to download track: %w", err)
		}
	case parser.UrlTypeSpotifyAlbum:
		return fmt.Errorf("album download not yet implemented")
	case parser.UrlTypeSpotifyPlaylist:
		return fmt.Errorf("playlist download not yet implemented")
	default:
		return fmt.Errorf("unsupported URL type: %s", url)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// downloadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
