package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/logger"
)

// AccessLogger 记录访问日志，探针与静态资源不记录。
func AccessLogger(log *logger.Logger) gin.HandlerFunc {
	if log == nil {
		return func(c *gin.Context) { c.Next() }
	}
	skip := map[string]struct{}{
		"/healthz": {}, "/readyz": {}, "/metrics": {}, "/favicon.ico": {},
	}
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if _, ok := skip[path]; ok {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		cost := time.Since(start)
		fields := []logger.Field{
			logger.String("method", c.Request.Method),
			logger.String("path", path),
			logger.String("query", c.Request.URL.RawQuery),
			logger.Int("status", c.Writer.Status()),
			logger.Duration("latency", cost),
			logger.String("ip", c.ClientIP()),
			logger.String("request_id", RequestID(c)),
		}
		if uid := UserID(c); uid > 0 {
			fields = append(fields, logger.Uint64("user_id", uid))
		}
		if c.Writer.Status() >= 500 {
			// 仅在服务端错误时输出错误详情，避免 4xx 噪音。
			if len(c.Errors) > 0 {
				fields = append(fields, logger.String("errors", c.Errors.String()))
			}
			log.Error("http request", fields...)
			return
		}
		log.Info("http request", fields...)
	}
}
