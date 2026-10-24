package service

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pichub/pichub/internal/auth"
	"github.com/pichub/pichub/internal/cache"
	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/mail"
	"github.com/pichub/pichub/internal/model"
)

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// AuthService 处理注册、登录、令牌与邮箱验证。
type AuthService struct {
	cfg       *config.Config
	log       *logger.Logger
	repos     Repos
	jwt       *auth.Manager
	blacklist *cache.TokenBlacklist
	limiter   *cache.RateLimiter
	captcha   *cache.Captcha
	email     *EmailService
	settings  *SettingsService
	mailer    mail.Sender
}

// NewAuthService 创建认证服务。
func NewAuthService(d Deps, settings *SettingsService, email *EmailService) *AuthService {
	return &AuthService{
		cfg:       d.Config,
		log:       d.Log,
		repos:     d.Repos,
		jwt:       d.JWT,
		blacklist: d.Blacklist,
		limiter:   d.Limiter,
		captcha:   d.Captcha,
		email:     email,
		settings:  settings,
		mailer:    d.Mailer,
	}
}

// Captcha 生成图片验证码。
func (s *AuthService) Captcha(ctx context.Context) (*dto.CaptchaResp, error) {
	if s.captcha == nil || !s.captchaEnabled(ctx) {
		return nil, errs.ErrInvalidParam.WithMessage("验证码未启用")
	}
	id, b64, err := s.captcha.Generate()
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return &dto.CaptchaResp{
		CaptchaID:   id,
		ImageBase64: b64,
		ExpiresIn:   s.captcha.ExpireSeconds(),
	}, nil
}

// CheckUsername 检查用户名是否可用。
func (s *AuthService) CheckUsername(ctx context.Context, name string) (*dto.CheckResp, error) {
	name = strings.TrimSpace(name)
	if err := auth.ValidateUsername(&s.cfg.Register, name); err != nil {
		return &dto.CheckResp{Available: false, Message: err.Error()}, nil
	}
	u, err := s.repos.User.GetByUsername(ctx, name)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if u != nil {
		return &dto.CheckResp{Available: false, Message: "用户名已占用"}, nil
	}
	return &dto.CheckResp{Available: true, Message: "用户名可用"}, nil
}

// CheckEmail 检查邮箱是否可用。
func (s *AuthService) CheckEmail(ctx context.Context, email string) (*dto.CheckResp, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !emailPattern.MatchString(email) {
		return &dto.CheckResp{Available: false, Message: "邮箱格式不正确"}, nil
	}
	u, err := s.repos.User.GetByEmail(ctx, email)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if u != nil {
		return &dto.CheckResp{Available: false, Message: "邮箱已注册"}, nil
	}
	return &dto.CheckResp{Available: true, Message: "邮箱可用"}, nil
}

