package group

import (
	"regexp"
	"strings"
)

func CreateSlug(title string) string {
	// lowercase & trim spaces
	slug := strings.ToLower(strings.TrimSpace(title))

	// replace any non-alphanumeric character with hyphen
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug = re.ReplaceAllString(slug, "-")

	// remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}
