package spotify

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Ceralex/spotydw/internal/youtube"
	_ "github.com/joho/godotenv/autoload"
	spotifyapi "github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

func ExtractId(url string) string {
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

func DownloadTrack(id string) error {
	ctx := context.Background()
	config := &clientcredentials.Config{
		ClientID:     os.Getenv("SPOTIFY_ID"),
		ClientSecret: os.Getenv("SPOTIFY_SECRET"),
		TokenURL:     spotifyauth.TokenURL,
	}

	token, err := config.Token(ctx)
	if err != nil {
		return err
	}

	httpClient := spotifyauth.New().Client(ctx, token)
	client := spotifyapi.New(httpClient)
	results, err := client.GetTrack(ctx, spotifyapi.ID(id))
	if err != nil {
		return err
	}

	track := results.SimpleTrack
	var values []string
	for _, name := range track.Artists {
		values = append(values, name.Name)
	}
	artists := strings.Join(values, ", ")

	videos, err := youtube.SearchVideos(track.Name + " " + artists)
	if err != nil {
		return err
	}
	if len(videos) == 0 {
		fmt.Println("No videos found for", track.Name)
		return nil
	}

	video := youtube.FindClosestVideo(track.TimeDuration(), videos)

	fmt.Printf("Downloading %#v\n", video)
	return nil
}
