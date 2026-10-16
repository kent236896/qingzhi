// Package response writes the JSON envelope used by ordinary APIs.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// ContextRequestID is the Gin context key for the current request id.
	ContextRequestID = "request_id"

	CodeOK               = 0
	CodeInvalidParams    = 40000
	CodeBadRequest       = 40001
	CodeUnauthorized     = 40100
	CodeForbidden        = 40300
	CodeNotFound         = 40400
	CodeConflict         = 40900
	CodeMethodNotAllowed = 40500
	CodePayloadTooLarge  = 41300
	CodeTooManyRequests  = 42900
	CodeInternal         = 50000
	CodeNotReady         = 50301
	CodeTimeout          = 50400
)

// Body is the ordinary JSON API envelope.
type Body struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id"`
}

// FieldError is a safe parameter validation message.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// RequestID returns the request id stored by middleware.
func RequestID(c *gin.Context) string {
	value, _ := c.Get(ContextRequestID)
	id, _ := value.(string)
	return id
}

// OK writes a successful envelope.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{
		Code:      CodeOK,
		Message:   "ok",
		Data:      data,
		RequestID: RequestID(c),
	})
}

// Fail writes an error envelope and stops the handler chain.
// message must be safe to show to a client.
func Fail(c *gin.Context, httpStatus, code int, message string, data any) {
	c.AbortWithStatusJSON(httpStatus, Body{
		Code:      code,
		Message:   message,
		Data:      data,
		RequestID: RequestID(c),
	})
}

// InvalidParams writes a parameter validation error.
func InvalidParams(c *gin.Context, fields []FieldError) {
	var data any
	if len(fields) > 0 {
		data = map[string]any{"fields": fields}
	}
	Fail(c, http.StatusBadRequest, CodeInvalidParams, "请求参数不合法", data)
}
