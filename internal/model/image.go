package model

import "time"

// Album 相册。
type Album struct {
	Base
	UserID       uint64    `gorm:"column:user_id" json:"user_id"`
	Name         string    `gorm:"column:name;size:64" json:"name"`
	Description  string    `gorm:"column:description;size:255" json:"description"`
	CoverImageID *uint64   `gorm:"column:cover_image_id" json:"cover_image_id"`
	IsPublic     bool      `gorm:"column:is_public" json:"is_public"`
	SortOrder    int       `gorm:"column:sort_order" json:"sort_order"`
	ImageCount   int       `gorm:"column:image_count" json:"image_count"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 表名。
func (Album) TableName() string { return "albums" }

// Image 图片元数据。
type Image struct {
	Base
	UserID        uint64     `gorm:"column:user_id" json:"user_id"`
	AlbumID       *uint64    `gorm:"column:album_id" json:"album_id"`
	Hash          string     `gorm:"column:hash;size:64" json:"hash"`
	OriginName    string     `gorm:"column:origin_name;size:255" json:"origin_name"`
	DisplayName   string     `gorm:"column:display_name;size:255" json:"display_name"`
	Ext           string     `gorm:"column:ext;size:10" json:"ext"`
	Mime          string     `gorm:"column:mime;size:64" json:"mime"`
	Size          int64      `gorm:"column:size" json:"size"`
	Width         int        `gorm:"column:width" json:"width"`
	Height        int        `gorm:"column:height" json:"height"`
	StorageDriver string     `gorm:"column:storage_driver;size:20" json:"storage_driver"`
	StorageKey    string     `gorm:"column:storage_key;size:500" json:"-"`
	URL           string     `gorm:"column:url;size:768" json:"url"`
	ThumbKey      string     `gorm:"column:thumb_key;size:500" json:"-"`
	IsPublic      bool       `gorm:"column:is_public" json:"is_public"`
	RefCount      int        `gorm:"column:ref_count" json:"ref_count"`
	Status        int8       `gorm:"column:status" json:"status"`
	DeletedFlag   int64      `gorm:"column:deleted_flag" json:"-"`
	DeletedAt     *time.Time `gorm:"column:deleted_at" json:"deleted_at"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 表名。
func (Image) TableName() string { return "images" }

// Tag 标签。
type Tag struct {
	Base
	UserID     uint64    `gorm:"column:user_id" json:"user_id"`
	Name       string    `gorm:"column:name;size:32" json:"name"`
	Color      string    `gorm:"column:color;size:16" json:"color"`
	ImageCount int       `gorm:"column:image_count" json:"image_count"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 表名。
func (Tag) TableName() string { return "tags" }

// ImageTag 图片-标签关联。
type ImageTag struct {
	ImageID   uint64    `gorm:"column:image_id;primaryKey" json:"image_id"`
	TagID     uint64    `gorm:"column:tag_id;primaryKey" json:"tag_id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 表名。
func (ImageTag) TableName() string { return "image_tags" }

// Share 分享链接。
type Share struct {
	Base
	UserID       uint64     `gorm:"column:user_id" json:"user_id"`
	Slug         string     `gorm:"column:slug;size:16" json:"slug"`
	TargetType   string     `gorm:"column:target_type;size:10" json:"target_type"`
	TargetID     uint64     `gorm:"column:target_id" json:"target_id"`
	PasswordHash string     `gorm:"column:password_hash;size:100" json:"-"`
	ExpiresAt    *time.Time `gorm:"column:expires_at" json:"expires_at"`
	MaxViews     int        `gorm:"column:max_views" json:"max_views"`
	Views        int        `gorm:"column:views" json:"views"`
	Status       int8       `gorm:"column:status" json:"status"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 表名。
func (Share) TableName() string { return "shares" }

// HasPassword 是否设置访问密码。
func (s *Share) HasPassword() bool { return s.PasswordHash != "" }

// ImageReferer 图片引用来源。
type ImageReferer struct {
	Base
	ImageID     uint64    `gorm:"column:image_id" json:"image_id"`
	RefererHash string    `gorm:"column:referer_hash;size:64" json:"-"`
	RefererURL  string    `gorm:"column:referer_url;size:1024" json:"referer_url"`
	RefererHost string    `gorm:"column:referer_host;size:255" json:"referer_host"`
	HitCount    int64     `gorm:"column:hit_count" json:"hit_count"`
	FirstSeenAt time.Time `gorm:"column:first_seen_at" json:"first_seen_at"`
	LastSeenAt  time.Time `gorm:"column:last_seen_at" json:"last_seen_at"`
}

// TableName 表名。
func (ImageReferer) TableName() string { return "image_referers" }

// ImageRefererDaily 图片引用按天聚合。
type ImageRefererDaily struct {
	ImageID     uint64 `gorm:"column:image_id;primaryKey" json:"image_id"`
	StatDate    Date   `gorm:"column:stat_date;type:date;primaryKey" json:"stat_date"`
	RefererHost string `gorm:"column:referer_host;size:255;primaryKey" json:"referer_host"`
	HitCount    int64  `gorm:"column:hit_count" json:"hit_count"`
}

// TableName 表名。
func (ImageRefererDaily) TableName() string { return "image_referer_daily" }

// UploadCounter 每日上传用量。
type UploadCounter struct {
	UserID    uint64    `gorm:"column:user_id;primaryKey" json:"user_id"`
	StatDate  Date      `gorm:"column:stat_date;type:date;primaryKey" json:"stat_date"`
	Count     int       `gorm:"column:count" json:"count"`
	Bytes     int64     `gorm:"column:bytes" json:"bytes"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 表名。
func (UploadCounter) TableName() string { return "upload_counters" }

// OperationLog 操作审计日志。
type OperationLog struct {
	Base
	UserID     uint64    `gorm:"column:user_id" json:"user_id"`
	Action     string    `gorm:"column:action;size:48" json:"action"`
	TargetType string    `gorm:"column:target_type;size:24" json:"target_type"`
	TargetID   uint64    `gorm:"column:target_id" json:"target_id"`
	IP         string    `gorm:"column:ip;type:varbinary(16)" json:"-"`
	UserAgent  string    `gorm:"column:user_agent;size:255" json:"user_agent"`
	Detail     []byte    `gorm:"column:detail;type:json" json:"detail"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 表名。
func (OperationLog) TableName() string { return "operation_logs" }

// SystemSetting 系统全局配置。
type SystemSetting struct {
	Base
	CfgKey      string    `gorm:"column:cfg_key;size:64" json:"cfg_key"`
	CfgValue    string    `gorm:"column:cfg_value;type:text" json:"cfg_value"`
	ValueType   string    `gorm:"column:value_type;size:16" json:"value_type"`
	IsSecret    bool      `gorm:"column:is_secret" json:"is_secret"`
	Description string    `gorm:"column:description;size:255" json:"description"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 表名。
func (SystemSetting) TableName() string { return "system_settings" }

// Invite 注册邀请码。
type Invite struct {
	Base
	Code      string     `gorm:"column:code;size:32" json:"code"`
	CreatorID uint64     `gorm:"column:creator_id" json:"creator_id"`
	MaxUses   int        `gorm:"column:max_uses" json:"max_uses"`
	UsedCount int        `gorm:"column:used_count" json:"used_count"`
	GroupID   uint64     `gorm:"column:group_id" json:"group_id"`
	ExpiresAt *time.Time `gorm:"column:expires_at" json:"expires_at"`
	Status    int8       `gorm:"column:status" json:"status"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 表名。
func (Invite) TableName() string { return "invites" }
