// Package model 定义数据库实体。本包不依赖任何 internal 包。
package model

import "time"

// 通用状态常量。
const (
	StatusDisabled = 0
	StatusEnabled  = 1
)

// 用户角色。
const (
	RoleUser  = 0
	RoleAdmin = 1
)

// 用户状态。
const (
	UserStatusDisabled = 0
	UserStatusNormal   = 1
	UserStatusPending  = 2
)

// 图片状态。
const (
	ImageStatusTrash  = 0
	ImageStatusNormal = 1
)

// 邮箱令牌场景。
const (
	EmailSceneVerify = "verify"
	EmailSceneReset  = "reset"
	EmailSceneChange = "change"
)

// 邮件发送状态。
const (
	EmailStatusPending = 0
	EmailStatusSuccess = 1
	EmailStatusFailed  = 2
)

// 分享目标类型。
const (
	ShareTargetImage = "image"
	ShareTargetAlbum = "album"
)

// Base 提供自增主键。
type Base struct {
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
}

// Timestamps 提供创建与更新时间。
type Timestamps struct {
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// Now 返回 UTC 当前时间。
func Now() time.Time { return time.Now().UTC() }
