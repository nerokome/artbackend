package utils

import (
	"regexp"
	"strings"
)

// Slugify converts a string into a URL-safe slug
func Slugify(input string) string {
	s := strings.TrimSpace(input)
	s = strings.ToLower(s)
	// Replace non-breaking spaces with normal spaces
	s = strings.ReplaceAll(s, "\u00A0", " ")
	// Collapse whitespace
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, "-")
	// Remove non-alphanumeric or dash
	s = regexp.MustCompile(`[^a-z0-9\-]`).ReplaceAllString(s, "")
	return s
}
