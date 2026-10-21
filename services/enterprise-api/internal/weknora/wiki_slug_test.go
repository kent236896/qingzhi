package weknora

import "testing"

func TestSanitizeWikiSlug(t *testing.T) {
	okCases := []string{"entity/lo-ra", "index", "summary/abc-123", "concept/fine-tuning"}
	for _, slug := range okCases {
		got, ok := SanitizeWikiSlug(slug)
		if !ok || got != slug {
			t.Fatalf("expected %q to be valid, got %q ok=%v", slug, got, ok)
		}
	}
	badCases := []string{"", "../etc", "a/../b", `entity\lo-ra`, "a\nb", stringsRepeat("x", 201)}
	for _, slug := range badCases {
		if _, ok := SanitizeWikiSlug(slug); ok {
			t.Fatalf("expected %q to be rejected", slug)
		}
	}
}

func stringsRepeat(value string, n int) string {
	out := make([]byte, 0, n*len(value))
	for i := 0; i < n; i++ {
		out = append(out, value...)
	}
	return string(out)
}
