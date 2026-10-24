// Package config 定义 PicHub 的全部配置结构，并负责 YAML 加载、校验与密钥自动生成。
package config

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// Config 是应用的全部配置，字段与 config.yaml 一一对应。
type Config struct {
	Site      SiteConfig      `mapstructure:"site" yaml:"site"`
	Server    ServerConfig    `mapstructure:"server" yaml:"server"`
	Database  DatabaseConfig  `mapstructure:"database" yaml:"database"`
	Redis     RedisConfig     `mapstructure:"redis" yaml:"redis"`
	Security  SecurityConfig  `mapstructure:"security" yaml:"security"`
	Register  RegisterConfig  `mapstructure:"register" yaml:"register"`
	Captcha   CaptchaConfig   `mapstructure:"captcha" yaml:"captcha"`
	Upload    UploadConfig    `mapstructure:"upload" yaml:"upload"`
	Storage   StorageConfig   `mapstructure:"storage" yaml:"storage"`
	Image     ImageConfig     `mapstructure:"image" yaml:"image"`
	Referer   RefererConfig   `mapstructure:"referer" yaml:"referer"`
	Mail      MailConfig      `mapstructure:"mail" yaml:"mail"`
	S3Gateway S3GatewayConfig `mapstructure:"s3_gateway" yaml:"s3_gateway"`
	Cron      CronConfig      `mapstructure:"cron" yaml:"cron"`
	Queue     QueueConfig     `mapstructure:"queue" yaml:"queue"`
	Migrate   MigrateConfig   `mapstructure:"migrate" yaml:"migrate"`
	Log       LogConfig       `mapstructure:"log" yaml:"log"`
	Monitor   MonitorConfig   `mapstructure:"monitor" yaml:"monitor"`

	loadedPath string
	warnings   []string
	// siteURL 保存运行时站点地址覆盖值（来自后台系统设置 site.url）。
	siteURL atomic.Value
}

// SiteConfig 站点信息。
type SiteConfig struct {
	Name        string `mapstructure:"name" yaml:"name"`
	URL         string `mapstructure:"url" yaml:"url"`
	BasePath    string `mapstructure:"base_path" yaml:"base_path"`
	Description string `mapstructure:"description" yaml:"description"`
	LogoURL     string `mapstructure:"logo_url" yaml:"logo_url"`
	ICP         string `mapstructure:"icp" yaml:"icp"`
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Addr           string        `mapstructure:"addr" yaml:"addr"`
	Mode           string        `mapstructure:"mode" yaml:"mode"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout" yaml:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout" yaml:"write_timeout"`
	IdleTimeout    time.Duration `mapstructure:"idle_timeout" yaml:"idle_timeout"`
	MaxHeaderBytes int           `mapstructure:"max_header_bytes" yaml:"max_header_bytes"`
	TrustedProxies []string      `mapstructure:"trusted_proxies" yaml:"trusted_proxies"`
}

// DatabaseConfig MySQL 配置。
type DatabaseConfig struct {
	Host            string        `mapstructure:"host" yaml:"host"`
	Port            int           `mapstructure:"port" yaml:"port"`
	Username        string        `mapstructure:"username" yaml:"username"`
	Password        string        `mapstructure:"password" yaml:"password"`
	Database        string        `mapstructure:"database" yaml:"database"`
	Charset         string        `mapstructure:"charset" yaml:"charset"`
	Collation       string        `mapstructure:"collation" yaml:"collation"`
	Loc             string        `mapstructure:"loc" yaml:"loc"`
	ParseTime       bool          `mapstructure:"parse_time" yaml:"parse_time"`
	MaxOpenConns    int           `mapstructure:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" yaml:"conn_max_idle_time"`
	SlowThreshold   time.Duration `mapstructure:"slow_threshold" yaml:"slow_threshold"`
	LogLevel        string        `mapstructure:"log_level" yaml:"log_level"`
}

// RedisConfig Redis 配置。
type RedisConfig struct {
	Addr         string        `mapstructure:"addr" yaml:"addr"`
	Password     string        `mapstructure:"password" yaml:"password"`
	DB           int           `mapstructure:"db" yaml:"db"`
	PoolSize     int           `mapstructure:"pool_size" yaml:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns" yaml:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout" yaml:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" yaml:"write_timeout"`
	KeyPrefix    string        `mapstructure:"key_prefix" yaml:"key_prefix"`
}

