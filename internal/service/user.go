package service

import (
	"context"
	"strings"
	"time"

	"github.com/pichub/pichub/internal/auth"
	"github.com/pichub/pichub/internal/cache"
	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
)

// UserService 处理用户资料、密码与用量统计。
type UserService struct {
	cfg      *config.Config
	log      *logger.Logger
	repos    Repos
	settings *SettingsService
	captcha  *cache.Captcha
}

// NewUserService 创建用户服务。
func NewUserService(d Deps, settings *SettingsService) *UserService {
	return &UserService{cfg: d.Config, log: d.Log, repos: d.Repos, settings: settings, captcha: d.Captcha}
}

// Get 按 ID 查询用户。
func (s *UserService) Get(ctx context.Context, id uint64) (*model.User, error) {
	u, err := s.repos.User.GetByID(ctx, id)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if u == nil {
		return nil, errs.ErrNotFound.WithMessage("用户不存在")
	}
	return u, nil
}

// Profile 返回用户资料。
func (s *UserService) Profile(ctx context.Context, userID uint64) (*dto.UserResp, error) {
	u, err := s.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dto.NewUserResp(u), nil
}

// UpdateProfile 更新用户资料。
func (s *UserService) UpdateProfile(ctx context.Context, userID uint64, req *dto.UpdateProfileReq) (*dto.UserResp, error) {
	fields := map[string]any{}
	if req.Nickname != nil {
		name := strings.TrimSpace(*req.Nickname)
		if name == "" || len([]rune(name)) > 32 {
			return nil, errs.ErrInvalidParam.WithDetail("昵称长度需为 1-32 个字符")
		}
		fields["nickname"] = name
	}
	if req.AvatarURL != nil {
		url := strings.TrimSpace(*req.AvatarURL)
		if len(url) > 500 {
			return nil, errs.ErrInvalidParam.WithDetail("头像地址过长")
		}
		fields["avatar_url"] = url
	}
	if req.DefaultVisibility != nil {
		v := strings.ToLower(strings.TrimSpace(*req.DefaultVisibility))
		if v != "public" && v != "private" {
			return nil, errs.ErrInvalidParam.WithDetail("可见性只能是 public 或 private")
		}
		fields["default_visibility"] = v
	}
	if req.RefererStatEnabled != nil {
		if !s.cfg.Referer.UserCanDisable {
			return nil, errs.ErrForbidden.WithMessage("引用统计不可由用户关闭")
		}
		fields["referer_stat_enabled"] = *req.RefererStatEnabled
	}
	if req.StorageConfigID != nil {
		if !s.cfg.Storage.AllowUserStrategy {
			return nil, errs.ErrForbidden.WithMessage("自定义存储未开放")
		}
		if *req.StorageConfigID == 0 {
			fields["storage_config_id"] = nil
		} else {
			sc, err := s.repos.Storage.GetByID(ctx, *req.StorageConfigID)
			if err != nil {
				return nil, errs.ErrInternal.Wrap(err)
			}
			if sc == nil || sc.Status != model.StatusEnabled || sc.UserID == nil || *sc.UserID != userID {
				return nil, errs.ErrInvalidParam.WithDetail("存储策略不可用")
			}
			fields["storage_config_id"] = sc.ID
		}
	}
	if len(fields) > 0 {
		if err := s.repos.User.UpdateFields(ctx, userID, fields); err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
	}
	return s.Profile(ctx, userID)
}

// ChangePassword 修改密码，需要校验原密码与验证码。
func (s *UserService) ChangePassword(ctx context.Context, userID uint64, req *dto.ChangePasswordReq, ip, ua string) error {
	u, err := s.Get(ctx, userID)
	if err != nil {
		return err
	}
	if s.captcha != nil && s.cfg.Captcha.Enabled &&
		s.settings.GetBool(ctx, KeyCaptchaEnabled, true) {
		if !s.captcha.Verify(req.CaptchaID, req.CaptchaCode) {
			return errs.ErrCaptchaWrong
		}
	}
	if !auth.VerifyPassword(u.PasswordHash, req.OldPassword) {
		return errs.ErrInvalidParam.WithMessage("原密码错误")
	}
	if err := auth.ValidatePasswordStrength(&s.cfg.Security, req.NewPassword); err != nil {
		return errs.ErrPasswordWeak.WithMessage(err.Error())
	}
	if req.OldPassword == req.NewPassword {
		return errs.ErrInvalidParam.WithMessage("新密码不能与原密码相同")
	}
	hash, err := auth.HashPassword(req.NewPassword, s.cfg.Security.BcryptCost)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if err := s.repos.User.UpdateFields(ctx, userID, map[string]any{"password_hash": hash}); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	logOp(ctx, s.log, s.repos.Log, userID, "user.change_password", "user", userID, ip, ua, nil)
	return nil
}

// Stats 返回用户用量统计。
func (s *UserService) Stats(ctx context.Context, userID uint64) (*dto.UserStatsResp, error) {
	u, err := s.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	imageCount, err := s.repos.Image.CountByUser(ctx, userID, model.ImageStatusNormal)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	trashCount, err := s.repos.Image.CountByUser(ctx, userID, model.ImageStatusTrash)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	albumCount, err := s.repos.Album.CountByUser(ctx, userID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	shareCount, err := s.repos.Share.CountByUser(ctx, userID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	size, err := s.repos.Image.SumSizeByUser(ctx, userID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	today := 0
	if counter, err := s.repos.Counter.Get(ctx, userID, model.Today()); err == nil && counter != nil {
		today = counter.Count
	}
	hits, err := s.repos.Referer.SumHitsByUser(ctx, userID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	now := time.Now().UTC()
	topHosts, err := s.repos.Referer.TopHosts(ctx, userID, now.AddDate(0, 0, -30), 10)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	trend, err := s.repos.Referer.Trend(ctx, userID, now.AddDate(0, 0, -29), now)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return &dto.UserStatsResp{
		ImageCount:   imageCount,
		AlbumCount:   albumCount,
		ShareCount:   shareCount,
		TrashCount:   trashCount,
		StorageBytes: size,
		UsedBytes:    u.UsedBytes,
		QuotaBytes:   u.QuotaBytes,
		DailyLimit:   u.DailyLimit,
		TodayUploads: today,
		RefererHits:  hits,
		TopHosts:     topHosts,
		Trend:        trend,
	}, nil
}
