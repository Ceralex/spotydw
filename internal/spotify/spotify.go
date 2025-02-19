package spotify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

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

func ExtractID(url string) string {
	// Find the last "/" in the URL
	idStart := strings.LastIndex(url, "/")
	if idStart < 0 {
		return ""
	}

	// Extract the ID part (before "?si=" if present)
	id := url[idStart+1:]
	if idx := strings.Index(id, "?"); idx != -1 {
		id = id[:idx]
	}

	return id
}

func (c *Client) DownloadTrack(ctx context.Context, id string) error {
	track, err := c.api.GetTrack(ctx, spotifyapi.ID(id))
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

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

	fmt.Println("Downloading track:", track.Name)
	ytdlpCmd := exec.Command(
		"yt-dlp",
		"-x",
		"--no-embed-metadata",
		"-o", "-", // Output to stdout
		fmt.Sprintf("https://youtu.be/%s", video.ID),
	)

	albumArtists := make([]string, 0, len(track.Album.Artists))
	for _, artist := range track.Album.Artists {
		albumArtists = append(albumArtists, artist.Name)
	}
	artistsStr := strings.Join(artistNames, "; ")
	albumArtistsStr := strings.Join(albumArtists, "; ")
	ffmpegCmd := exec.Command(
		"ffmpeg",
		"-i", "pipe:0", // Read from stdin
		"-f", "jpeg_pipe",
		"-i", track.Album.Images[0].URL,
		"-metadata", fmt.Sprintf("title=%s", track.Name),
		"-metadata", fmt.Sprintf("artist=%s", artistsStr),
		"-metadata", fmt.Sprintf("album_artist=%s", albumArtistsStr),
		"-metadata", fmt.Sprintf("album=%s", track.Album.Name),
		"-metadata", fmt.Sprintf("track=%d/%d", track.TrackNumber, track.Album.TotalTracks),
		"-metadata", fmt.Sprintf("date=%s", track.Album.ReleaseDate),
		"-map", "0",
		"-map", "1",
		"-c:v", "mjpeg",
		"-q:v", "2",
		"-metadata:s:v", "title='Album cover'",
		"-metadata:s:v", "comment='Cover (front)'",
		"-y",
		fmt.Sprintf("%s.mp3", track.Name),
	)

	// Pipe yt-dlp's output to ffmpeg's input
	ffmpegCmd.Stdin, _ = ytdlpCmd.StdoutPipe()
	ffmpegCmd.Stdout = nil
	ffmpegCmd.Stderr = nil

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
