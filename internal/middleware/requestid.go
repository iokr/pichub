package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// HeaderRequestID 是请求 ID 响应头。
const HeaderRequestID = "X-Request-ID"

// WithRequestID 为每个请求生成或透传请求 ID。
func WithRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			b := make([]byte, 8)
			if _, err := rand.Read(b); err == nil {
				id = hex.EncodeToString(b)
			} else {
				id = "unknown"
			}
		}
		c.Set(CtxRequestID, id)
		c.Header(HeaderRequestID, id)
		c.Next()
	}
}
