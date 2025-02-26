package spotify

import (
	"errors"
	"net/url"
	"strings"

	spotifyapi "github.com/zmb3/spotify/v2"
)

type ResourceType int

const (
	Track ResourceType = iota
	Album
	Playlist
)

type Resource struct {
	ID   string
	Type ResourceType
}

func getResource(URL *url.URL) (Resource, error) {
	path := URL.Path
	if strings.Contains(path, "track") {
		return Resource{ID: extractID(URL), Type: Track}, nil
	} else if strings.Contains(path, "album") {
		return Resource{ID: extractID(URL), Type: Album}, nil
	} else if strings.Contains(path, "playlist") {
		return Resource{ID: extractID(URL), Type: Playlist}, nil
	}

	return Resource{}, errors.New("could not determine valid resource type (track|album|playlist)")
}

func extractID(URL *url.URL) string {
	path := URL.Path
	return path[strings.LastIndex(path, "/")+1:]
}

func joinArtists(artists []spotifyapi.SimpleArtist, separator string) string {
	artistNames := make([]string, 0, len(artists))
	for _, artist := range artists {
		artistNames = append(artistNames, artist.Name)
	}
	return strings.Join(artistNames, separator)
}
