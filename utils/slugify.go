package utils

import (
	"regexp"
	"strings"
)

func Slugify(input string) string {
	s := strings.ToLower(input)
	// Replace non-alphanumeric characters with space
	reg := regexp.MustCompile(`[^\w\s-]`)
	s = reg.ReplaceAllString(s, "")
	// Replace spaces and underscores with dash
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	// Remove multiple dashes
	reg2 := regexp.MustCompile(`-+`)
	s = reg2.ReplaceAllString(s, "-")
	return s
}
