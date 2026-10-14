package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/api"
)

// OpenAPI serves the OpenAPI document for the routes that exist today.
func OpenAPI(c *gin.Context) {
	c.Data(http.StatusOK, "application/json; charset=utf-8", api.OpenAPI)
}
