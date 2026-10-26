package handler

import (
	"errors"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/identity"
	"qingzhi/enterprise-api/internal/knowledge"
	"qingzhi/enterprise-api/internal/response"
	"qingzhi/enterprise-api/internal/weknora"
)

// Knowledge serves knowledge-base routes.
type Knowledge struct {
	Logger   *slog.Logger
	Database Pinger
	Service  *knowledge.Service
	Identity *identity.Service
}

func (h Knowledge) Register(group *gin.RouterGroup, enforce bool) {
	perm := func(permission string) gin.HandlerFunc {
		if !enforce {
			return func(c *gin.Context) { c.Next() }
		}
		return auth.RequirePermission(permission)
	}
	query := perm("knowledge:query")
	group.GET("/knowledge-bases/limits", query, h.limits)
	group.GET("/knowledge-bases/options", query, h.options)
	group.GET("/knowledge-bases", query, h.list)
	group.POST("/knowledge-bases", query, h.create)
	group.GET("/knowledge-bases/:id", query, h.get)
	group.PATCH("/knowledge-bases/:id", query, h.update)
	group.PUT("/knowledge-bases/:id/shares", query, h.share)
	group.DELETE("/knowledge-bases/:id", query, h.delete)
	group.GET("/knowledge-bases/:id/files", query, h.files)
	group.POST("/knowledge-bases/:id/files", query, h.upload)
	group.POST("/knowledge-bases/:id/files/zip", query, h.uploadZip)
	group.POST("/knowledge-bases/:id/files/sync", query, h.syncFiles)
	group.POST("/knowledge-bases/:id/files/delete", query, h.deleteFiles)
	group.POST("/knowledge-bases/:id/files/reparse", query, h.reparseFiles)
	group.GET("/knowledge-bases/:id/wiki/status", query, h.wikiStatus)
	group.GET("/knowledge-bases/:id/wiki/graph", query, h.wikiGraph)
	group.GET("/knowledge-bases/:id/wiki/stats", query, h.wikiStats)
	group.GET("/knowledge-bases/:id/wiki/folders", query, h.wikiFolders)
	group.GET("/knowledge-bases/:id/wiki/pages", query, h.wikiPages)
	group.GET("/knowledge-bases/:id/wiki/pages/*slug", query, h.wikiPage)
	group.GET("/knowledge-bases/:id/wiki/index", query, h.wikiIndex)
	group.GET("/knowledge-bases/:id/wiki/search", query, h.wikiSearch)
	group.GET("/knowledge-bases/:id/wiki/titles", query, h.wikiTitles)
	group.GET("/knowledge-bases/:id/wiki/revisions/*slug", query, h.wikiRevisions)
}

func (h Knowledge) limits(c *gin.Context) {
	if !h.available(c) {
		return
	}
	response.OK(c, h.Service.Limits())
}

