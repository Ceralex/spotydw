package cmd

import (
	"fmt"

	"github.com/Ceralex/spotydw/internal/utils"
	"github.com/spf13/cobra"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a track, album or playlist from Spotify",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			typeUrl, err := utils.GetTypeUrl(arg)
			if err != nil {
				fmt.Println(err)
				continue
			}

			switch typeUrl {
			case utils.UrlTypeSpotifyTrack:
				fmt.Println("Downloading track:", arg)
			case utils.UrlTypeSpotifyAlbum:
				fmt.Println("Downloading album:", arg)
			case utils.UrlTypeSpotifyPlaylist:
				fmt.Println("Downloading playlist:", arg)
			default:
				fmt.Println("Unknown URL type:", arg)
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// downloadCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// downloadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