// RateLimitConfig 接口限流配置。
type RateLimitConfig struct {
	Enabled         bool `mapstructure:"enabled" yaml:"enabled"`
	LoginPerMinute  int  `mapstructure:"login_per_minute" yaml:"login_per_minute"`
	RegisterPerHour int  `mapstructure:"register_per_hour" yaml:"register_per_hour"`
	UploadPerMinute int  `mapstructure:"upload_per_minute" yaml:"upload_per_minute"`
	APIPerMinute    int  `mapstructure:"api_per_minute" yaml:"api_per_minute"`
	ImagePerMinute  int  `mapstructure:"image_per_minute" yaml:"image_per_minute"`
}

// SecurityConfig 安全相关配置。
type SecurityConfig struct {
	JWTSecret              string          `mapstructure:"jwt_secret" yaml:"jwt_secret"`
	SecretKey              string          `mapstructure:"secret_key" yaml:"secret_key"`
	JWTAccessTTL           time.Duration   `mapstructure:"jwt_access_ttl" yaml:"jwt_access_ttl"`
	JWTRefreshTTL          time.Duration   `mapstructure:"jwt_refresh_ttl" yaml:"jwt_refresh_ttl"`
	URLSignSecret          string          `mapstructure:"url_sign_secret" yaml:"url_sign_secret"`
	BcryptCost             int             `mapstructure:"bcrypt_cost" yaml:"bcrypt_cost"`
	PasswordMinLength      int             `mapstructure:"password_min_length" yaml:"password_min_length"`
	PasswordRequireLetter  bool            `mapstructure:"password_require_letter" yaml:"password_require_letter"`
	PasswordRequireDigit   bool            `mapstructure:"password_require_digit" yaml:"password_require_digit"`
	PasswordRequireSpecial bool            `mapstructure:"password_require_special" yaml:"password_require_special"`
	LoginMaxFails          int             `mapstructure:"login_max_fails" yaml:"login_max_fails"`
	LoginLockDuration      time.Duration   `mapstructure:"login_lock_duration" yaml:"login_lock_duration"`
	AutoGenerateSecrets    bool            `mapstructure:"auto_generate_secrets" yaml:"auto_generate_secrets"`
	RateLimit              RateLimitConfig `mapstructure:"rate_limit" yaml:"rate_limit"`
}

// RegisterConfig 注册策略。
type RegisterConfig struct {
	Enabled            bool     `mapstructure:"enabled" yaml:"enabled"`
	Mode               string   `mapstructure:"mode" yaml:"mode"`
	RequireEmailVerify string   `mapstructure:"require_email_verify" yaml:"require_email_verify"`
	DefaultGroupID     uint64   `mapstructure:"default_group_id" yaml:"default_group_id"`
	IPLimitPerDay      int      `mapstructure:"ip_limit_per_day" yaml:"ip_limit_per_day"`
	UsernameMinLength  int      `mapstructure:"username_min_length" yaml:"username_min_length"`
	UsernameMaxLength  int      `mapstructure:"username_max_length" yaml:"username_max_length"`
	UsernamePattern    string   `mapstructure:"username_pattern" yaml:"username_pattern"`
	ReservedUsernames  []string `mapstructure:"reserved_usernames" yaml:"reserved_usernames"`
	EmailDomainBlock   []string `mapstructure:"email_domain_blocklist" yaml:"email_domain_blocklist"`
}

// CaptchaConfig 图片验证码配置。
type CaptchaConfig struct {
	Enabled           bool   `mapstructure:"enabled" yaml:"enabled"`
	Type              string `mapstructure:"type" yaml:"type"`
	Length            int    `mapstructure:"length" yaml:"length"`
	Width             int    `mapstructure:"width" yaml:"width"`
	Height            int    `mapstructure:"height" yaml:"height"`
	NoiseCount        int    `mapstructure:"noise_count" yaml:"noise_count"`
	LineCount         int    `mapstructure:"line_count" yaml:"line_count"`
	ExpireSeconds     int    `mapstructure:"expire_seconds" yaml:"expire_seconds"`
	LoginTriggerFails int    `mapstructure:"login_trigger_fails" yaml:"login_trigger_fails"`
	StoreDriver       string `mapstructure:"store_driver" yaml:"store_driver"`
}

