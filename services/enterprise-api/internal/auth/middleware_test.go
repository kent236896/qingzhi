package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireTenantMatchRejectsAnotherTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "tenantId", Value: "tenant-b"}}
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tenants/tenant-b/organizations", nil)
	SetPrincipal(context, Principal{UserID: "user-1", TenantID: "tenant-a", Permissions: map[string]struct{}{
		"system:organization:query": {},
	}})

	RequireTenantMatch("tenantId")(context)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestRequireTenantMatchOrPermissionAllowsTenantAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "tenantId", Value: "tenant-b"}}
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tenants/tenant-b/organizations", nil)
	SetPrincipal(context, Principal{UserID: "user-1", TenantID: "tenant-a", Permissions: map[string]struct{}{
		"system:tenant:query": {},
	}})

	RequireTenantMatchOrPermission("tenantId", "system:tenant:query")(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestRequireTenantMatchAllowsSessionTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "tenantId", Value: "tenant-a"}}
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tenants/tenant-a/organizations", nil)
	SetPrincipal(context, Principal{UserID: "user-1", TenantID: "tenant-a"})

	RequireTenantMatch("tenantId")(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}
