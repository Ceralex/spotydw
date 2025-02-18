package utils

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
)

type UrlType int

const (
	UrlTypeUnknown UrlType = iota // Unknown type
	UrlTypeSpotifyTrack
	UrlTypeSpotifyAlbum
	UrlTypeSpotifyPlaylist
)

// List of supported hosts
var supportedHosts = []string{"open.spotify.com"}

var (
	spotifyTrackRegex    = regexp.MustCompile(`^/track/[a-zA-Z0-9]+`)
	spotifyAlbumRegex    = regexp.MustCompile(`^/album/[a-zA-Z0-9]+`)
	spotifyPlaylistRegex = regexp.MustCompile(`^/playlist/[a-zA-Z0-9]+`)
)

// GetTypeUrl determines the type of the URL based on its host and path
func GetTypeUrl(str string) (UrlType, error) {
	if !IsUrl(str) {
		return UrlTypeUnknown, errors.New("invalid URL")
	}

	parsedUrl, err := url.Parse(str)
	if err != nil {
		return UrlTypeUnknown, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Check if the host is supported
	if !slices.Contains(supportedHosts, parsedUrl.Host) {
		return UrlTypeUnknown, errors.New("unsupported host")
	}

	// Determine the URL type based on the host and path
	switch parsedUrl.Host {
	case "open.spotify.com":
		if spotifyPlaylistRegex.MatchString(parsedUrl.Path) {
			return UrlTypeSpotifyPlaylist, nil
		} else if spotifyTrackRegex.MatchString(parsedUrl.Path) {
			return UrlTypeSpotifyTrack, nil
		} else if spotifyAlbumRegex.MatchString(parsedUrl.Path) {
			return UrlTypeSpotifyAlbum, nil
		}
	}

	// If no specific type is matched, return unknown
	return UrlTypeUnknown, nil
}

// IsUrl checks if a string is a valid URL
func IsUrl(str string) bool {
	_, err := url.ParseRequestURI(str)
	return err == nil
}
