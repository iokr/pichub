package service

import (
	"context"
	"time"

	"github.com/pichub/pichub/internal/auth"
	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/mail"
	"github.com/pichub/pichub/internal/model"
	"github.com/pichub/pichub/internal/queue"
)

// 邮箱令牌有效期。
const (
	verifyTokenTTL = 24 * time.Hour
	resetTokenTTL  = 30 * time.Minute
)

// EmailService 负责生成邮箱令牌、投递邮件任务与执行发送。
type EmailService struct {
	cfg      *config.Config
	log      *logger.Logger
	mailer   mail.Sender
	repo     EmailRepo
	queue    Enqueuer
	settings *SettingsService
}

// NewEmailService 创建邮件服务。
func NewEmailService(d Deps, settings *SettingsService) *EmailService {
	return &EmailService{
		cfg:      d.Config,
		log:      d.Log,
		mailer:   d.Mailer,
		repo:     d.Repos.Email,
		queue:    d.Queue,
		settings: settings,
	}
}

// Enabled 返回邮件功能是否可用。
func (s *EmailService) Enabled() bool { return s.mailer != nil && s.mailer.Enabled() }

// SendVerify 为用户生成邮箱验证令牌并投递邮件。
func (s *EmailService) SendVerify(ctx context.Context, u *model.User) error {
	if u == nil || u.Email == "" {
		return nil
	}
	return s.enqueueToken(ctx, u, model.EmailSceneVerify, u.Email, verifyTokenTTL)
}

// SendReset 为用户生成重置密码令牌并投递邮件。
func (s *EmailService) SendReset(ctx context.Context, u *model.User) error {
	if u == nil || u.Email == "" {
		return nil
	}
	return s.enqueueToken(ctx, u, model.EmailSceneReset, u.Email, resetTokenTTL)
}

func (s *EmailService) enqueueToken(ctx context.Context, u *model.User, scene, email string, ttl time.Duration) error {
	raw, err := auth.NewOpaqueToken(32)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	now := model.Now()
	tok := &model.EmailToken{
		UserID:    u.ID,
		Email:     email,
		TokenHash: auth.HashToken(raw),
		Scene:     scene,
		ExpiresAt: now.Add(ttl),
	}
	if err := s.repo.CreateToken(ctx, tok); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	kind := mail.KindVerify
	path := "/verify-email?token=" + raw
	if scene == model.EmailSceneReset {
		kind = mail.KindReset
		path = "/reset-password?token=" + raw
	}
	payload := queue.SendEmailPayload{
		UserID:        u.ID,
		Kind:          kind,
		To:            email,
		Username:      u.Username,
		URL:           s.cfg.SiteURL() + path,
		Email:         email,
		ExpireMinutes: int(ttl.Minutes()),
	}
	if s.queue != nil {
		if err := s.queue.Enqueue(ctx, queue.TaskSendEmail, payload); err != nil {
			if s.log != nil {
				s.log.Warn("enqueue email task failed", logger.Err(err), logger.Uint64("user_id", u.ID))
			}
			// 入队失败时同步兜底发送，避免用户收不到邮件。
			return s.HandleTask(ctx, payload)
		}
		return nil
	}
	return s.HandleTask(ctx, payload)
}

// HandleTask 执行邮件发送任务（由队列 worker 调用）。
func (s *EmailService) HandleTask(ctx context.Context, payload queue.SendEmailPayload) error {
	entry := &model.EmailLog{
		UserID:   payload.UserID,
		ToEmail:  payload.To,
		Template: payload.Kind,
		Status:   model.EmailStatusPending,
	}
	_ = s.repo.CreateLog(ctx, entry)

	if !s.Enabled() {
		_ = s.repo.UpdateLog(ctx, entry.ID, map[string]any{
			"status": model.EmailStatusFailed,
			"error":  mail.ErrDisabled.Error(),
		})
		if s.log != nil {
			s.log.Warn("skip email because smtp disabled", logger.String("to", payload.To))
		}
		return nil
	}

	if limit := s.cfg.Mail.DailyLimit; limit > 0 {
		today := time.Now().UTC().Truncate(24 * time.Hour)
		n, err := s.repo.CountTodaySuccess(ctx, today)
		if err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		if n >= int64(limit) {
			_ = s.repo.UpdateLog(ctx, entry.ID, map[string]any{
				"status": model.EmailStatusFailed,
				"error":  "daily mail limit reached",
			})
			return nil
		}
	}

	msg, err := mail.Build(payload.Kind, payload.To, mail.TemplateData{
		SiteName:      s.settings.SiteName(ctx),
		Username:      payload.Username,
		Email:         payload.Email,
		ActionURL:     payload.URL,
		ExpireMinutes: payload.ExpireMinutes,
	})
	if err != nil {
		_ = s.repo.UpdateLog(ctx, entry.ID, map[string]any{
			"status": model.EmailStatusFailed,
			"error":  trimTo(err.Error(), 500),
		})
		return err
	}
	entry.Subject = msg.Subject
	_ = s.repo.UpdateLog(ctx, entry.ID, map[string]any{"subject": msg.Subject})

	if err := s.mailer.Send(ctx, msg); err != nil {
		_ = s.repo.UpdateLog(ctx, entry.ID, map[string]any{
			"status": model.EmailStatusFailed,
			"error":  trimTo(err.Error(), 500),
		})
		return err
	}
	_ = s.repo.UpdateLog(ctx, entry.ID, map[string]any{
		"status":  model.EmailStatusSuccess,
		"sent_at": model.Now(),
	})
	return nil
}
