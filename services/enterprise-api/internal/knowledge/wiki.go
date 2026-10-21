package knowledge

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"qingzhi/enterprise-api/internal/weknora"
)

const (
	wikiSearchMaxRunes = 100
	wikiPageSizeMax    = 50
	wikiIndexLimitMax  = 200
	wikiSearchLimitMax = 20
)

var wikiSortBy = map[string]struct{}{
	"title":      {},
	"updated_at": {},
	"created_at": {},
}

// EscapeWikiSearch escapes a POSIX regex search term and enforces length.
func EscapeWikiSearch(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", invalid("请输入搜索词")
	}
	if utf8.RuneCountInString(raw) > wikiSearchMaxRunes {
		return "", invalid("搜索词不能超过 100 个字符")
	}
	return regexp.QuoteMeta(raw), nil
}

// ClampWikiPageSize caps page_size at 50.
func ClampWikiPageSize(raw string, fallback int) (int, error) {
	if raw == "" {
		if fallback <= 0 {
			fallback = 20
		}
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, invalid("分页大小不正确")
	}
	if n > wikiPageSizeMax {
		n = wikiPageSizeMax
	}
	return n, nil
}

func clampWikiLimit(raw string, fallback, max int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, invalid("数量不正确")
	}
	if n > max {
		n = max
	}
	return n, nil
}

// ClampWikiSearchLimit caps search result count at 20.
func ClampWikiSearchLimit(raw string) (int, error) {
	return clampWikiLimit(raw, 10, wikiSearchLimitMax)
}

func (s *Service) WikiFolders(ctx context.Context, actor Actor, id string, query url.Values) (weknora.WikiFolderList, error) {
	item, err := s.wikiReadyBase(ctx, actor, id)
	if err != nil {
		return weknora.WikiFolderList{}, err
	}
	return s.remote.WikiFolders(ctx, item.RemoteID, query)
}

func (s *Service) WikiPages(ctx context.Context, actor Actor, id string, query url.Values) (weknora.WikiPageList, error) {
	item, err := s.wikiReadyBase(ctx, actor, id)
	if err != nil {
		return weknora.WikiPageList{}, err
	}
	return s.remote.WikiPages(ctx, item.RemoteID, query)
}

func (s *Service) WikiPage(ctx context.Context, actor Actor, id, slug string) (weknora.WikiPage, error) {
	item, err := s.wikiReadyBase(ctx, actor, id)
	if err != nil {
		return weknora.WikiPage{}, err
	}
	clean, ok := weknora.SanitizeWikiSlug(slug)
	if !ok {
		return weknora.WikiPage{}, invalid("页面地址不正确")
	}
	page, err := s.remote.WikiPage(ctx, item.RemoteID, clean)
	if errors.Is(err, weknora.ErrNotFound) {
		return weknora.WikiPage{}, notFound("Wiki 页面不存在")
	}
	return page, err
}

func (s *Service) WikiIndex(ctx context.Context, actor Actor, id string, query url.Values) (weknora.WikiIndex, error) {
	item, err := s.wikiReadyBase(ctx, actor, id)
	if err != nil {
		return weknora.WikiIndex{}, err
	}
	return s.remote.WikiIndex(ctx, item.RemoteID, query)
}

