package spotify

import (
	"context"
	"errors"
	"fmt"
	"os"

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
