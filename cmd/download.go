package cmd

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/Ceralex/spotydw/internal/spotify"
	"github.com/Ceralex/spotydw/internal/utils"
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
		for _, arg := range args {
			if err := processURL(arg); err != nil {
				log.Printf("Error processing %s: %v", arg, err)
			}
		}

		return nil
	},
}

func processURL(URL string) error {
	if !utils.IsUrl(URL) {
		return fmt.Errorf("invalid URL")
	}

	parsedUrl, err := url.Parse(URL)
	if err != nil {
		return fmt.Errorf("error parsing URL: %v", err)
	}

	switch parsedUrl.Host {
	case "open.spotify.com":
		return spotify.Download(context.Background(), parsedUrl)
	default:
		return fmt.Errorf("unsupported URL")
	}
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}
