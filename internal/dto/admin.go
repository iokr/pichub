package dto

import (
	"encoding/json"
	"time"

	"github.com/pichub/pichub/internal/model"
)

// AdminUserResp 是管理端用户视图。
type AdminUserResp struct {
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
	RefererStatEnabled bool       `json:"referer_stat_enabled"`
	RegisterIP         string     `json:"register_ip"`
	LastLoginIP        string     `json:"last_login_ip"`
	LastLoginAt        *time.Time `json:"last_login_at"`
	CreatedAt          time.Time  `json:"created_at"`
}

// NewAdminUserResp 由模型构造管理端用户视图。
func NewAdminUserResp(u *model.User) *AdminUserResp {
	if u == nil {
		return nil
	}
	return &AdminUserResp{
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
		RefererStatEnabled: u.RefererStatEnabled,
		RegisterIP:         u.RegisterIP,
		LastLoginIP:        u.LastLoginIP,
		LastLoginAt:        u.LastLoginAt,
		CreatedAt:          u.CreatedAt,
	}
}

// NewAdminUserRespList 批量构造管理端用户视图。
func NewAdminUserRespList(list []*model.User) []*AdminUserResp {
	out := make([]*AdminUserResp, 0, len(list))
	for _, u := range list {
		out = append(out, NewAdminUserResp(u))
	}
	return out
}

// UpdateUserStatusReq 是启用/禁用用户请求。
type UpdateUserStatusReq struct {
	Status int8   `json:"status"`
	Reason string `json:"reason"`
}

// UpdateUserGroupReq 是调整用户组及配额请求。
type UpdateUserGroupReq struct {
	GroupID     uint64 `json:"group_id" binding:"required"`
	QuotaBytes  *int64 `json:"quota_bytes"`
	MaxFileSize *int64 `json:"max_file_size"`
	DailyLimit  *int   `json:"daily_limit"`
}

// ResetUserPasswordReq 是管理员重置密码请求。
type ResetUserPasswordReq struct {
	NewPassword string `json:"new_password"`
}