// UploadConfig 上传策略。
type UploadConfig struct {
	DefaultVisibility string   `mapstructure:"default_visibility" yaml:"default_visibility"`
	MaxFileSize       int64    `mapstructure:"max_file_size" yaml:"max_file_size"`
	AllowedExtensions []string `mapstructure:"allowed_extensions" yaml:"allowed_extensions"`
	AllowSVG          bool     `mapstructure:"allow_svg" yaml:"allow_svg"`
	ChunkSize         int64    `mapstructure:"chunk_size" yaml:"chunk_size"`
	MaxChunkCount     int      `mapstructure:"max_chunk_count" yaml:"max_chunk_count"`
	DedupEnabled      bool     `mapstructure:"dedup_enabled" yaml:"dedup_enabled"`
	DedupScope        string   `mapstructure:"dedup_scope" yaml:"dedup_scope"`
	KeepOriginName    bool     `mapstructure:"keep_origin_name" yaml:"keep_origin_name"`
	ConcurrentLimit   int      `mapstructure:"concurrent_limit" yaml:"concurrent_limit"`
}

// LocalStorageConfig 本地存储驱动配置。
type LocalStorageConfig struct {
	RootPath       string `mapstructure:"root_path" yaml:"root_path"`
	PathPrefix     string `mapstructure:"path_prefix" yaml:"path_prefix"`
	URLPrefix      string `mapstructure:"url_prefix" yaml:"url_prefix"`
	ThumbPath      string `mapstructure:"thumb_path" yaml:"thumb_path"`
	DirPermission  uint32 `mapstructure:"dir_permission" yaml:"dir_permission"`
	FilePermission uint32 `mapstructure:"file_permission" yaml:"file_permission"`
	PathTemplate   string `mapstructure:"path_template" yaml:"path_template"`
}

// ObjectStorageConfig 通用对象存储驱动配置。
type ObjectStorageConfig struct {
	Endpoint  string `mapstructure:"endpoint" yaml:"endpoint"`
	Region    string `mapstructure:"region" yaml:"region"`
	Bucket    string `mapstructure:"bucket" yaml:"bucket"`
	AccessKey string `mapstructure:"access_key" yaml:"access_key"`
	SecretKey string `mapstructure:"secret_key" yaml:"secret_key"`
	PathStyle bool   `mapstructure:"path_style" yaml:"path_style"`
	UseSSL    bool   `mapstructure:"use_ssl" yaml:"use_ssl"`
	PublicURL string `mapstructure:"public_url" yaml:"public_url"`
}

// WebDAVConfig WebDAV 驱动配置。
type WebDAVConfig struct {
	URL       string `mapstructure:"url" yaml:"url"`
	Username  string `mapstructure:"username" yaml:"username"`
	Password  string `mapstructure:"password" yaml:"password"`
	PublicURL string `mapstructure:"public_url" yaml:"public_url"`
}

// StorageConfig 存储总配置。
type StorageConfig struct {
	DefaultDriver       string              `mapstructure:"default_driver" yaml:"default_driver"`
	DefaultStrategyName string              `mapstructure:"default_strategy_name" yaml:"default_strategy_name"`
	AllowUserStrategy   bool                `mapstructure:"allow_user_strategy" yaml:"allow_user_strategy"`
	Local               LocalStorageConfig  `mapstructure:"local" yaml:"local"`
	S3                  ObjectStorageConfig `mapstructure:"s3" yaml:"s3"`
	OSS                 ObjectStorageConfig `mapstructure:"oss" yaml:"oss"`
	COS                 ObjectStorageConfig `mapstructure:"cos" yaml:"cos"`
	WebDAV              WebDAVConfig        `mapstructure:"webdav" yaml:"webdav"`
}

// ThumbSizeConfig 预设缩略图尺寸。
type ThumbSizeConfig struct {
	Name  string `mapstructure:"name" yaml:"name"`
	Width int    `mapstructure:"width" yaml:"width"`
}

// ThumbnailConfig 缩略图配置。
type ThumbnailConfig struct {
	Enabled bool              `mapstructure:"enabled" yaml:"enabled"`
	Sizes   []ThumbSizeConfig `mapstructure:"sizes" yaml:"sizes"`
	Format  string            `mapstructure:"format" yaml:"format"`
	Quality int               `mapstructure:"quality" yaml:"quality"`
}

// DynamicConfig 动态图片处理配置。
type DynamicConfig struct {
	Enabled        bool          `mapstructure:"enabled" yaml:"enabled"`
	MaxWidth       int           `mapstructure:"max_width" yaml:"max_width"`
	MaxHeight      int           `mapstructure:"max_height" yaml:"max_height"`
	DefaultQuality int           `mapstructure:"default_quality" yaml:"default_quality"`
	AllowedFormats []string      `mapstructure:"allowed_formats" yaml:"allowed_formats"`
	CacheEnabled   bool          `mapstructure:"cache_enabled" yaml:"cache_enabled"`
	CacheDir       string        `mapstructure:"cache_dir" yaml:"cache_dir"`
	CacheMaxSize   int64         `mapstructure:"cache_max_size" yaml:"cache_max_size"`
	CacheTTL       time.Duration `mapstructure:"cache_ttl" yaml:"cache_ttl"`
}