// Register 注册新用户。
func (s *AuthService) Register(ctx context.Context, req *dto.RegisterReq, ip, ua string) (*dto.AuthResp, error) {
	if !s.settings.GetBool(ctx, KeyRegisterEnabled, s.cfg.Register.Enabled) {
		return nil, errs.ErrForbidden.WithMessage("注册通道已关闭")
	}
	if s.captchaEnabled(ctx) {
		if !s.captcha.Verify(req.CaptchaID, req.CaptchaCode) {
			return nil, errs.ErrCaptchaWrong
		}
	}
	username := strings.TrimSpace(req.Username)
	if err := auth.ValidateUsername(&s.cfg.Register, username); err != nil {
		return nil, errs.ErrUsernameInvalid.WithMessage(err.Error())
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !emailPattern.MatchString(email) {
		return nil, errs.ErrInvalidParam.WithDetail("邮箱格式不正确")
	}
	if s.emailBlocked(email) {
		return nil, errs.ErrInvalidParam.WithDetail("该邮箱域名不允许注册")
	}
	if err := auth.ValidatePasswordStrength(&s.cfg.Security, req.Password); err != nil {
		return nil, errs.ErrPasswordWeak.WithMessage(err.Error())
	}
	exists, err := s.repos.User.GetByUsername(ctx, username)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if exists != nil {
		return nil, errs.ErrUsernameTaken
	}
	exists, err = s.repos.User.GetByEmail(ctx, email)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if exists != nil {
		return nil, errs.ErrEmailTaken
	}

	// IP 每日注册上限。
	ipLimit := s.settings.GetInt(ctx, KeyRegisterIPLimitPerDay, s.cfg.Register.IPLimitPerDay)
	if ipLimit > 0 {
		n, err := s.repos.User.CountByIP(ctx, ip, time.Now().UTC().Add(-24*time.Hour))
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if n >= int64(ipLimit) {
			return nil, errs.ErrTooManyRequests.WithMessage("当前 IP 注册次数已达上限")
		}
	}

	// 邀请码与用户组。
	var invite *model.Invite
	groupID := s.defaultGroupID(ctx)
	mode := strings.ToLower(s.settings.GetString(ctx, KeyRegisterMode, s.cfg.Register.Mode))
	if mode == "invite" {
		code := strings.TrimSpace(req.InviteCode)
		if code == "" {
			return nil, errs.ErrInviteInvalid.WithMessage("邀请码不能为空")
		}
		invite, err = s.repos.Invite.GetByCode(ctx, code)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if invite == nil || invite.Status != model.StatusEnabled ||
			(invite.ExpiresAt != nil && invite.ExpiresAt.Before(model.Now())) {
			return nil, errs.ErrInviteInvalid
		}
		if invite.GroupID > 0 {
			groupID = invite.GroupID
		}
	}
	group, err := s.repos.Group.GetByID(ctx, groupID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if group == nil || group.Status != model.StatusEnabled || !group.AllowRegister {
		if group, err = s.repos.Group.GetDefault(ctx); err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if group == nil {
			return nil, errs.ErrForbidden.WithMessage("注册暂不可用")
		}
	}

	hash, err := auth.HashPassword(req.Password, s.cfg.Security.BcryptCost)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	level := s.emailVerifyLevel(ctx)
	status := int8(model.UserStatusNormal)
	if level == "force" {
		status = model.UserStatusPending
	}
	visibility := s.settings.GetString(ctx, KeyUploadDefaultVisibility, s.cfg.Upload.DefaultVisibility)
	if visibility == "" {
		visibility = "public"
	}
	// 首个注册用户自动成为管理员，后续用户均为普通角色。
	role := int8(model.RoleUser)
	if n, cerr := s.repos.User.Count(ctx); cerr != nil {
		if s.log != nil {
			s.log.Warn("count users failed", logger.Err(cerr))
		}
	} else if n == 0 {
		role = model.RoleAdmin
	}
	u := &model.User{
		Username:           username,
		Email:              email,
		PasswordHash:       hash,
		Nickname:           username,
		Role:               role,
		Status:             status,
		EmailVerified:      level == "off",
		GroupID:            group.ID,
		QuotaBytes:         group.QuotaBytes,
		MaxFileSize:        group.MaxFileSize,
		DailyLimit:         group.DailyLimit,
		DefaultVisibility:  visibility,
		RefererStatEnabled: true,
		RegisterIP:         ip,
		LastLoginIP:        ip,
		LastLoginAt:        ptrTime(model.Now()),
	}
	if err := s.repos.User.Create(ctx, u); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if invite != nil {
		if ok, ierr := s.repos.Invite.IncrUsed(ctx, invite.ID); ierr == nil && !ok {
			if s.log != nil {
				s.log.Warn("invite code exhausted after registration", logger.Uint64("invite_id", invite.ID))
			}
		}
	}
	logOp(ctx, s.log, s.repos.Log, u.ID, "user.register", "user", u.ID, ip, ua, map[string]any{"username": username})
	if level == "force" {
		if err := s.email.SendVerify(ctx, u); err != nil && s.log != nil {
			s.log.Warn("send verify mail failed", logger.Err(err), logger.Uint64("user_id", u.ID))
		}
	}
	return s.issueTokens(ctx, u)
}

// Login 校验账号密码并签发令牌。
func (s *AuthService) Login(ctx context.Context, req *dto.LoginReq, ip, ua string) (*dto.AuthResp, error) {
	name := strings.TrimSpace(req.LoginName())
	if name == "" {
		return nil, errs.ErrInvalidParam.WithDetail("请输入用户名或邮箱")
	}
	failKey := "login:fail:" + ip
	fails, err := s.limiter.Count(ctx, failKey)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if max := s.cfg.Security.LoginMaxFails; max > 0 && fails >= int64(max) {
		return nil, errs.ErrTooManyRequests.WithMessage("登录失败次数过多，请稍后再试")
	}
	if s.captchaEnabled(ctx) && fails >= int64(s.settings.GetInt(ctx, KeyCaptchaLoginTriggerFails, s.cfg.Captcha.LoginTriggerFails)) {
		if req.CaptchaID == "" || req.CaptchaCode == "" {
			return nil, errs.ErrCaptchaRequired
		}
		if !s.captcha.Verify(req.CaptchaID, req.CaptchaCode) {
			return nil, errs.ErrCaptchaWrong
		}
	}

	var u *model.User
	if strings.Contains(name, "@") {
		u, err = s.repos.User.GetByEmail(ctx, strings.ToLower(name))
	} else {
		u, err = s.repos.User.GetByUsername(ctx, name)
	}
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if u == nil || !auth.VerifyPassword(u.PasswordHash, req.Password) {
		window := s.cfg.Security.LoginLockDuration
		if window <= 0 {
			window = 10 * time.Minute
		}
		_, _ = s.limiter.Incr(ctx, failKey, window)
		return nil, errs.ErrUnauthorized.WithMessage("用户名或密码错误")
	}
	_ = s.limiter.Reset(ctx, failKey)

	if u.Status == model.UserStatusDisabled {
		return nil, errs.ErrForbidden.WithMessage("账号已被禁用")
	}
	if s.emailVerifyLevel(ctx) == "force" && !u.EmailVerified {
		return nil, errs.ErrEmailUnverified
	}
	if err := s.repos.User.UpdateFields(ctx, u.ID, map[string]any{
		"last_login_ip": ip,
		"last_login_at": model.Now(),
	}); err != nil && s.log != nil {
		s.log.Warn("update login info failed", logger.Err(err), logger.Uint64("user_id", u.ID))
	}
	u.LastLoginIP = ip
	u.LastLoginAt = ptrTime(model.Now())
	logOp(ctx, s.log, s.repos.Log, u.ID, "user.login", "user", u.ID, ip, ua, nil)
	return s.issueTokens(ctx, u)
}

// Refresh 使用刷新令牌换取新的令牌对。
func (s *AuthService) Refresh(ctx context.Context, req *dto.RefreshReq) (*dto.AuthResp, error) {
	claims, err := s.jwt.ParseRefresh(req.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrTokenExpired) {
			return nil, errs.ErrTokenExpired
		}
		return nil, errs.ErrUnauthorized.WithDetail("刷新令牌无效")
	}
	if s.blacklist != nil {
		if revoked, rerr := s.blacklist.IsRevoked(ctx, claims.ID); rerr == nil && revoked {
			return nil, errs.ErrUnauthorized.WithDetail("刷新令牌已失效")
		}
	}
	u, err := s.repos.User.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if u == nil || u.Status == model.UserStatusDisabled {
		return nil, errs.ErrForbidden.WithMessage("账号不可用")
	}
	// 轮换刷新令牌，旧令牌立即失效。
	if s.blacklist != nil && claims.ExpiresAt != nil {
		_ = s.blacklist.Revoke(ctx, claims.ID, claims.ExpiresAt.Time)
	}
	return s.issueTokens(ctx, u)
}

// Logout 注销访问令牌与可选的刷新令牌。
func (s *AuthService) Logout(ctx context.Context, claims *auth.Claims, refreshToken string) error {
	if claims != nil && s.blacklist != nil {
		if claims.ExpiresAt != nil {
			_ = s.blacklist.Revoke(ctx, claims.ID, claims.ExpiresAt.Time)
		}
	}
	if refreshToken != "" && s.blacklist != nil {
		if rc, err := s.jwt.ParseRefresh(refreshToken); err == nil && rc.ExpiresAt != nil {
			_ = s.blacklist.Revoke(ctx, rc.ID, rc.ExpiresAt.Time)
		}
	}
	return nil
}

// VerifyEmail 校验邮箱令牌并激活账号。
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	tok, err := s.repos.Email.GetTokenByHash(ctx, auth.HashToken(strings.TrimSpace(token)))
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if tok == nil || tok.Scene != model.EmailSceneVerify || tok.UsedAt != nil || tok.ExpiresAt.Before(model.Now()) {
		return errs.ErrInvalidParam.WithMessage("验证链接无效或已过期")
	}
	u, err := s.repos.User.GetByID(ctx, tok.UserID)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if u == nil {
		return errs.ErrNotFound
	}
	if err := s.repos.User.UpdateFields(ctx, u.ID, map[string]any{
		"email_verified": true,
		"status":         model.UserStatusNormal,
	}); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if err := s.repos.Email.MarkTokenUsed(ctx, tok.ID, model.Now()); err != nil && s.log != nil {
		s.log.Warn("mark email token used failed", logger.Err(err))
	}
	return nil
}

