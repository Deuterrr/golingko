// Package slug provides deterministic, URL-friendly kebab-case slug generation for entity names.
package slug

import (
	"regexp"
	"strings"
)

var (
	nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9]+`)
	multipleHyphenRegex  = regexp.MustCompile(`-+`)
)

// Generates a deterministic, URL-friendly kebab-case slug from an entity name.
func Generate(name string) string {
	s := strings.TrimSpace(strings.ToLower(name))
	if s == "" {
		return ""
	}

	// Remove periods from abbreviations (e.g. "M.H." -> "mh")
	s = strings.ReplaceAll(s, ".", "")
	// Transliterate common transit delimiters ('/' or '&' or '+') with spaces
	s = strings.ReplaceAll(s, "/", " ")
	s = strings.ReplaceAll(s, "&", " ")
	s = strings.ReplaceAll(s, "+", " ")

	s = nonAlphanumericRegex.ReplaceAllString(s, "-")
	s = multipleHyphenRegex.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// GenerateSlug is an alias for Generate to maintain API compatibility.
func GenerateSlug(name string) string {
	return Generate(name)
}
