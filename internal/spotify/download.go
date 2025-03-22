package spotify

import (
	"context"
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
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Download(URL *url.URL, concurrentN int) error {
	ctx := context.Background()
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

	guard := make(chan struct{}, concurrentN)
	wg := sync.WaitGroup{}

	for _, track := range album.Tracks.Tracks {
		wg.Add(1)
		guard <- struct{}{}

		go func(track spotifyapi.SimpleTrack) {
			defer func() { <-guard; wg.Done() }()
			if err := downloadSimpleTrack(&track, &album.SimpleAlbum, album.Name+"/"); err != nil {
				log.Printf("Error downloading track %s: %v", track.Name, err)
			}
		}(track)
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

	guard := make(chan struct{}, concurrentN)
	wg := sync.WaitGroup{}

	for _, track := range playlist.Tracks.Tracks {
		wg.Add(1)
		guard <- struct{}{}

		go func(track spotifyapi.PlaylistTrack) {
			defer func() { <-guard; wg.Done() }()
			if err := downloadSimpleTrack(&track.Track.SimpleTrack, &track.Track.Album, playlist.Name+"/"); err != nil {
				log.Printf("Error downloading track %s: %v", track.Track.Name, err)
			}
		}(track)
	}

	wg.Wait()
	return nil
}

func downloadSimpleTrack(track *spotifyapi.SimpleTrack, album *spotifyapi.SimpleAlbum, outFolder string) error {
	fmt.Printf("Downloading track: %s\n", track.Name)

	// Build the search query
	searchQuery := fmt.Sprintf("%s - %s", track.Name, joinArtists(track.Artists, ", "))
	videos, err := youtube.SearchVideos(searchQuery)
	if err != nil {
		return fmt.Errorf("YouTube search failed: %w", err)
	}
	if len(videos) == 0 {
		return fmt.Errorf("no YouTube videos found for: %s", searchQuery)
	}

	// Find the closest video
	video := youtube.FindClosestVideo(track.TimeDuration(), videos)

	// Prepare yt-dlp command
	ytdlpCmd := exec.Command(
		"yt-dlp",
		"-x",
		"--no-embed-metadata",
		"-o", "-", // Output to stdout
		fmt.Sprintf("https://youtu.be/%s", video.ID),
	)

	// Prepare ffmpeg command
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
		"-metadata:s:v", "title=Album cover",
		"-metadata:s:v", "comment=Cover (front)",
		"-y",
		fileName,
	)

	// Pipe yt-dlp's output to ffmpeg's input
	if ffmpegCmd.Stdin, err = ytdlpCmd.StdoutPipe(); err != nil {
		return fmt.Errorf("failed to create stdout pipe for yt-dlp: %w", err)
	}

	// Start ffmpeg and yt-dlp commands
	if err := ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}
	if err := ytdlpCmd.Run(); err != nil {
		return fmt.Errorf("failed to run yt-dlp: %w", err)
	}
	if err := ffmpegCmd.Wait(); err != nil {
		return fmt.Errorf("failed to process with ffmpeg: %w", err)
	}

	return nil
}
