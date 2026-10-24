package model

import "time"

// UserGroup 用户组：配额与权限策略。
type UserGroup struct {
	Base
	Name               string    `gorm:"column:name;size:32" json:"name"`
	Description        string    `gorm:"column:description;size:255" json:"description"`
	QuotaBytes         int64     `gorm:"column:quota_bytes" json:"quota_bytes"`
	MaxFileSize        int64     `gorm:"column:max_file_size" json:"max_file_size"`
	DailyLimit         int       `gorm:"column:daily_limit" json:"daily_limit"`
	MaxAlbums          int       `gorm:"column:max_albums" json:"max_albums"`
	MaxTokens          int       `gorm:"column:max_tokens" json:"max_tokens"`
	MaxShares          int       `gorm:"column:max_shares" json:"max_shares"`
	AllowRegister      bool      `gorm:"column:allow_register" json:"allow_register"`
	AllowCustomStorage bool      `gorm:"column:allow_custom_storage" json:"allow_custom_storage"`
	IsDefault          bool      `gorm:"column:is_default" json:"is_default"`
	Status             int8      `gorm:"column:status" json:"status"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 表名。
func (UserGroup) TableName() string { return "user_groups" }

// User 用户账号。
type User struct {
	Base
	Username           string     `gorm:"column:username;size:32" json:"username"`
	Email              string     `gorm:"column:email;size:128" json:"email"`
	PasswordHash       string     `gorm:"column:password_hash;size:100" json:"-"`
	Nickname           string     `gorm:"column:nickname;size:32" json:"nickname"`
	AvatarURL          string     `gorm:"column:avatar_url;size:500" json:"avatar_url"`
	Role               int8       `gorm:"column:role" json:"role"`
	Status             int8       `gorm:"column:status" json:"status"`
	EmailVerified      bool       `gorm:"column:email_verified" json:"email_verified"`
	GroupID            uint64     `gorm:"column:group_id" json:"group_id"`
	QuotaBytes         int64      `gorm:"column:quota_bytes" json:"quota_bytes"`
	UsedBytes          int64      `gorm:"column:used_bytes" json:"used_bytes"`
	MaxFileSize        int64      `gorm:"column:max_file_size" json:"max_file_size"`
	DailyLimit         int        `gorm:"column:daily_limit" json:"daily_limit"`
	DefaultVisibility  string     `gorm:"column:default_visibility;size:10" json:"default_visibility"`
	StorageConfigID    *uint64    `gorm:"column:storage_config_id" json:"storage_config_id"`
	RefererStatEnabled bool       `gorm:"column:referer_stat_enabled" json:"referer_stat_enabled"`
	RegisterIP         string     `gorm:"column:register_ip;type:varbinary(16)" json:"-"`
	LastLoginIP        string     `gorm:"column:last_login_ip;type:varbinary(16)" json:"-"`
	LastLoginAt        *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 表名。
func (User) TableName() string { return "users" }

// IsAdmin 是否管理员。
func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

// EmailToken 邮箱令牌。
type EmailToken struct {
	Base
	UserID    uint64     `gorm:"column:user_id" json:"user_id"`
	Email     string     `gorm:"column:email;size:128" json:"email"`
	TokenHash string     `gorm:"column:token_hash;size:64" json:"-"`
	Scene     string     `gorm:"column:scene;size:20" json:"scene"`
	ExpiresAt time.Time  `gorm:"column:expires_at" json:"expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at" json:"used_at"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 表名。
func (EmailToken) TableName() string { return "email_tokens" }

// EmailLog 邮件发送日志。
type EmailLog struct {
	Base
	UserID    uint64     `gorm:"column:user_id" json:"user_id"`
	ToEmail   string     `gorm:"column:to_email;size:128" json:"to_email"`
	Subject   string     `gorm:"column:subject;size:255" json:"subject"`
	Template  string     `gorm:"column:template;size:32" json:"template"`
	Status    int8       `gorm:"column:status" json:"status"`
	Error     string     `gorm:"column:error;size:500" json:"error"`
	Retry     int8       `gorm:"column:retry" json:"retry"`
	SentAt    *time.Time `gorm:"column:sent_at" json:"sent_at"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 表名。
func (EmailLog) TableName() string { return "email_logs" }

// APIToken API 访问令牌。
type APIToken struct {
	Base
	UserID      uint64     `gorm:"column:user_id" json:"user_id"`
	Name        string     `gorm:"column:name;size:64" json:"name"`
	TokenHash   string     `gorm:"column:token_hash;size:64" json:"-"`
	TokenPrefix string     `gorm:"column:token_prefix;size:12" json:"token_prefix"`
	Scopes      string     `gorm:"column:scopes;size:255" json:"scopes"`
	ExpiresAt   *time.Time `gorm:"column:expires_at" json:"expires_at"`
	LastUsedAt  *time.Time `gorm:"column:last_used_at" json:"last_used_at"`
	LastUsedIP  string     `gorm:"column:last_used_ip;type:varbinary(16)" json:"-"`
	CallCount   int64      `gorm:"column:call_count" json:"call_count"`
	Status      int8       `gorm:"column:status" json:"status"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 表名。
func (APIToken) TableName() string { return "api_tokens" }

// S3Credential S3 兼容凭据。
type S3Credential struct {
	Base
	UserID     uint64     `gorm:"column:user_id" json:"user_id"`
	AccessKey  string     `gorm:"column:access_key;size:32" json:"access_key"`
	SecretEnc  []byte     `gorm:"column:secret_enc;type:varbinary(256)" json:"-"`
	Name       string     `gorm:"column:name;size:64" json:"name"`
	Status     int8       `gorm:"column:status" json:"status"`
	LastUsedAt *time.Time `gorm:"column:last_used_at" json:"last_used_at"`
	LastUsedIP string     `gorm:"column:last_used_ip;type:varbinary(16)" json:"-"`
	CallCount  int64      `gorm:"column:call_count" json:"call_count"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 表名。
func (S3Credential) TableName() string { return "s3_credentials" }

// StorageConfig 存储策略配置。
type StorageConfig struct {
	Base
	UserID     *uint64    `gorm:"column:user_id" json:"user_id"`
	Name       string     `gorm:"column:name;size:64" json:"name"`
	Driver     string     `gorm:"column:driver;size:20" json:"driver"`
	Config     []byte     `gorm:"column:config;type:json" json:"-"`
	Secret     []byte     `gorm:"column:secret;type:varbinary(1024)" json:"-"`
	Domain     string     `gorm:"column:domain;size:255" json:"domain"`
	PathPrefix string     `gorm:"column:path_prefix;size:128" json:"path_prefix"`
	IsDefault  bool       `gorm:"column:is_default" json:"is_default"`
	Status     int8       `gorm:"column:status" json:"status"`
	LastTestAt *time.Time `gorm:"column:last_test_at" json:"last_test_at"`
	LastTestOK *bool      `gorm:"column:last_test_ok" json:"last_test_ok"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 表名。
func (StorageConfig) TableName() string { return "storage_configs" }
