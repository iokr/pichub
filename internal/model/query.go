package model

import "time"

// ImageQuery 图片列表查询条件。
type ImageQuery struct {
	UserID     uint64
	Keyword    string
	AlbumID    *uint64
	TagID      uint64
	Ext        string
	Referred   bool
	Status     *int8
	IsPublic   *bool
	Cursor     uint64 // 游标模式下返回 id < cursor 的记录
	Page       int
	Size       int
	TrashOnly  bool
	OrderByExt bool
}

// UserQuery 用户列表查询条件。
type UserQuery struct {
	Keyword string
	Status  *int8
	Role    *int8
	GroupID uint64
	Page    int
	Size    int
}

// LogQuery 操作日志查询条件。
type LogQuery struct {
	UserID uint64
	Action string
	Page   int
	Size   int
}

// EmailLogQuery 邮件日志查询条件。
type EmailLogQuery struct {
	UserID uint64
	Status *int8
	Page   int
	Size   int
}

// RefererQuery 引用列表查询条件。
type RefererQuery struct {
	ImageID uint64
	Page    int
	Size    int
	OrderBy string
}

// HostStat 域名汇总统计。
type HostStat struct {
	Host   string `gorm:"column:referer_host" json:"host"`
	Hits   int64  `gorm:"column:hits" json:"hits"`
	Images int64  `gorm:"column:images" json:"images"`
}

// DayStat 按天统计。
type DayStat struct {
	Date Date  `gorm:"column:stat_date" json:"date"`
	Hits int64 `gorm:"column:hits" json:"hits"`
}

// AdminStats 管理后台仪表盘数据。
type AdminStats struct {
	UserCount    int64 `json:"user_count"`
	ImageCount   int64 `json:"image_count"`
	StorageBytes int64 `json:"storage_bytes"`
	TodayUploads int64 `json:"today_uploads"`
	RefererHits  int64 `json:"referer_hits"`
	AlbumCount   int64 `json:"album_count"`
	ShareCount   int64 `json:"share_count"`
	TrashCount   int64 `json:"trash_count"`
}

// TimeRange 时间区间。
type TimeRange struct {
	From time.Time
	To   time.Time
}
