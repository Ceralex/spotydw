package spotify

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/Ceralex/spotydw/internal/utils"
	"github.com/Ceralex/spotydw/internal/youtube"
	_ "github.com/joho/godotenv/autoload"
	spotifyapi "github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

// Client represents a Spotify API client
type Client struct {
	api *spotifyapi.Client
}

// NewClient creates a new authenticated Spotify client
func NewClient(ctx context.Context) (*Client, error) {
	clientID := os.Getenv("SPOTIFY_ID")
	clientSecret := os.Getenv("SPOTIFY_SECRET")
	if clientID == "" || clientSecret == "" {
		return nil, errors.New("missing required environment variables: SPOTIFY_ID and/or SPOTIFY_SECRET")
	}

	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     spotifyauth.TokenURL,
	}

	token, err := config.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	httpClient := spotifyauth.New().Client(ctx, token)
	return &Client{api: spotifyapi.New(httpClient)}, nil
}

func downloadSimpleTrack(track *spotifyapi.SimpleTrack, album *spotifyapi.SimpleAlbum, outFolder string) error {
	artistNames := make([]string, 0, len(track.Artists))
	for _, artist := range track.Artists {
		artistNames = append(artistNames, artist.Name)
	}
	searchQuery := fmt.Sprintf("%s %s", track.Name, strings.Join(artistNames, " "))

	videos, err := youtube.SearchVideos(searchQuery)
	if err != nil {
		return fmt.Errorf("youtube search failed: %w", err)
	}

	if len(videos) == 0 {
		return fmt.Errorf("no YouTube videos found for: %s", searchQuery)
	}

	video := youtube.FindClosestVideo(track.TimeDuration(), videos)

	log.Printf("Downloading track: %s", track.Name)
	ytdlpCmd := exec.Command(
		"yt-dlp",
		"-x",
		"--no-embed-metadata",
		"-o", "-", // Output to stdout
		fmt.Sprintf("https://youtu.be/%s", video.ID),
	)

	albumArtists := make([]string, 0, len(album.Artists))
	for _, artist := range album.Artists {
		albumArtists = append(albumArtists, artist.Name)
	}
	artistsStr := strings.Join(artistNames, "; ")
	albumArtistsStr := strings.Join(albumArtists, "; ")

	fileName := fmt.Sprintf("%s%s.mp3", outFolder, utils.SanitizeFileName(track.Name))

	ffmpegCmd := exec.Command(
		"ffmpeg",
		"-i", "pipe:0", // Read from stdin
		"-f", "jpeg_pipe",
		"-i", album.Images[0].URL,
		"-metadata", fmt.Sprintf("title=%s", track.Name),
		"-metadata", fmt.Sprintf("artist=%s", artistsStr),
		"-metadata", fmt.Sprintf("album_artist=%s", albumArtistsStr),
		"-metadata", fmt.Sprintf("album=%s", album.Name),
		"-metadata", fmt.Sprintf("track=%d/%d", track.TrackNumber, album.TotalTracks),
		"-metadata", fmt.Sprintf("date=%s", album.ReleaseDate),
		"-map", "0",
		"-map", "1",
		"-c:v", "mjpeg",
		"-q:v", "2",
		"-metadata:s:v", "title='Album cover'",
		"-metadata:s:v", "comment='Cover (front)'",
		"-y",
		fileName,
	)

	// Pipe yt-dlp's output to ffmpeg's input
	ffmpegCmd.Stdin, err = ytdlpCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe for yt-dlp: %v", err)
	}
	ffmpegCmd.Stdout = nil
	ffmpegCmd.Stderr = os.Stderr

	// Start ffmpeg first
	if err := ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	// Run yt-dlp and pipe to ffmpeg
	if err := ytdlpCmd.Run(); err != nil {
		return fmt.Errorf("failed to run yt-dlp: %v", err)
	}

	// Wait for ffmpeg to finish
	if err := ffmpegCmd.Wait(); err != nil {
		return fmt.Errorf("failed to process with ffmpeg: %v", err)
	}

	return nil
}

func (c *Client) DownloadTrack(ctx context.Context, id string) error {
	track, err := c.api.GetTrack(ctx, spotifyapi.ID(id))

	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	return downloadSimpleTrack(&track.SimpleTrack, &track.Album, "")
}

func (c *Client) DownloadAlbum(ctx context.Context, id string) error {
	album, err := c.api.GetAlbum(ctx, spotifyapi.ID(id))
	if err != nil {
		return fmt.Errorf("failed to get album: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(len(album.Tracks.Tracks))

	for _, track := range album.Tracks.Tracks {
		os.Mkdir(album.Name, 0755)
		go func(track *spotifyapi.SimpleTrack) {
			defer wg.Done()
			if err := downloadSimpleTrack(track, &album.SimpleAlbum, album.Name+"/"); err != nil {
				log.Printf("Error downloading track: %v\n", err)
			}
		}(&track)
	}

	wg.Wait()
	return nil
}
