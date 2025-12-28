package utils

import (
	"strings"
)

func GenerateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	return slug
}

func GenerateSEO(title string, imageURL string) map[string]interface{} {
	return map[string]interface{}{
		"title":       title + " | Artfolio",
		"description": "Explore the artwork titled '" + title + "' on Artfolio. View engagement, style, and performance insights.",
		"image":       imageURL,
		"keywords": []string{
			"artfolio",
			"digital art",
			"title:" + title,
			"creative portfolio",
		},
	}
}
