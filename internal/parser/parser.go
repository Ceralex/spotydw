package parser

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/Ceralex/spotydw/internal/utils"
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
	if !utils.IsUrl(str) {
		return UrlTypeUnknown, errors.New("invalid URL")
	}

	parsedUrl, err := url.Parse(str)
	if err != nil {
		return UrlTypeUnknown, fmt.Errorf("failed to parse URL: %w", err)
	}

	if !slices.Contains(supportedHosts, parsedUrl.Host) {
		return UrlTypeUnknown, errors.New("unsupported host")
	}

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

func ExtractSpotifyID(url string) string {
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
