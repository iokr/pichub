package dto

import (
	"time"

	"github.com/pichub/pichub/internal/model"
)

// UserResp 是用户信息响应。
type UserResp struct {
	ID                 uint64     `json:"id"`
	Username           string     `json:"username"`
	Email              string     `json:"email"`
	Nickname           string     `json:"nickname"`
	AvatarURL          string     `json:"avatar_url"`
	Role               int8       `json:"role"`
	Status             int8       `json:"status"`
	EmailVerified      bool       `json:"email_verified"`
	GroupID            uint64     `json:"group_id"`
	QuotaBytes         int64      `json:"quota_bytes"`
	UsedBytes          int64      `json:"used_bytes"`
	MaxFileSize        int64      `json:"max_file_size"`
	DailyLimit         int        `json:"daily_limit"`
	DefaultVisibility  string     `json:"default_visibility"`
	StorageConfigID    *uint64    `json:"storage_config_id"`
	RefererStatEnabled bool       `json:"referer_stat_enabled"`
	LastLoginAt        *time.Time `json:"last_login_at"`
	CreatedAt          time.Time  `json:"created_at"`
}

// NewUserResp 由模型构造用户响应。
func NewUserResp(u *model.User) *UserResp {
	if u == nil {
		return nil
	}
	return &UserResp{
		ID:                 u.ID,
		Username:           u.Username,
		Email:              u.Email,
		Nickname:           u.Nickname,
		AvatarURL:          u.AvatarURL,
		Role:               u.Role,
		Status:             u.Status,
		EmailVerified:      u.EmailVerified,
		GroupID:            u.GroupID,
		QuotaBytes:         u.QuotaBytes,
		UsedBytes:          u.UsedBytes,
		MaxFileSize:        u.MaxFileSize,
		DailyLimit:         u.DailyLimit,
		DefaultVisibility:  u.DefaultVisibility,
		StorageConfigID:    u.StorageConfigID,
		RefererStatEnabled: u.RefererStatEnabled,
		LastLoginAt:        u.LastLoginAt,
		CreatedAt:          u.CreatedAt,
	}
}

// CaptchaResp 是验证码响应。
type CaptchaResp struct {
	CaptchaID   string `json:"captcha_id"`
	ImageBase64 string `json:"image_base64"`
	ExpiresIn   int    `json:"expires_in"`
}

// RegisterReq 是注册请求。
type RegisterReq struct {
	Username    string `json:"username" binding:"required"`
	Email       string `json:"email" binding:"required"`
	Password    string `json:"password" binding:"required"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
	InviteCode  string `json:"invite_code"`
	IP          string `json:"-"`
}

// LoginReq 是登录请求。
type LoginReq struct {
	Account     string `json:"account"`
	Username    string `json:"username"`
	Password    string `json:"password" binding:"required"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
	IP          string `json:"-"`
}

// LoginName 返回登录账号（用户名或邮箱）。
func (r *LoginReq) LoginName() string {
	if r.Account != "" {
		return r.Account
	}
	return r.Username
}

// RefreshReq 是刷新令牌请求。
type RefreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutReq 是登出请求。
type LogoutReq struct {
	RefreshToken string `json:"refresh_token"`
}

// AuthResp 是登录/注册响应。
type AuthResp struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"`
	User         *UserResp `json:"user"`
}

// VerifyEmailReq 是邮箱验证请求。
type VerifyEmailReq struct {
	Token string `json:"token" binding:"required"`
}

// ResendVerifyReq 是重发验证邮件请求。
type ResendVerifyReq struct {
	Email       string `json:"email" binding:"required"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

// ForgotPasswordReq 是忘记密码请求。
type ForgotPasswordReq struct {
	Email       string `json:"email" binding:"required"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

// ResetPasswordReq 是重置密码请求。
type ResetPasswordReq struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// ChangePasswordReq 是修改密码请求。
type ChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

// UpdateProfileReq 是修改资料请求，指针字段表示可选更新。
type UpdateProfileReq struct {
	Nickname           *string `json:"nickname"`
	AvatarURL          *string `json:"avatar_url"`
	DefaultVisibility  *string `json:"default_visibility"`
	RefererStatEnabled *bool   `json:"referer_stat_enabled"`
	StorageConfigID    *uint64 `json:"storage_config_id"`
}

// CheckResp 是可用性检查响应。
type CheckResp struct {
	Available bool   `json:"available"`
	Message   string `json:"message"`
}

// UserStatsResp 是用户用量统计。
type UserStatsResp struct {
	ImageCount   int64             `json:"image_count"`
	AlbumCount   int64             `json:"album_count"`
	ShareCount   int64             `json:"share_count"`
	TrashCount   int64             `json:"trash_count"`
	StorageBytes int64             `json:"storage_bytes"`
	UsedBytes    int64             `json:"used_bytes"`
	QuotaBytes   int64             `json:"quota_bytes"`
	DailyLimit   int               `json:"daily_limit"`
	TodayUploads int               `json:"today_uploads"`
	RefererHits  int64             `json:"referer_hits"`
	TopHosts     []*model.HostStat `json:"top_hosts"`
	Trend        []*model.DayStat  `json:"trend"`
}
