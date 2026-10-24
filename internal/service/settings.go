package service

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
)

// 系统设置键名。
const (
	KeySiteName                   = "site.name"
	KeySiteURL                    = "site.url"
	KeyRegisterEnabled            = "register.enabled"
	KeyRegisterMode               = "register.mode"
	KeyRegisterRequireEmailVerify = "register.require_email_verify"
	KeyRegisterIPLimitPerDay      = "register.ip_limit_per_day"
	KeyRegisterDefaultGroupID     = "register.default_group_id"
	KeyCaptchaEnabled             = "captcha.enabled"
	KeyCaptchaLoginTriggerFails   = "captcha.login_trigger_fails"
	KeyUploadDefaultVisibility    = "upload.default_visibility"
	KeyUploadAllowedExtensions    = "upload.allowed_extensions"
	KeyRefererEnabled             = "referer.enabled"
	KeyRefererMaxPerImage         = "referer.max_per_image"
	KeyS3GatewayEnabled           = "s3_gateway.enabled"
	KeyS3GatewayPathPrefix        = "s3_gateway.path_prefix"
)

// defaultSettings 是数据库缺失配置时的兜底值。
var defaultSettings = map[string]string{
	KeySiteName:                   "PicHub",
	KeyRegisterEnabled:            "true",
	KeyRegisterMode:               "open",
	KeyRegisterRequireEmailVerify: "force",
	KeyRegisterIPLimitPerDay:      "3",
	KeyRegisterDefaultGroupID:     "1",
	KeyCaptchaEnabled:             "true",
	KeyCaptchaLoginTriggerFails:   "2",
	KeyUploadDefaultVisibility:    "public",
	KeyRefererEnabled:             "true",
	KeyRefererMaxPerImage:         "200",
	KeyS3GatewayEnabled:           "true",
	KeyS3GatewayPathPrefix:        "/s3",
}

// SettingsService 提供系统设置的读取缓存与更新。
type SettingsService struct {
	repo SettingRepo
	cfg  *config.Config
	log  *logger.Logger

	mu       sync.RWMutex
	values   map[string]string
	loadedAt time.Time
	ttl      time.Duration
}

// NewSettingsService 创建设置服务。
func NewSettingsService(repo SettingRepo, cfg *config.Config, log *logger.Logger) *SettingsService {
	return &SettingsService{repo: repo, cfg: cfg, log: log, values: map[string]string{}, ttl: time.Minute}
}

// Get 返回配置值，未配置时返回兜底值。
func (s *SettingsService) Get(ctx context.Context, key string) string {
	s.ensure(ctx)
	s.mu.RLock()
	v, ok := s.values[key]
	s.mu.RUnlock()
	if ok {
		return v
	}
	return defaultSettings[key]
}

// GetString 返回配置值或指定默认值。
func (s *SettingsService) GetString(ctx context.Context, key, def string) string {
	if v := s.Get(ctx, key); v != "" {
		return v
	}
	return def
}

// GetBool 解析布尔配置。
func (s *SettingsService) GetBool(ctx context.Context, key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(s.Get(ctx, key)))
	switch v {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	}
	return def
}

// GetInt 解析整型配置。
func (s *SettingsService) GetInt(ctx context.Context, key string, def int) int {
	v := strings.TrimSpace(s.Get(ctx, key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// SiteName 返回站点名称。
func (s *SettingsService) SiteName(ctx context.Context) string {
	if v := s.Get(ctx, KeySiteName); v != "" {
		return v
	}
	return "PicHub"
}

// List 返回全部设置项（含明文值）。
func (s *SettingsService) List(ctx context.Context) ([]*model.SystemSetting, error) {
	list, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return list, nil
}

// Update 批量更新设置项。密文占位符 "******" 会被忽略。
func (s *SettingsService) Update(ctx context.Context, updates map[string]string, adminID uint64, ip, ua string) error {
	if len(updates) == 0 {
		return nil
	}
	existing, err := s.repo.GetAll(ctx)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	byKey := make(map[string]*model.SystemSetting, len(existing))
	for _, item := range existing {
		byKey[item.CfgKey] = item
	}
	items := make([]*model.SystemSetting, 0, len(updates))
	for key, value := range updates {
		cur, ok := byKey[key]
		if !ok {
			return errs.ErrInvalidParam.WithDetail("未知配置项: " + key)
		}
		if cur.IsSecret && value == "******" {
			continue
		}
		items = append(items, &model.SystemSetting{
			CfgKey:      key,
			CfgValue:    value,
			ValueType:   cur.ValueType,
			IsSecret:    cur.IsSecret,
			Description: cur.Description,
		})
	}
	if err := s.repo.SetMany(ctx, items); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	s.Invalidate()
	logOp(ctx, s.log, nil, adminID, "settings.update", "settings", 0, ip, ua, map[string]any{"keys": len(items)})
	if s.log != nil {
		s.log.Info("settings updated", logger.Uint64("admin_id", adminID), logger.Int("count", len(items)))
	}
	return nil
}

// Invalidate 使缓存立即失效。
func (s *SettingsService) Invalidate() {
	s.mu.Lock()
	s.loadedAt = time.Time{}
	s.mu.Unlock()
}

func (s *SettingsService) ensure(ctx context.Context) {
	s.mu.RLock()
	fresh := time.Since(s.loadedAt) < s.ttl && !s.loadedAt.IsZero()
	s.mu.RUnlock()
	if fresh {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Since(s.loadedAt) < s.ttl && !s.loadedAt.IsZero() {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	list, err := s.repo.GetAll(ctx)
	if err != nil {
		if s.log != nil {
			s.log.Warn("load settings failed", logger.Err(err))
		}
		return
	}
	values := make(map[string]string, len(list))
	for _, item := range list {
		values[item.CfgKey] = item.CfgValue
	}
	s.values = values
	s.loadedAt = time.Now()
	if s.cfg != nil {
		// 站点地址以数据库设置为准，为空时回退到配置文件。
		s.cfg.SetSiteURL(values[KeySiteURL])
	}
}