// WatermarkConfig 水印配置。
type WatermarkConfig struct {
	Enabled   bool    `mapstructure:"enabled" yaml:"enabled"`
	Type      string  `mapstructure:"type" yaml:"type"`
	Text      string  `mapstructure:"text" yaml:"text"`
	ImagePath string  `mapstructure:"image_path" yaml:"image_path"`
	Position  string  `mapstructure:"position" yaml:"position"`
	Opacity   float64 `mapstructure:"opacity" yaml:"opacity"`
	FontSize  float64 `mapstructure:"font_size" yaml:"font_size"`
	Color     string  `mapstructure:"color" yaml:"color"`
}

// AntiLeechConfig 防盗链配置。
type AntiLeechConfig struct {
	Enabled           bool     `mapstructure:"enabled" yaml:"enabled"`
	AllowEmptyReferer bool     `mapstructure:"allow_empty_referer" yaml:"allow_empty_referer"`
	RefererWhitelist  []string `mapstructure:"referer_whitelist" yaml:"referer_whitelist"`
	BlockAction       string   `mapstructure:"block_action" yaml:"block_action"`
}

// ImageConfig 图片处理总配置。
type ImageConfig struct {
	Thumbnail ThumbnailConfig `mapstructure:"thumbnail" yaml:"thumbnail"`
	Dynamic   DynamicConfig   `mapstructure:"dynamic" yaml:"dynamic"`
	Watermark WatermarkConfig `mapstructure:"watermark" yaml:"watermark"`
	AntiLeech AntiLeechConfig `mapstructure:"anti_leech" yaml:"anti_leech"`
}

// RefererConfig 引用统计配置。
type RefererConfig struct {
	Enabled         bool          `mapstructure:"enabled" yaml:"enabled"`
	MaxPerImage     int           `mapstructure:"max_per_image" yaml:"max_per_image"`
	BufferSize      int           `mapstructure:"buffer_size" yaml:"buffer_size"`
	FlushInterval   time.Duration `mapstructure:"flush_interval" yaml:"flush_interval"`
	DailyRetainDays int           `mapstructure:"daily_retain_days" yaml:"daily_retain_days"`
	UserCanDisable  bool          `mapstructure:"user_can_disable" yaml:"user_can_disable"`
	ExcludeBots     bool          `mapstructure:"exclude_bots" yaml:"exclude_bots"`
	BotUserAgents   []string      `mapstructure:"bot_user_agents" yaml:"bot_user_agents"`
}

// MailConfig SMTP 配置。
type MailConfig struct {
	Enabled    bool          `mapstructure:"enabled" yaml:"enabled"`
	Host       string        `mapstructure:"host" yaml:"host"`
	Port       int           `mapstructure:"port" yaml:"port"`
	Username   string        `mapstructure:"username" yaml:"username"`
	Password   string        `mapstructure:"password" yaml:"password"`
	FromName   string        `mapstructure:"from_name" yaml:"from_name"`
	FromEmail  string        `mapstructure:"from_email" yaml:"from_email"`
	Encryption string        `mapstructure:"encryption" yaml:"encryption"`
	DailyLimit int           `mapstructure:"daily_limit" yaml:"daily_limit"`
	Timeout    time.Duration `mapstructure:"timeout" yaml:"timeout"`
	RetryTimes int           `mapstructure:"retry_times" yaml:"retry_times"`
}

// S3GatewayConfig S3 兼容网关配置。
type S3GatewayConfig struct {
	Enabled        bool          `mapstructure:"enabled" yaml:"enabled"`
	PathPrefix     string        `mapstructure:"path_prefix" yaml:"path_prefix"`
	Region         string        `mapstructure:"region" yaml:"region"`
	ClockSkew      time.Duration `mapstructure:"clock_skew" yaml:"clock_skew"`
	ReplayWindow   time.Duration `mapstructure:"replay_window" yaml:"replay_window"`
	AllowMultipart bool          `mapstructure:"allow_multipart" yaml:"allow_multipart"`
	PartSize       int64         `mapstructure:"part_size" yaml:"part_size"`
}

// TrashCleanupConfig 回收站清理配置。
type TrashCleanupConfig struct {
	Enabled    bool   `mapstructure:"enabled" yaml:"enabled"`
	Schedule   string `mapstructure:"schedule" yaml:"schedule"`
	RetainDays int    `mapstructure:"retain_days" yaml:"retain_days"`
}

