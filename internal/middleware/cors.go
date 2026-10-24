package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 允许前端跨域访问 API（生产环境由同源托管，保持宽松无副作用）。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, X-Amz-Date, X-Amz-Content-Sha256, X-Amz-Security-Token")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, ETag, Content-Length")
		c.Header("Access-Control-Max-Age", "86400")
		if strings.EqualFold(c.Request.Method, http.MethodOptions) {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
