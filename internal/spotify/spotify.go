package spotify

import (
	"strings"
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
