package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestOKEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(ContextRequestID, "req-success")

	OK(ctx, map[string]string{"status": "live"})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	body := decode(t, recorder.Body.Bytes())
	if body["code"] != float64(0) || body["message"] != "ok" || body["request_id"] != "req-success" {
		t.Fatalf("body = %#v", body)
	}
	data, _ := body["data"].(map[string]any)
	if data["status"] != "live" {
		t.Fatalf("data = %#v", body["data"])
	}
}

func TestErrorEnvelopeHidesInternals(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(ContextRequestID, "req-error")

	Fail(ctx, http.StatusInternalServerError, CodeInternal, "服务器内部错误", nil)

	bodyText := recorder.Body.String()
	if strings.Contains(bodyText, "goroutine") || strings.Contains(bodyText, "password") {
		t.Fatalf("response leaked diagnostics: %s", bodyText)
	}
	body := decode(t, recorder.Body.Bytes())
	if recorder.Code != http.StatusInternalServerError || body["code"] != float64(CodeInternal) {
		t.Fatalf("status %d body %#v", recorder.Code, body)
	}
	if body["message"] != "服务器内部错误" || body["request_id"] != "req-error" || body["data"] != nil {
		t.Fatalf("body = %#v", body)
	}
}

func TestBindJSONValidation(t *testing.T) {
	type payload struct {
		Name string `json:"name" binding:"required"`
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(ContextRequestID, "req-validate")
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/example", strings.NewReader(`{}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	if BindJSON(ctx, &payload{}) {
		t.Fatal("expected validation failure")
	}
	body := decode(t, recorder.Body.Bytes())
	if recorder.Code != http.StatusBadRequest || body["code"] != float64(CodeInvalidParams) {
		t.Fatalf("status %d body %#v", recorder.Code, body)
	}
	if body["message"] != "请求参数不合法" {
		t.Fatalf("message = %#v", body["message"])
	}
	data, _ := body["data"].(map[string]any)
	fields, _ := data["fields"].([]any)
	if len(fields) != 1 {
		t.Fatalf("fields = %#v", data)
	}
	field, _ := fields[0].(map[string]any)
	if field["field"] != "name" || field["message"] != "name为必填项" {
		t.Fatalf("field = %#v", field)
	}
}

func TestBindJSONRejectsMalformedBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(ContextRequestID, "req-json")
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/example", strings.NewReader(`{`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	if BindJSON(ctx, &map[string]string{}) {
		t.Fatal("expected json failure")
	}
	body := decode(t, recorder.Body.Bytes())
	if body["code"] != float64(CodeBadRequest) || body["message"] != "请求体不是合法的 JSON" {
		t.Fatalf("body = %#v", body)
	}
}

func decode(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("json: %v body=%s", err, raw)
	}
	return body
}