// ResendVerify 重发邮箱验证邮件。
func (s *AuthService) ResendVerify(ctx context.Context, req *dto.ResendVerifyReq) error {
	if s.captchaEnabled(ctx) && !s.captcha.Verify(req.CaptchaID, req.CaptchaCode) {
		return errs.ErrCaptchaWrong
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	u, err := s.repos.User.GetByEmail(ctx, email)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	// 不暴露邮箱是否存在。
	if u == nil || u.EmailVerified {
		return nil
	}
	if s.emailVerifyLevel(ctx) == "off" {
		return nil
	}
	if !s.email.Enabled() {
		return errs.ErrForbidden.WithMessage("邮件服务未启用")
	}
	return s.email.SendVerify(ctx, u)
}

// ForgotPassword 发送重置密码邮件。
func (s *AuthService) ForgotPassword(ctx context.Context, req *dto.ForgotPasswordReq) error {
	if s.captchaEnabled(ctx) && !s.captcha.Verify(req.CaptchaID, req.CaptchaCode) {
		return errs.ErrCaptchaWrong
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	u, err := s.repos.User.GetByEmail(ctx, email)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if u == nil || u.Status == model.UserStatusDisabled {
		return nil
	}
	if !s.email.Enabled() {
		return errs.ErrForbidden.WithMessage("邮件服务未启用，请联系管理员")
	}
	return s.email.SendReset(ctx, u)
}

// ResetPassword 使用令牌重置密码。
func (s *AuthService) ResetPassword(ctx context.Context, req *dto.ResetPasswordReq) error {
	if err := auth.ValidatePasswordStrength(&s.cfg.Security, req.Password); err != nil {
		return errs.ErrPasswordWeak.WithMessage(err.Error())
	}
	tok, err := s.repos.Email.GetTokenByHash(ctx, auth.HashToken(strings.TrimSpace(req.Token)))
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if tok == nil || tok.Scene != model.EmailSceneReset || tok.UsedAt != nil || tok.ExpiresAt.Before(model.Now()) {
		return errs.ErrInvalidParam.WithMessage("重置链接无效或已过期")
	}
	hash, err := auth.HashPassword(req.Password, s.cfg.Security.BcryptCost)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if err := s.repos.User.UpdateFields(ctx, tok.UserID, map[string]any{"password_hash": hash}); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if err := s.repos.Email.MarkTokenUsed(ctx, tok.ID, model.Now()); err != nil && s.log != nil {
		s.log.Warn("mark reset token used failed", logger.Err(err))
	}
	return nil
}

// CheckLoginCaptcha 校验登录验证码（供 Handler 在特殊场景复用）。
func (s *AuthService) CheckLoginCaptcha(ctx context.Context, id, code string) error {
	if !s.captchaEnabled(ctx) {
		return nil
	}
	if !s.captcha.Verify(id, code) {
		return errs.ErrCaptchaWrong
	}
	return nil
}

func (s *AuthService) captchaEnabled(ctx context.Context) bool {
	if s.captcha == nil {
		return false
	}
	return s.cfg.Captcha.Enabled && s.settings.GetBool(ctx, KeyCaptchaEnabled, true)
}

// emailVerifyLevel 返回生效的邮箱验证级别，SMTP 未配置时 force 降级为 soft。
func (s *AuthService) emailVerifyLevel(ctx context.Context) string {
	level := strings.ToLower(s.settings.GetString(ctx, KeyRegisterRequireEmailVerify, s.cfg.Register.RequireEmailVerify))
	switch level {
	case "off", "soft", "force":
	default:
		level = "force"
	}
	if level == "force" && !s.email.Enabled() {
		level = "soft"
	}
	return level
}

func (s *AuthService) defaultGroupID(ctx context.Context) uint64 {
	if id := s.settings.GetInt(ctx, KeyRegisterDefaultGroupID, 0); id > 0 {
		return uint64(id)
	}
	if s.cfg.Register.DefaultGroupID > 0 {
		return s.cfg.Register.DefaultGroupID
	}
	return 1
}

func (s *AuthService) emailBlocked(email string) bool {
	at := strings.LastIndexByte(email, '@')
	if at < 0 {
		return false
	}
	domain := email[at+1:]
	for _, blocked := range s.cfg.Register.EmailDomainBlock {
		if strings.EqualFold(blocked, domain) {
			return true
		}
	}
	return false
}

func (s *AuthService) issueTokens(ctx context.Context, u *model.User) (*dto.AuthResp, error) {
	jti, err := auth.NewOpaqueToken(16)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	pair, err := s.jwt.GeneratePair(u, jti)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return &dto.AuthResp{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
		User:         dto.NewUserResp(u),
	}, nil
}

func ptrTime(t time.Time) *time.Time { return &t }

// expiredBefore 解析过期时间字符串（供签名校验复用）。
func expiredBefore(exp string) bool {
	n, err := strconv.ParseInt(exp, 10, 64)
	if err != nil {
		return true
	}
	return time.Now().Unix() > n
}
