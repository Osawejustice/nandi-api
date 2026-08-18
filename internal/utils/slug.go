package utils

import (
	"regexp"
	"strings"
)

var (
	nonSlug   = regexp.MustCompile(`[^a-z0-9]+`)
	multiDash = regexp.MustCompile(`-+`)
)

func Slugify(value string) string {
	s := strings.ToLower(strings.TrimSpace(value))
	s = nonSlug.ReplaceAllString(s, "-")
	s = multiDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "org"
	}
	if len(s) > 80 {
		s = strings.Trim(s[:80], "-")
	}
	return s
}