func (h Knowledge) options(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok || !h.available(c) || h.Identity == nil {
		return
	}
	type choice struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		TenantID string `json:"tenant_id,omitempty"`
	}
	tenants := make([]choice, 0)
	if actor.CanQueryAll {
		page, err := h.Identity.ListTenants(c.Request.Context(), identity.ListQuery{Page: 1, PageSize: 100})
		if err != nil {
			h.write(c, nil, err)
			return
		}
		for _, item := range page.Items {
			tenants = append(tenants, choice{ID: item.ID, Name: item.Name})
		}
	} else {
		for _, id := range actor.TenantIDs {
			item, err := h.Identity.GetTenant(c.Request.Context(), id)
			if err == nil {
				tenants = append(tenants, choice{ID: item.ID, Name: item.Name})
			}
		}
	}
	orgs := make([]choice, 0)
	posts := make([]choice, 0)
	roles := make([]choice, 0)
	users := make([]choice, 0)
	if actor.CanManage {
		for _, tenant := range tenants {
			list, err := h.Identity.ListOrganizationRecords(c.Request.Context(), tenant.ID)
			if err == nil {
				for _, org := range list {
					orgs = append(orgs, choice{ID: org.ID, Name: org.Name, TenantID: org.TenantID})
				}
			}
			postPage, err := h.Identity.ListPosts(c.Request.Context(), tenant.ID, identity.ListQuery{Page: 1, PageSize: 100})
			if err == nil {
				for _, post := range postPage.Items {
					posts = append(posts, choice{ID: post.ID, Name: post.Name, TenantID: post.TenantID})
				}
			}
			rolePage, err := h.Identity.ListRoles(c.Request.Context(), tenant.ID, identity.ListQuery{Page: 1, PageSize: 100})
			if err == nil {
				for _, role := range rolePage.Items {
					roles = append(roles, choice{ID: role.ID, Name: role.Name, TenantID: role.TenantID})
				}
			}
		}
	}
	userPage, err := h.Identity.ListUsers(c.Request.Context(), identity.ListQuery{Page: 1, PageSize: 100})
	if err == nil {
		for _, user := range userPage.Items {
			users = append(users, choice{ID: user.ID, Name: user.Nickname + "（" + user.Username + "）"})
		}
	}
	models, modelErr := h.Service.Models(c.Request.Context())
	if modelErr != nil && h.Logger != nil {
		h.Logger.Warn("knowledge default models unavailable", slog.String("component", "knowledge"), slog.String("error", modelErr.Error()))
		models = knowledge.KnowledgeModels{}
	}
	response.OK(c, map[string]any{
		"tenants": tenants, "organizations": orgs, "posts": posts, "roles": roles, "users": users,
		"can_manage": actor.CanManage, "models": models,
	})
}

