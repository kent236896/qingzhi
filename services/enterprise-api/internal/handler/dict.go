package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/dict"
	"qingzhi/enterprise-api/internal/response"
)

var dictTypePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,100}$`)

// Dict serves dictionary type and data routes.
type Dict struct {
	Logger   *slog.Logger
	Database Pinger
	Service  *dict.Service
	Timeout  time.Duration
}

// Register adds dictionary routes. They stay registered when the database is down.
func (h Dict) Register(group *gin.RouterGroup, enforce bool) {
	perm := func(permission string) gin.HandlerFunc {
		if !enforce {
			return func(c *gin.Context) { c.Next() }
		}
		return auth.RequirePermission(permission)
	}

	group.GET("/dict-types", perm("system:dict:query"), h.listTypes)
	group.POST("/dict-types", perm("system:dict:create"), h.createType)
	group.GET("/dict-types/:typeId", perm("system:dict:query"), h.getType)
	group.PATCH("/dict-types/:typeId", perm("system:dict:update"), h.updateType)
	group.DELETE("/dict-types/:typeId", perm("system:dict:delete"), h.deleteType)

	group.GET("/dict-data", perm("system:dict:query"), h.listData)
	group.POST("/dict-data", perm("system:dict:create"), h.createData)
	group.GET("/dict-data/:dataId", perm("system:dict:query"), h.getData)
	group.PATCH("/dict-data/:dataId", perm("system:dict:update"), h.updateData)
	group.DELETE("/dict-data/:dataId", perm("system:dict:delete"), h.deleteData)

	group.GET("/dicts/:dictType", perm("system:dict:query"), h.listEnabledData)
}

func (h Dict) listTypes(c *gin.Context) {
	if !h.available(c) {
		return
	}
	query, ok := dictListQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListTypes(c.Request.Context(), query)
	h.write(c, page, err)
}

func (h Dict) createType(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Status int    `json:"status"`
		Remark string `json:"remark"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.CreateType(c.Request.Context(), dict.CreateTypeInput{
		Name: body.Name, Type: body.Type, Status: body.Status, Remark: body.Remark, Actor: dictActor(c),
	})
	h.write(c, item, err)
}

func (h Dict) getType(c *gin.Context) {
	id, ok := pathID(c, "typeId")
	if !ok || !h.available(c) {
		return
	}
	item, err := h.Service.GetType(c.Request.Context(), id)
	h.write(c, item, err)
}

func (h Dict) updateType(c *gin.Context) {
	id, ok := pathID(c, "typeId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		Name   *string `json:"name"`
		Type   *string `json:"type"`
		Status *int    `json:"status"`
		Remark *string `json:"remark"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.UpdateType(c.Request.Context(), id, dict.UpdateTypeInput{
		Name: body.Name, Type: body.Type, Status: body.Status, Remark: body.Remark, Actor: dictActor(c),
	})
	h.write(c, item, err)
}

func (h Dict) deleteType(c *gin.Context) {
	id, ok := pathID(c, "typeId")
	if !ok || !h.available(c) {
		return
	}
	h.write(c, nil, h.Service.DeleteType(c.Request.Context(), id))
}

func (h Dict) listData(c *gin.Context) {
	if !h.available(c) {
		return
	}
	dictType := strings.TrimSpace(c.Query("dict_type"))
	if !dictTypePattern.MatchString(dictType) {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "字典类型不合法", nil)
		return
	}
	query, ok := dictListQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListData(c.Request.Context(), dictType, query)
	h.write(c, page, err)
}

func (h Dict) createData(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body struct {
		Sort      int    `json:"sort"`
		Label     string `json:"label"`
		Value     string `json:"value"`
		DictType  string `json:"dict_type"`
		Status    int    `json:"status"`
		ColorType string `json:"color_type"`
		CSSClass  string `json:"css_class"`
		Remark    string `json:"remark"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.CreateData(c.Request.Context(), dict.CreateDataInput{
		Sort: body.Sort, Label: body.Label, Value: body.Value, DictType: body.DictType,
		Status: body.Status, ColorType: body.ColorType, CSSClass: body.CSSClass, Remark: body.Remark,
		Actor: dictActor(c),
	})
	h.write(c, item, err)
}

func (h Dict) getData(c *gin.Context) {
	id, ok := pathID(c, "dataId")
	if !ok || !h.available(c) {
		return
	}
	item, err := h.Service.GetData(c.Request.Context(), id)
	h.write(c, item, err)
}

func (h Dict) updateData(c *gin.Context) {
	id, ok := pathID(c, "dataId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		Sort      *int    `json:"sort"`
		Label     *string `json:"label"`
		Value     *string `json:"value"`
		Status    *int    `json:"status"`
		ColorType *string `json:"color_type"`
		CSSClass  *string `json:"css_class"`
		Remark    *string `json:"remark"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.UpdateData(c.Request.Context(), id, dict.UpdateDataInput{
		Sort: body.Sort, Label: body.Label, Value: body.Value, Status: body.Status,
		ColorType: body.ColorType, CSSClass: body.CSSClass, Remark: body.Remark, Actor: dictActor(c),
	})
	h.write(c, item, err)
}

func (h Dict) deleteData(c *gin.Context) {
	id, ok := pathID(c, "dataId")
	if !ok || !h.available(c) {
		return
	}
	h.write(c, nil, h.Service.DeleteData(c.Request.Context(), id))
}

func (h Dict) listEnabledData(c *gin.Context) {
	dictType := c.Param("dictType")
	if !dictTypePattern.MatchString(dictType) || !h.available(c) {
		if !dictTypePattern.MatchString(dictType) {
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "字典类型不合法", nil)
		}
		return
	}
	items, err := h.Service.ListEnabledData(c.Request.Context(), dictType)
	h.write(c, items, err)
}

func dictActor(c *gin.Context) string {
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		return ""
	}
	return principal.Username
}

func (h Dict) available(c *gin.Context) bool {
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return false
	}
	if h.Database == nil {
		return true
	}
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()
	if err := h.Database.Ping(ctx); err != nil {
		if h.Logger != nil {
			h.Logger.Warn("dictionary database unavailable",
				slog.String("component", "app"),
				slog.String("error", err.Error()),
			)
		}
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return false
	}
	return true
}

func (h Dict) write(c *gin.Context, data any, err error) {
	if err == nil {
		response.OK(c, data)
		return
	}
	var domain *dict.Error
	if errors.As(err, &domain) {
		switch {
		case errors.Is(err, dict.ErrInvalid):
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		case errors.Is(err, dict.ErrNotFound):
			response.Fail(c, http.StatusNotFound, response.CodeNotFound, domain.Message, nil)
		case errors.Is(err, dict.ErrConflict):
			response.Fail(c, http.StatusConflict, response.CodeConflict, domain.Message, nil)
		default:
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		}
		return
	}
	if h.Logger != nil {
		h.Logger.Error("dictionary request failed",
			slog.String("component", "app"),
			slog.String("error", err.Error()),
		)
	}
	response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误", nil)
}

func dictListQuery(c *gin.Context) (dict.ListQuery, bool) {
	page, err := dictOptionalInt(c.Query("page"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "分页参数不合法", nil)
		return dict.ListQuery{}, false
	}
	pageSize, err := dictOptionalInt(c.Query("page_size"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "分页参数不合法", nil)
		return dict.ListQuery{}, false
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	if len(keyword) > 64 {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "请求参数不合法", nil)
		return dict.ListQuery{}, false
	}
	return dict.ListQuery{Page: page, PageSize: pageSize, Keyword: keyword}, true
}

func dictOptionalInt(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}