func (s *Service) WikiSearch(ctx context.Context, actor Actor, id, keyword string, limit int) (weknora.WikiPageList, error) {
	item, err := s.wikiReadyBase(ctx, actor, id)
	if err != nil {
		return weknora.WikiPageList{}, err
	}
	escaped, err := EscapeWikiSearch(keyword)
	if err != nil {
		return weknora.WikiPageList{}, err
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > wikiSearchLimitMax {
		limit = wikiSearchLimitMax
	}
	query := url.Values{}
	query.Set("q", escaped)
	query.Set("limit", strconv.Itoa(limit))
	return s.remote.WikiSearch(ctx, item.RemoteID, query)
}

func (s *Service) WikiStatsPublic(ctx context.Context, actor Actor, id string) (weknora.WikiStats, error) {
	item, err := s.wikiReadyBase(ctx, actor, id)
	if err != nil {
		return weknora.WikiStats{}, err
	}
	return s.remote.WikiStats(ctx, item.RemoteID)
}

func (s *Service) WikiRevisions(ctx context.Context, actor Actor, id, slug string, query url.Values) (weknora.WikiRevisionList, error) {
	item, err := s.wikiReadyBase(ctx, actor, id)
	if err != nil {
		return weknora.WikiRevisionList{}, err
	}
	clean, ok := weknora.SanitizeWikiSlug(slug)
	if !ok {
		return weknora.WikiRevisionList{}, invalid("页面地址不正确")
	}
	list, err := s.remote.WikiRevisions(ctx, item.RemoteID, clean, query)
	if errors.Is(err, weknora.ErrNotFound) {
		return weknora.WikiRevisionList{}, notFound("Wiki 页面不存在")
	}
	return list, err
}

// WikiTitleMap builds a lightweight slug→title map from the index (low latency for reverse links).
func (s *Service) WikiTitleMap(ctx context.Context, actor Actor, id string) ([]weknora.WikiTitle, error) {
	item, err := s.wikiReadyBase(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	titles := map[string]weknora.WikiTitle{}
	types := []string{"summary", "entity", "concept", "synthesis", "comparison", "index"}
	for _, pageType := range types {
		cursor := ""
		for round := 0; round < 40; round++ {
			query := url.Values{}
			query.Set("types", pageType)
			query.Set("limit", strconv.Itoa(wikiIndexLimitMax))
			if cursor != "" {
				query.Set("cursor", cursor)
			}
			index, err := s.remote.WikiIndex(ctx, item.RemoteID, query)
			if err != nil {
				return nil, err
			}
			var next string
			for _, group := range index.Groups {
				if group.Type != pageType {
					continue
				}
				for _, entry := range group.Items {
					titles[entry.Slug] = weknora.WikiTitle{Slug: entry.Slug, Title: entry.Title, PageType: pageType}
				}
				next = group.NextCursor
			}
			if next == "" {
				break
			}
			cursor = next
		}
	}
	out := make([]weknora.WikiTitle, 0, len(titles))
	for _, item := range titles {
		out = append(out, item)
	}
	return out, nil
}

// BuildWikiPagesQuery validates and caps list query params for the pages proxy.
func BuildWikiPagesQuery(cQuery url.Values) (url.Values, error) {
	query := url.Values{}
	if page := cQuery.Get("page"); page != "" {
		n, err := strconv.Atoi(page)
		if err != nil || n < 1 {
			return nil, invalid("页码不正确")
		}
		query.Set("page", strconv.Itoa(n))
	}
	size, err := ClampWikiPageSize(cQuery.Get("page_size"), 20)
	if err != nil {
		return nil, err
	}
	query.Set("page_size", strconv.Itoa(size))
	if folderID := strings.TrimSpace(cQuery.Get("folder_id")); folderID != "" {
		query.Set("folder_id", folderID)
	}
	if pageType := cQuery.Get("page_type"); pageType != "" {
		if !safeWikiToken(pageType) {
			return nil, invalid("页面类型不正确")
		}
		query.Set("page_type", pageType)
	}
	if keyword := cQuery.Get("query"); keyword != "" {
		escaped, err := EscapeWikiSearch(keyword)
		if err != nil {
			return nil, err
		}
		query.Set("query", escaped)
	}
	if sortBy := cQuery.Get("sort_by"); sortBy != "" {
		if _, ok := wikiSortBy[sortBy]; !ok {
			return nil, invalid("排序字段不正确")
		}
		query.Set("sort_by", sortBy)
	}
	if sortOrder := cQuery.Get("sort_order"); sortOrder != "" {
		if sortOrder != "asc" && sortOrder != "desc" {
			return nil, invalid("排序方向不正确")
		}
		query.Set("sort_order", sortOrder)
	}
	return query, nil
}

// BuildWikiIndexQuery validates index query params.
func BuildWikiIndexQuery(cQuery url.Values) (url.Values, error) {
	query := url.Values{}
	if types := cQuery.Get("types"); types != "" {
		for _, item := range strings.Split(types, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if !safeWikiToken(item) {
				return nil, invalid("索引类型不正确")
			}
		}
		query.Set("types", types)
	}
	limit, err := clampWikiLimit(cQuery.Get("limit"), 50, wikiIndexLimitMax)
	if err != nil {
		return nil, err
	}
	query.Set("limit", strconv.Itoa(limit))
	if cursor := cQuery.Get("cursor"); cursor != "" {
		query.Set("cursor", cursor)
	}
	return query, nil
}

func safeWikiToken(value string) bool {
	return regexp.MustCompile(`^[a-z0-9_-]{1,32}$`).MatchString(value)
}
