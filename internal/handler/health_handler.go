package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/service"
)

// Healthz 是存活探针。
func (h *Handler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz 是就绪探针，检查数据库与 Redis。
func (h *Handler) Readyz(c *gin.Context) {
	if h.ready != nil {
		if err := h.ready(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// SiteConfig 返回站点公开配置。
func (h *Handler) SiteConfig(c *gin.Context) {
	ctx := c.Request.Context()
	settings := h.svc.Settings

	level := strings.ToLower(settings.GetString(ctx, service.KeyRegisterRequireEmailVerify, h.cfg.Register.RequireEmailVerify))
	switch level {
	case "off", "soft", "force":
	default:
		level = "force"
	}
	if level == "force" && !h.cfg.Mail.Enabled {
		level = "soft"
	}

	sizes := make([]string, 0, len(h.cfg.Image.Thumbnail.Sizes))
	for _, s := range h.cfg.Image.Thumbnail.Sizes {
		if s.Name != "" {
			sizes = append(sizes, s.Name)
		}
	}

	OK(c, dto.SiteConfigResp{
		Name:              settings.SiteName(ctx),
		Description:       h.cfg.Site.Description,
		LogoURL:           h.cfg.Site.LogoURL,
		ICP:               h.cfg.Site.ICP,
		SiteURL:           h.cfg.SiteURL(),
		RegisterEnabled:   settings.GetBool(ctx, service.KeyRegisterEnabled, h.cfg.Register.Enabled),
		RegisterMode:      settings.GetString(ctx, service.KeyRegisterMode, h.cfg.Register.Mode),
		EmailVerifyLevel:  level,
		CaptchaEnabled:    h.cfg.Captcha.Enabled && settings.GetBool(ctx, service.KeyCaptchaEnabled, true),
		DefaultVisibility: settings.GetString(ctx, service.KeyUploadDefaultVisibility, h.cfg.Upload.DefaultVisibility),
		MaxFileSize:       h.cfg.Upload.MaxFileSize,
		AllowedExtensions: h.allowedExtensions(ctx),
		ThumbnailEnabled:  h.cfg.Image.Thumbnail.Enabled,
		ThumbnailSizes:    sizes,
		S3GatewayEnabled:  settings.GetBool(ctx, service.KeyS3GatewayEnabled, h.cfg.S3Gateway.Enabled),
		S3GatewayPrefix:   settings.GetString(ctx, service.KeyS3GatewayPathPrefix, h.cfg.S3Gateway.PathPrefix),
	})
}

func (h *Handler) allowedExtensions(ctx context.Context) []string {
	exts := h.cfg.Upload.AllowedExtensions
	raw := strings.TrimSpace(h.svc.Settings.Get(ctx, service.KeyUploadAllowedExtensions))
	if raw != "" {
		var parsed []string
		if strings.HasPrefix(raw, "[") {
			_ = json.Unmarshal([]byte(raw), &parsed)
		}
		if len(parsed) == 0 {
			parsed = strings.Split(raw, ",")
		}
		if len(parsed) > 0 {
			exts = parsed
		}
	}
	out := make([]string, 0, len(exts))
	for _, e := range exts {
		e = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(e), "."))
		if e != "" {
			out = append(out, e)
		}
	}
	return out
}