// SimpleCronConfig 简单定时任务配置。
type SimpleCronConfig struct {
	Enabled  bool   `mapstructure:"enabled" yaml:"enabled"`
	Schedule string `mapstructure:"schedule" yaml:"schedule"`
}

// RetainCronConfig 带保留天数的定时任务配置。
type RetainCronConfig struct {
	Enabled    bool   `mapstructure:"enabled" yaml:"enabled"`
	Schedule   string `mapstructure:"schedule" yaml:"schedule"`
	RetainDays int    `mapstructure:"retain_days" yaml:"retain_days"`
}

// CronConfig 定时任务总配置。
type CronConfig struct {
	Enabled             bool               `mapstructure:"enabled" yaml:"enabled"`
	TrashCleanup        TrashCleanupConfig `mapstructure:"trash_cleanup" yaml:"trash_cleanup"`
	RefererAggregate    SimpleCronConfig   `mapstructure:"referer_aggregate" yaml:"referer_aggregate"`
	CacheCleanup        SimpleCronConfig   `mapstructure:"cache_cleanup" yaml:"cache_cleanup"`
	EmailLogCleanup     RetainCronConfig   `mapstructure:"email_log_cleanup" yaml:"email_log_cleanup"`
	OperationLogCleanup RetainCronConfig   `mapstructure:"operation_log_cleanup" yaml:"operation_log_cleanup"`
}

// QueueConfig 异步队列配置。
type QueueConfig struct {
	WorkerCount int           `mapstructure:"worker_count" yaml:"worker_count"`
	MaxRetry    int           `mapstructure:"max_retry" yaml:"max_retry"`
	RetryDelay  time.Duration `mapstructure:"retry_delay" yaml:"retry_delay"`
	TaskTimeout time.Duration `mapstructure:"task_timeout" yaml:"task_timeout"`
}

// MigrateConfig 迁移配置。
type MigrateConfig struct {
	OnStart bool `mapstructure:"on_start" yaml:"on_start"`
}

// LogConfig 日志配置。
type LogConfig struct {
	Level      string `mapstructure:"level" yaml:"level"`
	Format     string `mapstructure:"format" yaml:"format"`
	Output     string `mapstructure:"output" yaml:"output"`
	FilePath   string `mapstructure:"file_path" yaml:"file_path"`
	MaxSize    int    `mapstructure:"max_size" yaml:"max_size"`
	MaxBackups int    `mapstructure:"max_backups" yaml:"max_backups"`
	MaxAge     int    `mapstructure:"max_age" yaml:"max_age"`
	Compress   bool   `mapstructure:"compress" yaml:"compress"`
	Color      bool   `mapstructure:"color" yaml:"color"`
	Caller     bool   `mapstructure:"caller" yaml:"caller"`
}

// MonitorConfig 监控配置。
type MonitorConfig struct {
	PprofEnabled   bool   `mapstructure:"pprof_enabled" yaml:"pprof_enabled"`
	PprofAddr      string `mapstructure:"pprof_addr" yaml:"pprof_addr"`
	MetricsEnabled bool   `mapstructure:"metrics_enabled" yaml:"metrics_enabled"`
	MetricsPath    string `mapstructure:"metrics_path" yaml:"metrics_path"`
}

// LoadedPath 返回本次启动实际加载的配置文件绝对路径，未找到文件时返回空串。
func (c *Config) LoadedPath() string { return c.loadedPath }

// SetLoadedPath 记录配置文件路径。
func (c *Config) SetLoadedPath(p string) { c.loadedPath = p }

// Warnings 返回加载过程中产生的警告（如文件权限过宽）。
func (c *Config) Warnings() []string { return c.warnings }

func (c *Config) warnf(format string, args ...any) {
	c.warnings = append(c.warnings, fmt.Sprintf(format, args...))
}

// SiteURL 返回去掉结尾斜杠的站点地址。
// 优先使用后台系统设置覆盖值，未设置时回退到 config.yaml 的 site.url。
func (c *Config) SiteURL() string {
	if v, ok := c.siteURL.Load().(string); ok && v != "" {
		return v
	}
	u := c.Site.URL
	for len(u) > 0 && u[len(u)-1] == '/' {
		u = u[:len(u)-1]
	}
	return u
}

// SetSiteURL 设置运行时站点地址覆盖值，传空字符串表示回退到配置文件。
func (c *Config) SetSiteURL(u string) {
	c.siteURL.Store(strings.TrimRight(strings.TrimSpace(u), "/"))
}
