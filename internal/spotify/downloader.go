package spotify

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/Ceralex/spotydw/internal/utils"
	"github.com/Ceralex/spotydw/internal/youtube"
	_ "github.com/joho/godotenv/autoload"
	spotifyapi "github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

type Client struct {
	api *spotifyapi.Client
}

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

func Download(ctx context.Context, URL *url.URL, concurrentN int) error {
	client, err := NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create Spotify client: %v", err)
	}

	resource, err := getResource(URL)
	if err != nil {
		return err
	}

	switch resource.Type {
	case Track:
		return client.DownloadTrack(ctx, resource.ID)
	case Album:
		return client.DownloadAlbum(ctx, resource.ID, concurrentN)
	case Playlist:
		return client.DownloadPlaylist(ctx, resource.ID, concurrentN)
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

func (c *Client) DownloadAlbum(ctx context.Context, id string, concurrentN int) error {
	album, err := c.api.GetAlbum(ctx, spotifyapi.ID(id))
	if err != nil {
		return fmt.Errorf("failed to get album: %w", err)
	}

	if err := os.Mkdir(album.Name, 0755); err != nil {
		return fmt.Errorf("failed to create album directory: %w", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(album.Tracks.Tracks))

	guard := make(chan struct{}, concurrentN)

	for _, track := range album.Tracks.Tracks {
		guard <- struct{}{}

		go func(track *spotifyapi.SimpleTrack, album *spotifyapi.SimpleAlbum) {
			defer func() {
				wg.Done()
				<-guard
			}()

			if err := downloadSimpleTrack(track, album, album.Name+"/"); err != nil {
				log.Printf("Error downloading track: %v\n", err)
			}
		}(&track, &album.SimpleAlbum)
	}

	wg.Wait()
	return nil
}

func (c *Client) DownloadPlaylist(ctx context.Context, id string, concurrentN int) error {
	playlist, err := c.api.GetPlaylist(ctx, spotifyapi.ID(id))
	if err != nil {
		return fmt.Errorf("failed to get playlist: %w", err)
	}

	if err := os.Mkdir(playlist.Name, 0755); err != nil {
		return fmt.Errorf("failed to create playlist directory: %w", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(playlist.Tracks.Tracks))

	guard := make(chan struct{}, concurrentN)

	for _, track := range playlist.Tracks.Tracks {
		guard <- struct{}{}

		go func(track *spotifyapi.PlaylistTrack, playlist *spotifyapi.FullPlaylist) {
			defer func() {
				wg.Done()
				<-guard
			}()

			if err := downloadSimpleTrack(&track.Track.SimpleTrack, &track.Track.Album, playlist.Name+"/"); err != nil {
				log.Printf("Error downloading track: %v\n", err)
			}
		}(&track, playlist)
	}

	return nil
}

func downloadSimpleTrack(track *spotifyapi.SimpleTrack, album *spotifyapi.SimpleAlbum, outFolder string) error {
	fmt.Printf("Downloading track: %s\n", track.Name)

	searchQuery := fmt.Sprintf("%s - %s", track.Name, joinArtists(track.Artists, ", "))

	videos, err := youtube.SearchVideos(searchQuery)
	if err != nil {
		return fmt.Errorf("youtube search failed: %w", err)
	}

	if len(videos) == 0 {
		return fmt.Errorf("no YouTube videos found for: %s", searchQuery)
	}

	video := youtube.FindClosestVideo(track.TimeDuration(), videos)

	ytdlpCmd := exec.Command(
		"yt-dlp",
		"-x",
		"--no-embed-metadata",
		"-o", "-", // Output to stdout
		fmt.Sprintf("https://youtu.be/%s", video.ID),
	)

	fileName := filepath.Join(outFolder, utils.SanitizeFileName(track.Name)+".mp3")

	ffmpegCmd := exec.Command(
		"ffmpeg",
		"-i", "pipe:0", // Read from stdin
		"-f", "jpeg_pipe",
		"-i", album.Images[0].URL,
		"-metadata", fmt.Sprintf("title=%s", track.Name),
		"-metadata", fmt.Sprintf("artist=%s", joinArtists(track.Artists, ";")),
		"-metadata", fmt.Sprintf("album_artist=%s", joinArtists(album.Artists, ";")),
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
