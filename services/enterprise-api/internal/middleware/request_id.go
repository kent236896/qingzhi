// Package middleware contains HTTP middleware shared by every route.
package middleware

import (
	"crypto/rand"
	"fmt"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/response"
)

const requestIDHeader = "X-Request-ID"

// RequestID accepts a safe client request id or generates one.
// The id is returned in the response header and the JSON envelope.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if !validRequestID(id) {
			id = newRequestID()
		}
		c.Set(response.ContextRequestID, id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

func validRequestID(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return false
		}
	}
	return true
}

func newRequestID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "request-id-unavailable"
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}