func (h Knowledge) list(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok || !h.available(c) {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	item, err := h.Service.List(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, c.Query("scope_type"), c.Query("keyword"), page, pageSize)
	h.write(c, item, err)
}

func (h Knowledge) create(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok || !h.available(c) {
		return
	}
	var body knowledge.CreateInput
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.Create(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, body)
	h.write(c, item, err)
}

func (h Knowledge) get(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	item, err := h.Service.Get(c.Request.Context(), actor, id)
	h.write(c, item, err)
}

func (h Knowledge) update(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	var body struct {
		Name         *string  `json:"name"`
		Description  *string  `json:"description"`
		TenantIDs    []string `json:"tenant_ids"`
		ShareUserIDs []string `json:"share_user_ids"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.Update(c.Request.Context(), actor, id, body.Name, body.Description, body.TenantIDs, body.ShareUserIDs, body.TenantIDs != nil, body.ShareUserIDs != nil)
	h.write(c, item, err)
}

func (h Knowledge) share(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	var body knowledge.ShareInput
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.Share(c.Request.Context(), actor, id, body)
	h.write(c, item, err)
}

func (h Knowledge) delete(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	err := h.Service.Delete(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id)
	h.write(c, nil, err)
}

func (h Knowledge) files(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	items, err := h.Service.ListFiles(c.Request.Context(), actor, id, c.Query("sync") == "1")
	h.write(c, items, err)
}

func (h Knowledge) upload(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "请选择要上传的文件", nil)
		return
	}
	headers := form.File["file"]
	limits := h.Service.Limits()
	if len(headers) == 0 || len(headers) > limits.MaxFiles {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "上传文件数量不正确", nil)
		return
	}
	files, cleanup := saveUploads(headers, limits)
	defer cleanup()
	results, err := h.Service.Upload(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id, files)
	h.write(c, results, err)
}

func (h Knowledge) uploadZip(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	header, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "请选择压缩包", nil)
		return
	}
	src, err := header.Open()
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "压缩包无法读取", nil)
		return
	}
	defer src.Close()
	path, _, err := knowledge.SaveLimited(src, int64(h.Service.Limits().ZipMaxTotalMB)<<20)
	if err != nil {
		h.write(c, nil, err)
		return
	}
	defer os.Remove(path)
	results, err := h.Service.UploadZip(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id, path)
	h.write(c, results, err)
}

func (h Knowledge) syncFiles(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	items, err := h.Service.ListFiles(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id, true)
	h.write(c, items, err)
}

func (h Knowledge) deleteFiles(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	var body struct {
		IDs []string `json:"ids"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	err := h.Service.DeleteFiles(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id, body.IDs)
	h.write(c, nil, err)
}

func (h Knowledge) reparseFiles(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	var body struct {
		IDs []string `json:"ids"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	err := h.Service.ReparseFiles(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id, body.IDs)
	h.write(c, nil, err)
}

func (h Knowledge) wikiStatus(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	item, err := h.Service.WikiStatus(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id)
	h.write(c, item, err)
}

func (h Knowledge) wikiGraph(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	query, err := graphQuery(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error(), nil)
		return
	}
	item, err := h.Service.WikiGraph(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id, query)
	h.write(c, item, err)
}

func (h Knowledge) wikiStats(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	item, err := h.Service.WikiStatsPublic(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id)
	h.write(c, item, err)
}

func (h Knowledge) wikiFolders(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	query := url.Values{}
	if parentID := c.Query("parent_id"); parentID != "" {
		query.Set("parent_id", parentID)
	}
	if pageTypes := c.Query("page_types"); pageTypes != "" {
		for _, item := range strings.Split(pageTypes, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if !safeToken.MatchString(item) {
				response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "页面类型不正确", nil)
				return
			}
		}
		query.Set("page_types", pageTypes)
	}
	item, err := h.Service.WikiFolders(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id, query)
	h.write(c, item, err)
}

func (h Knowledge) wikiPages(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	query, err := knowledge.BuildWikiPagesQuery(c.Request.URL.Query())
	if err != nil {
		h.write(c, nil, err)
		return
	}
	item, err := h.Service.WikiPages(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id, query)
	h.write(c, item, err)
}

func (h Knowledge) wikiPage(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	slug := strings.Trim(c.Param("slug"), "/")
	item, err := h.Service.WikiPage(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id, slug)
	h.write(c, item, err)
}

func (h Knowledge) wikiIndex(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	query, err := knowledge.BuildWikiIndexQuery(c.Request.URL.Query())
	if err != nil {
		h.write(c, nil, err)
		return
	}
	item, err := h.Service.WikiIndex(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id, query)
	h.write(c, item, err)
}

func (h Knowledge) wikiSearch(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	limit, err := knowledge.ClampWikiSearchLimit(c.Query("limit"))
	if err != nil {
		h.write(c, nil, err)
		return
	}
	item, err := h.Service.WikiSearch(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id, c.Query("q"), limit)
	h.write(c, item, err)
}

func (h Knowledge) wikiTitles(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	item, err := h.Service.WikiTitleMap(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id)
	h.write(c, item, err)
}

func (h Knowledge) wikiRevisions(c *gin.Context) {
	actor, id, ok := h.base(c)
	if !ok {
		return
	}
	query := url.Values{}
	if version := c.Query("version"); version != "" {
		query.Set("version", version)
	}
	if limit := c.Query("limit"); limit != "" {
		query.Set("limit", limit)
	}
	if offset := c.Query("offset"); offset != "" {
		query.Set("offset", offset)
	}
	slug := strings.Trim(c.Param("slug"), "/")
	item, err := h.Service.WikiRevisions(weknora.WithRequestID(c.Request.Context(), response.RequestID(c)), actor, id, slug, query)
	h.write(c, item, err)
}

func (h Knowledge) actor(c *gin.Context) (knowledge.Actor, bool) {
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
		return knowledge.Actor{}, false
	}
	actor := knowledge.Actor{
		UserID: principal.UserID, Username: principal.Username,
		CanManage: principal.HasPermission("knowledge:manage"), CanQueryAll: principal.HasPermission("system:tenant:query"),
	}
	if h.Identity != nil {
		ids, err := h.Identity.ListMembershipTenantIDs(c.Request.Context(), principal.UserID)
		if err != nil {
			h.write(c, nil, err)
			return knowledge.Actor{}, false
		}
		actor.TenantIDs = ids
	}
	if principal.TenantID != "" {
		found := false
		for _, id := range actor.TenantIDs {
			if id == principal.TenantID {
				found = true
			}
		}
		if !found {
			actor.TenantIDs = append(actor.TenantIDs, principal.TenantID)
		}
	}
	return actor, true
}

func (h Knowledge) base(c *gin.Context) (knowledge.Actor, string, bool) {
	actor, ok := h.actor(c)
	if !ok || !h.available(c) {
		return knowledge.Actor{}, "", false
	}
	id := c.Param("id")
	if !uuidPattern.MatchString(id) {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "请求参数不合法", nil)
		return knowledge.Actor{}, "", false
	}
	return actor, id, true
}

func (h Knowledge) available(c *gin.Context) bool {
	if h.Service == nil || h.Database == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return false
	}
	return true
}

func (h Knowledge) write(c *gin.Context, data any, err error) {
	if err == nil {
		response.OK(c, data)
		return
	}
	var domain *knowledge.Error
	if errors.As(err, &domain) {
		switch {
		case errors.Is(err, knowledge.ErrForbidden):
			response.Fail(c, http.StatusForbidden, response.CodeForbidden, domain.Message, nil)
		case errors.Is(err, knowledge.ErrNotFound):
			response.Fail(c, http.StatusNotFound, response.CodeNotFound, domain.Message, nil)
		case errors.Is(err, knowledge.ErrConflict):
			response.Fail(c, http.StatusConflict, response.CodeConflict, domain.Message, nil)
		case errors.Is(err, knowledge.ErrNotReady):
			response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, domain.Message, nil)
		default:
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		}
		return
	}
	var identityErr *identity.Error
	if errors.As(err, &identityErr) {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, identityErr.Message, nil)
		return
	}
	if h.Logger != nil {
		h.Logger.Error("knowledge request failed", slog.String("component", "knowledge"), slog.String("error", err.Error()))
	}
	response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误", nil)
}

func saveUploads(headers []*multipart.FileHeader, limits knowledge.Limits) ([]knowledge.LocalFile, func()) {
	files := make([]knowledge.LocalFile, 0, len(headers))
	paths := make([]string, 0, len(headers))
	cleanup := func() {
		for _, path := range paths {
			os.Remove(path)
		}
	}
	for _, header := range headers {
		if !knowledge.AcceptExt(header.Filename, limits.AllowedExts) {
			files = append(files, knowledge.LocalFile{Name: header.Filename})
			continue
		}
		src, err := header.Open()
		if err != nil {
			files = append(files, knowledge.LocalFile{Name: header.Filename})
			continue
		}
		path, size, err := knowledge.SaveLimited(src, int64(limits.MaxFileMB)<<20)
		src.Close()
		if err != nil {
			files = append(files, knowledge.LocalFile{Name: header.Filename})
			continue
		}
		paths = append(paths, path)
		files = append(files, knowledge.LocalFile{Name: header.Filename, Path: path, Size: size})
	}
	return files, cleanup
}

var safeToken = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

func graphQuery(c *gin.Context) (url.Values, error) {
	query := url.Values{}
	mode := c.DefaultQuery("mode", "overview")
	if mode != "overview" && mode != "ego" {
		return nil, errors.New("图谱模式不正确")
	}
	query.Set("mode", mode)
	if mode == "ego" {
		center := c.Query("center")
		if center == "" || strings.Contains(center, "..") {
			return nil, errors.New("展开邻居时必须指定中心页面")
		}
		query.Set("center", center)
	}
	if depth := c.Query("depth"); depth != "" {
		number, err := strconv.Atoi(depth)
		if err != nil || number < 1 || number > 3 {
			return nil, errors.New("图谱深度必须在 1 到 3 之间")
		}
		query.Set("depth", depth)
	}
	if types := c.Query("types"); types != "" {
		for _, item := range strings.Split(types, ",") {
			if !safeToken.MatchString(item) {
				return nil, errors.New("图谱类型不正确")
			}
		}
		query.Set("types", types)
	}
	limit := 500
	if raw := c.Query("limit"); raw != "" {
		number, err := strconv.Atoi(raw)
		if err != nil || number < 1 {
			return nil, errors.New("图谱数量不正确")
		}
		limit = number
	}
	if limit > 800 {
		limit = 800
	}
	query.Set("limit", strconv.Itoa(limit))
	return query, nil
}