// GroupResp 是用户组响应。
type GroupResp struct {
	ID                 uint64    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	QuotaBytes         int64     `json:"quota_bytes"`
	MaxFileSize        int64     `json:"max_file_size"`
	DailyLimit         int       `json:"daily_limit"`
	MaxAlbums          int       `json:"max_albums"`
	MaxTokens          int       `json:"max_tokens"`
	MaxShares          int       `json:"max_shares"`
	AllowRegister      bool      `json:"allow_register"`
	AllowCustomStorage bool      `json:"allow_custom_storage"`
	IsDefault          bool      `json:"is_default"`
	Status             int8      `json:"status"`
	UserCount          int64     `json:"user_count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// NewGroupResp 由模型构造用户组响应。
func NewGroupResp(g *model.UserGroup) *GroupResp {
	if g == nil {
		return nil
	}
	return &GroupResp{
		ID:                 g.ID,
		Name:               g.Name,
		Description:        g.Description,
		QuotaBytes:         g.QuotaBytes,
		MaxFileSize:        g.MaxFileSize,
		DailyLimit:         g.DailyLimit,
		MaxAlbums:          g.MaxAlbums,
		MaxTokens:          g.MaxTokens,
		MaxShares:          g.MaxShares,
		AllowRegister:      g.AllowRegister,
		AllowCustomStorage: g.AllowCustomStorage,
		IsDefault:          g.IsDefault,
		Status:             g.Status,
		CreatedAt:          g.CreatedAt,
		UpdatedAt:          g.UpdatedAt,
	}
}

// GroupCreateReq 是创建用户组请求。
type GroupCreateReq struct {
	Name               string `json:"name" binding:"required"`
	Description        string `json:"description"`
	QuotaBytes         int64  `json:"quota_bytes"`
	MaxFileSize        int64  `json:"max_file_size"`
	DailyLimit         int    `json:"daily_limit"`
	MaxAlbums          int    `json:"max_albums"`
	MaxTokens          int    `json:"max_tokens"`
	MaxShares          int    `json:"max_shares"`
	AllowRegister      *bool  `json:"allow_register"`
	AllowCustomStorage *bool  `json:"allow_custom_storage"`
	IsDefault          bool   `json:"is_default"`
	Status             *int8  `json:"status"`
}

// GroupUpdateReq 是更新用户组请求。
type GroupUpdateReq struct {
	Name               *string `json:"name"`
	Description        *string `json:"description"`
	QuotaBytes         *int64  `json:"quota_bytes"`
	MaxFileSize        *int64  `json:"max_file_size"`
	DailyLimit         *int    `json:"daily_limit"`
	MaxAlbums          *int    `json:"max_albums"`
	MaxTokens          *int    `json:"max_tokens"`
	MaxShares          *int    `json:"max_shares"`
	AllowRegister      *bool   `json:"allow_register"`
	AllowCustomStorage *bool   `json:"allow_custom_storage"`
	IsDefault          *bool   `json:"is_default"`
	Status             *int8   `json:"status"`
}

// SettingResp 是系统设置项响应。
type SettingResp struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Type        string    `json:"type"`
	IsSecret    bool      `json:"is_secret"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewSettingResp 由模型构建设置项响应（密文值仅返回掩码）。
func NewSettingResp(s *model.SystemSetting) *SettingResp {
	if s == nil {
		return nil
	}
	value := s.CfgValue
	if s.IsSecret && value != "" {
		value = "******"
	}
	return &SettingResp{
		Key:         s.CfgKey,
		Value:       value,
		Type:        s.ValueType,
		IsSecret:    s.IsSecret,
		Description: s.Description,
		UpdatedAt:   s.UpdatedAt,
	}
}

// UpdateSettingsReq 是批量更新系统设置请求。
type UpdateSettingsReq struct {
	Settings map[string]string `json:"settings" binding:"required"`
}

// LogResp 是操作日志响应。
type LogResp struct {
	ID         uint64          `json:"id"`
	UserID     uint64          `json:"user_id"`
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   uint64          `json:"target_id"`
	IP         string          `json:"ip"`
	UserAgent  string          `json:"user_agent"`
	Detail     json.RawMessage `json:"detail,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// NewLogResp 由模型构造操作日志响应。
func NewLogResp(l *model.OperationLog) *LogResp {
	if l == nil {
		return nil
	}
	return &LogResp{
		ID:         l.ID,
		UserID:     l.UserID,
		Action:     l.Action,
		TargetType: l.TargetType,
		TargetID:   l.TargetID,
		IP:         l.IP,
		UserAgent:  l.UserAgent,
		Detail:     json.RawMessage(l.Detail),
		CreatedAt:  l.CreatedAt,
	}
}

// NewLogRespList 批量构造操作日志响应。
func NewLogRespList(list []*model.OperationLog) []*LogResp {
	out := make([]*LogResp, 0, len(list))
	for _, l := range list {
		out = append(out, NewLogResp(l))
	}
	return out
}

// InviteResp 是邀请码响应。
type InviteResp struct {
	ID        uint64     `json:"id"`
	Code      string     `json:"code"`
	CreatorID uint64     `json:"creator_id"`
	MaxUses   int        `json:"max_uses"`
	UsedCount int        `json:"used_count"`
	GroupID   uint64     `json:"group_id"`
	ExpiresAt *time.Time `json:"expires_at"`
	Status    int8       `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

// NewInviteResp 由模型构造邀请码响应。
func NewInviteResp(i *model.Invite) *InviteResp {
	if i == nil {
		return nil
	}
	return &InviteResp{
		ID:        i.ID,
		Code:      i.Code,
		CreatorID: i.CreatorID,
		MaxUses:   i.MaxUses,
		UsedCount: i.UsedCount,
		GroupID:   i.GroupID,
		ExpiresAt: i.ExpiresAt,
		Status:    i.Status,
		CreatedAt: i.CreatedAt,
	}
}

// InviteCreateReq 是创建邀请码请求。
type InviteCreateReq struct {
	MaxUses        int    `json:"max_uses"`
	GroupID        uint64 `json:"group_id"`
	ExpiresInHours int    `json:"expires_in_hours"`
	Count          int    `json:"count"`
}

// EmailLogResp 是邮件日志响应。
type EmailLogResp struct {
	ID        uint64     `json:"id"`
	UserID    uint64     `json:"user_id"`
	ToEmail   string     `json:"to_email"`
	Subject   string     `json:"subject"`
	Template  string     `json:"template"`
	Status    int8       `json:"status"`
	Error     string     `json:"error"`
	Retry     int8       `json:"retry"`
	SentAt    *time.Time `json:"sent_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// NewEmailLogResp 由模型构造邮件日志响应。
func NewEmailLogResp(l *model.EmailLog) *EmailLogResp {
	if l == nil {
		return nil
	}
	return &EmailLogResp{
		ID:        l.ID,
		UserID:    l.UserID,
		ToEmail:   l.ToEmail,
		Subject:   l.Subject,
		Template:  l.Template,
		Status:    l.Status,
		Error:     l.Error,
		Retry:     l.Retry,
		SentAt:    l.SentAt,
		CreatedAt: l.CreatedAt,
	}
}

// NewEmailLogRespList 批量构造邮件日志响应。
func NewEmailLogRespList(list []*model.EmailLog) []*EmailLogResp {
	out := make([]*EmailLogResp, 0, len(list))
	for _, l := range list {
		out = append(out, NewEmailLogResp(l))
	}
	return out
}

// DashboardResp 是管理后台仪表盘响应。
type DashboardResp struct {
	model.AdminStats
	GroupCount   int64 `json:"group_count"`
	InviteCount  int64 `json:"invite_count"`
	QueueReady   int64 `json:"queue_ready"`
	QueueDelayed int64 `json:"queue_delayed"`
	QueueDead    int64 `json:"queue_dead"`
}
