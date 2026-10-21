package weknora

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxWikiSlugRunes = 200

var controlChar = regexp.MustCompile(`[\x00-\x1f\x7f]`)

// SanitizeWikiSlug rejects traversal, backslashes, control characters, and overlong values.
func SanitizeWikiSlug(slug string) (string, bool) {
	slug = strings.Trim(strings.TrimSpace(slug), "/")
	if slug == "" || utf8.RuneCountInString(slug) > maxWikiSlugRunes {
		return "", false
	}
	if strings.Contains(slug, "..") || strings.Contains(slug, `\`) || strings.Contains(slug, "\\") {
		return "", false
	}
	if controlChar.MatchString(slug) {
		return "", false
	}
	for _, part := range strings.Split(slug, "/") {
		if part == "" || part == "." || part == ".." {
			return "", false
		}
	}
	return slug, true
}
