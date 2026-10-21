package knowledge

import (
	"net/url"
	"strings"
	"testing"
)

func TestEscapeWikiSearch(t *testing.T) {
	got, err := EscapeWikiSearch("a+b(c)")
	if err != nil {
		t.Fatal(err)
	}
	if got != `a\+b\(c\)` {
		t.Fatalf("got %q", got)
	}
	if _, err := EscapeWikiSearch(""); err == nil {
		t.Fatal("expected empty search to fail")
	}
	long := strings.Repeat("字", 101)
	if _, err := EscapeWikiSearch(long); err == nil {
		t.Fatal("expected overlong search to fail")
	}
}

func TestClampWikiPageSize(t *testing.T) {
	got, err := ClampWikiPageSize("80", 20)
	if err != nil || got != 50 {
		t.Fatalf("got %d err=%v", got, err)
	}
	got, err = ClampWikiPageSize("", 20)
	if err != nil || got != 20 {
		t.Fatalf("got %d err=%v", got, err)
	}
	if _, err := ClampWikiPageSize("0", 20); err == nil {
		t.Fatal("expected invalid page size")
	}
}

func TestBuildWikiPagesQuery(t *testing.T) {
	query, err := BuildWikiPagesQuery(url.Values{
		"page":       {"1"},
		"page_size":  {"100"},
		"page_type":  {"entity"},
		"sort_by":    {"updated_at"},
		"sort_order": {"desc"},
		"query":      {"lora+"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if query.Get("page_size") != "50" {
		t.Fatalf("page_size=%s", query.Get("page_size"))
	}
	if query.Get("query") != `lora\+` {
		t.Fatalf("query=%s", query.Get("query"))
	}
	if _, err := BuildWikiPagesQuery(url.Values{"sort_by": {"hack"}}); err == nil {
		t.Fatal("expected bad sort_by to fail")
	}
}

func TestBuildWikiIndexQuery(t *testing.T) {
	query, err := BuildWikiIndexQuery(url.Values{"types": {"entity"}, "limit": {"500"}})
	if err != nil {
		t.Fatal(err)
	}
	if query.Get("limit") != "200" {
		t.Fatalf("limit=%s", query.Get("limit"))
	}
}
