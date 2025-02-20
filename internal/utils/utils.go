package utils

import (
	"net/url"
	"regexp"
	"strings"
)

func SanitizeFileName(name string) string {
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)

	sanitized := invalidChars.ReplaceAllString(name, " ")

	sanitized = strings.Trim(sanitized, " .")

	return sanitized
}

// IsUrl checks if a string is a valid URL
func IsUrl(str string) bool {
	_, err := url.ParseRequestURI(str)
	return err == nil
}
