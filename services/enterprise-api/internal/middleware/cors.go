package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/config"
)

// CORS applies the configured allowed origins.
func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	corsConfig := cors.Config{
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders: []string{
			"X-Request-ID",
		},
		AllowCredentials: false,
		MaxAge:           10 * time.Minute,
	}
	if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = cfg.AllowedOrigins
	}
	return cors.New(corsConfig)
}
