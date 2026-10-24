package dto

import (
	"strings"
	"time"

	"github.com/pichub/pichub/internal/model"
)

// AlbumResp 是相册响应。
type AlbumResp struct {
	ID           uint64    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	CoverImageID *uint64   `json:"cover_image_id"`
	CoverURL     string    `json:"cover_url"`
	IsPublic     bool      `json:"is_public"`
	SortOrder    int       `json:"sort_order"`
	ImageCount   int       `json:"image_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NewAlbumResp 由模型构造相册响应。
func NewAlbumResp(a *model.Album) *AlbumResp {
	if a == nil {
		return nil
	}
	return &AlbumResp{
		ID:           a.ID,
		Name:         a.Name,
		Description:  a.Description,
		CoverImageID: a.CoverImageID,
		IsPublic:     a.IsPublic,
		SortOrder:    a.SortOrder,
		ImageCount:   a.ImageCount,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
}

// AlbumCreateReq 是创建相册请求。
type AlbumCreateReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IsPublic    *bool  `json:"is_public"`
}

// AlbumUpdateReq 是更新相册请求。
type AlbumUpdateReq struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	IsPublic     *bool   `json:"is_public"`
	CoverImageID *uint64 `json:"cover_image_id"`
	SortOrder    *int    `json:"sort_order"`
}

// TagResp 是标签响应。
type TagResp struct {
	ID         uint64    `json:"id"`
	Name       string    `json:"name"`
	Color      string    `json:"color"`
	ImageCount int       `json:"image_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewTagResp 由模型构造标签响应。
func NewTagResp(t *model.Tag) *TagResp {
	if t == nil {
		return nil
	}
	return &TagResp{ID: t.ID, Name: t.Name, Color: t.Color, ImageCount: t.ImageCount, CreatedAt: t.CreatedAt}
}

// TagCreateReq 是创建标签请求。
type TagCreateReq struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color"`
}

// TagUpdateReq 是更新标签请求。
type TagUpdateReq struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

// TokenResp 是 API Token 响应。
type TokenResp struct {
	ID         uint64     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CallCount  int64      `json:"call_count"`
	Status     int8       `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
}

// NewTokenResp 由模型构造令牌响应。
func NewTokenResp(t *model.APIToken) *TokenResp {
	if t == nil {
		return nil
	}
	var scopes []string
	for _, s := range strings.Split(t.Scopes, ",") {
		if s = strings.TrimSpace(s); s != "" {
			scopes = append(scopes, s)
		}
	}
	return &TokenResp{
		ID:         t.ID,
		Name:       t.Name,
		Prefix:     t.TokenPrefix,
		Scopes:     scopes,
		ExpiresAt:  t.ExpiresAt,
		LastUsedAt: t.LastUsedAt,
		CallCount:  t.CallCount,
		Status:     t.Status,
		CreatedAt:  t.CreatedAt,
	}
}

// TokenCreateReq 是创建令牌请求。
type TokenCreateReq struct {
	Name          string   `json:"name" binding:"required"`
	Scopes        []string `json:"scopes"`
	ExpiresInDays int      `json:"expires_in_days"`
}

// TokenCreateResp 是创建令牌响应，明文仅返回一次。
type TokenCreateResp struct {
	Token string     `json:"token"`
	Info  *TokenResp `json:"info"`
}

// TokenStatusReq 是令牌状态变更请求。
type TokenStatusReq struct {
	Status int8 `json:"status"`
}

// S3CredResp 是 S3 凭据响应。
type S3CredResp struct {
	ID         uint64     `json:"id"`
	Name       string     `json:"name"`
	AccessKey  string     `json:"access_key"`
	CallCount  int64      `json:"call_count"`
	LastUsedAt *time.Time `json:"last_used_at"`
	Status     int8       `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
}

// NewS3CredResp 由模型构造凭据响应。
func NewS3CredResp(c *model.S3Credential) *S3CredResp {
	if c == nil {
		return nil
	}
	return &S3CredResp{
		ID: c.ID, Name: c.Name, AccessKey: c.AccessKey,
		CallCount: c.CallCount, LastUsedAt: c.LastUsedAt,
		Status: c.Status, CreatedAt: c.CreatedAt,
	}
}

// S3CredCreateReq 是创建 S3 凭据请求。
type S3CredCreateReq struct {
	Name string `json:"name" binding:"required"`
}

// S3CredCreateResp 是创建 S3 凭据响应，密钥仅返回一次。
type S3CredCreateResp struct {
	AccessKey string      `json:"access_key"`
	SecretKey string      `json:"secret_key"`
	Info      *S3CredResp `json:"info"`
}

// ShareResp 是分享响应。
type ShareResp struct {
	ID          uint64     `json:"id"`
	Slug        string     `json:"slug"`
	TargetType  string     `json:"target_type"`
	TargetID    uint64     `json:"target_id"`
	HasPassword bool       `json:"has_password"`
	ExpiresAt   *time.Time `json:"expires_at"`
	MaxViews    int        `json:"max_views"`
	Views       int        `json:"views"`
	Status      int8       `json:"status"`
	URL         string     `json:"url"`
	CreatedAt   time.Time  `json:"created_at"`
}

// NewShareResp 由模型构造分享响应。
func NewShareResp(s *model.Share) *ShareResp {
	if s == nil {
		return nil
	}
	return &ShareResp{
		ID:          s.ID,
		Slug:        s.Slug,
		TargetType:  s.TargetType,
		TargetID:    s.TargetID,
		HasPassword: s.HasPassword(),
		ExpiresAt:   s.ExpiresAt,
		MaxViews:    s.MaxViews,
		Views:       s.Views,
		Status:      s.Status,
		CreatedAt:   s.CreatedAt,
	}
}

// ShareCreateReq 是创建分享请求。
type ShareCreateReq struct {
	TargetType     string `json:"target_type" binding:"required"`
	TargetID       uint64 `json:"target_id" binding:"required"`
	Password       string `json:"password"`
	ExpiresInHours int    `json:"expires_in_hours"`
	MaxViews       int    `json:"max_views"`
}

// ShareAccessReq 是访问分享请求。
type ShareAccessReq struct {
	Password string `form:"password" json:"password"`
}

// ShareAccessResp 是分享内容响应。
type ShareAccessResp struct {
	Share  *ShareResp   `json:"share"`
	Album  *AlbumResp   `json:"album,omitempty"`
	Images []*ImageResp `json:"images,omitempty"`
}
