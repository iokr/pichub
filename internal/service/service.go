package service

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/pichub/pichub/internal/auth"
	"github.com/pichub/pichub/internal/cache"
	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/crypto"
	"github.com/pichub/pichub/internal/imageproc"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/mail"
	"github.com/pichub/pichub/internal/model"
)

// Deps 是业务层的全部依赖。
type Deps struct {
	Config     *config.Config
	Log        *logger.Logger
	Repos      Repos
	JWT        *auth.Manager
	Blacklist  *cache.TokenBlacklist
	Limiter    *cache.RateLimiter
	Captcha    *cache.Captcha
	Deduper    *cache.Deduper
	Buffer     *cache.RefererBuffer
	Cipher     *crypto.Cipher
	Mailer     mail.Sender
	Queue      Enqueuer
	QueueStats QueueStats
	Storages   StorageManager
	Processor  *imageproc.Processor
}

// Services 聚合全部业务服务。
type Services struct {
	Settings *SettingsService
	Email    *EmailService
	Auth     *AuthService
	User     *UserService
	Upload   *UploadService
	Image    *ImageService
	Album    *AlbumService
	Tag      *TagService
	Token    *TokenService
	Share    *ShareService
	Storage  *StorageService
	Referer  *RefererService
	Stats    *StatsService
	Admin    *AdminService
}

// New 按依赖顺序装配全部服务。
func New(d Deps) *Services {
	settings := NewSettingsService(d.Repos.Setting, d.Config, d.Log)
	emailSvc := NewEmailService(d, settings)
	authSvc := NewAuthService(d, settings, emailSvc)
	userSvc := NewUserService(d, settings)
	tokenSvc := NewTokenService(d)
	albumSvc := NewAlbumService(d, settings)
	tagSvc := NewTagService(d)
	refererSvc := NewRefererService(d, settings)
	imageSvc := NewImageService(d, settings, albumSvc, tagSvc, refererSvc)
	uploadSvc := NewUploadService(d, settings, imageSvc)
	shareSvc := NewShareService(d, settings)
	storageSvc := NewStorageService(d)
	statsSvc := NewStatsService(d, refererSvc)
	adminSvc := NewAdminService(d, settings, statsSvc)
	return &Services{
		Settings: settings,
		Email:    emailSvc,
		Auth:     authSvc,
		User:     userSvc,
		Upload:   uploadSvc,
		Image:    imageSvc,
		Album:    albumSvc,
		Tag:      tagSvc,
		Token:    tokenSvc,
		Share:    shareSvc,
		Storage:  storageSvc,
		Referer:  refererSvc,
		Stats:    statsSvc,
		Admin:    adminSvc,
	}
}

// logOp 异步安全地写入操作日志，失败仅记录告警。
func logOp(ctx context.Context, log *logger.Logger, repo LogRepo, userID uint64, action, targetType string, targetID uint64, ip, ua string, detail any) {
	if repo == nil {
		return
	}
	l := &model.OperationLog{
		UserID:     userID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		IP:         ip,
		UserAgent:  trimTo(ua, 255),
	}
	if detail != nil {
		if b, err := json.Marshal(detail); err == nil {
			l.Detail = b
		}
	}
	if err := repo.Create(ctx, l); err != nil && log != nil {
		log.Warn("write operation log failed", logger.Err(err))
	}
}

func trimTo(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// keyMu 用于进程内的短临界区保护。
var keyMu sync.Mutex
