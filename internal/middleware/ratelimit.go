package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/cache"
	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
)

// Limiter 基于 Redis 的固定窗口限流中间件。
type Limiter struct {
	limiter *cache.RateLimiter
	log     *logger.Logger
}

// NewLimiter 创建限流器。
func NewLimiter(l *cache.RateLimiter, log *logger.Logger) *Limiter {
	return &Limiter{limiter: l, log: log}
}

// ByIP 按客户端 IP 限流。
func (l *Limiter) ByIP(name string, limit int, window time.Duration) gin.HandlerFunc {
	return l.middleware(name, limit, window, func(c *gin.Context) string {
		return "ip:" + c.ClientIP()
	})
}

// ByUser 按登录用户限流，未登录时退化为 IP。
func (l *Limiter) ByUser(name string, limit int, window time.Duration) gin.HandlerFunc {
	return l.middleware(name, limit, window, func(c *gin.Context) string {
		if uid := UserID(c); uid > 0 {
			return "user:" + strconv.FormatUint(uid, 10)
		}
		return "ip:" + c.ClientIP()
	})
}

func (l *Limiter) middleware(name string, limit int, window time.Duration, keyFn func(*gin.Context) string) gin.HandlerFunc {
	if l == nil || l.limiter == nil || limit <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		key := name + ":" + keyFn(c)
		allowed, count, err := l.limiter.Allow(c.Request.Context(), key, limit, window)
		if err != nil {
			// Redis 故障时放行，避免限流组件拖垮业务。
			if l.log != nil {
				l.log.Warn("rate limit check failed", logger.Err(err), logger.String("key", key))
			}
			c.Next()
			return
		}
		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(window.Seconds())))
			c.AbortWithStatusJSON(errs.ErrTooManyRequests.HTTPStatus(), dto.Response{
				Code:    errs.ErrTooManyRequests.Code(),
				Message: errs.ErrTooManyRequests.Message(),
				Data:    gin.H{"count": count},
			})
			return
		}
		c.Next()
	}
}
