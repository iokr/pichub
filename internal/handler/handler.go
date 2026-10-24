// Package handler 是 HTTP 薄层：绑定参数、调用 service、组装响应。
package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/auth"
	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/middleware"
	"github.com/pichub/pichub/internal/model"
	"github.com/pichub/pichub/internal/service"
)

// ReadyFunc 检查依赖（数据库/Redis）是否就绪。
type ReadyFunc func(ctx context.Context) error

// Handler 聚合全部 HTTP 处理器。
type Handler struct {
	cfg   *config.Config
	log   *logger.Logger
	svc   *service.Services
	ready ReadyFunc
}

// New 创建处理器集合。
func New(cfg *config.Config, log *logger.Logger, svc *service.Services, ready ReadyFunc) *Handler {
	return &Handler{cfg: cfg, log: log, svc: svc, ready: ready}
}

// user 返回当前登录用户。
func (h *Handler) user(c *gin.Context) *model.User {
	return middleware.User(c)
}

// userID 返回当前登录用户 ID。
func (h *Handler) userID(c *gin.Context) uint64 {
	return middleware.UserID(c)
}

// claims 返回当前 JWT 载荷（API Token 访问时为空）。
func (h *Handler) claims(c *gin.Context) *auth.Claims {
	return middleware.Claims(c)
}

// meta 返回客户端 IP 与 User-Agent。
func (h *Handler) meta(c *gin.Context) (string, string) {
	return c.ClientIP(), c.Request.UserAgent()
}
