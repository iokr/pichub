package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
)

// Recovery 捕获 panic，返回统一错误响应。
func Recovery(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				if log != nil {
					log.Error("panic recovered",
						logger.Any("panic", r),
						logger.String("method", c.Request.Method),
						logger.String("path", c.Request.URL.Path),
						logger.String("request_id", RequestID(c)),
						logger.String("stack", string(debug.Stack())),
					)
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, dto.Response{
					Code:    errs.CodeInternal,
					Message: "内部错误",
				})
			}
		}()
		c.Next()
	}
}
