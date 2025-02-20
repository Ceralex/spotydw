package utils

import (
	"regexp"
	"strings"
)

func SanitizeFileName(name string) string {
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)

	sanitized := invalidChars.ReplaceAllString(name, " ")

	sanitized = strings.Trim(sanitized, " .")

	return sanitized
}
