package dto

import (
	"time"

	"github.com/pichub/pichub/internal/model"
)

// 批量操作类型。
const (
	BatchDelete     = "delete"
	BatchRestore    = "restore"
	BatchMove       = "move"
	BatchSetPublic  = "public"
	BatchSetPrivate = "private"
)

// ImageListReq 是图片列表查询参数。
type ImageListReq struct {
	UserID   uint64  `json:"-" form:"-"`
	Keyword  string  `form:"keyword"`
	AlbumID  *uint64 `form:"album_id"`
	TagID    uint64  `form:"tag_id"`
	Ext      string  `form:"ext"`
	Referred bool    `form:"referred"`
	IsPublic *bool   `form:"is_public"`
	Cursor   uint64  `form:"cursor"`
	Page     int     `form:"page"`
	Size     int     `form:"size"`
	Trash    bool    `form:"trash"`
}

// ImageResp 是图片响应。
type ImageResp struct {
	ID         uint64     `json:"id"`
	Hash       string     `json:"hash"`
	Name       string     `json:"name"`
	OriginName string     `json:"origin_name"`
	Ext        string     `json:"ext"`
	Mime       string     `json:"mime"`
	Size       int64      `json:"size"`
	Width      int        `json:"width"`
	Height     int        `json:"height"`
	URL        string     `json:"url"`
	ThumbURL   string     `json:"thumb_url"`
	IsPublic   bool       `json:"is_public"`
	AlbumID    *uint64    `json:"album_id"`
	Tags       []*TagResp `json:"tags,omitempty"`
	RefCount   int        `json:"ref_count"`
	Status     int8       `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	DeletedAt  *time.Time `json:"deleted_at"`
}

// NewImageResp 由模型构造图片响应。
func NewImageResp(img *model.Image) *ImageResp {
	if img == nil {
		return nil
	}
	return &ImageResp{
		ID:         img.ID,
		Hash:       img.Hash,
		Name:       img.DisplayName,
		OriginName: img.OriginName,
		Ext:        img.Ext,
		Mime:       img.Mime,
		Size:       img.Size,
		Width:      img.Width,
		Height:     img.Height,
		URL:        img.URL,
		IsPublic:   img.IsPublic,
		AlbumID:    img.AlbumID,
		RefCount:   img.RefCount,
		Status:     img.Status,
		CreatedAt:  img.CreatedAt,
		DeletedAt:  img.DeletedAt,
	}
}

// NewImageRespList 批量构造图片响应。
func NewImageRespList(list []*model.Image) []*ImageResp {
	out := make([]*ImageResp, 0, len(list))
	for _, img := range list {
		out = append(out, NewImageResp(img))
	}
	return out
}

// ImageUpdateReq 是图片修改请求。
type ImageUpdateReq struct {
	DisplayName *string   `json:"display_name"`
	AlbumID     *uint64   `json:"album_id"`
	ClearAlbum  bool      `json:"clear_album"`
	Tags        *[]string `json:"tags"`
	IsPublic    *bool     `json:"is_public"`
}

// ImageBatchReq 是批量操作请求。
type ImageBatchReq struct {
	IDs     []uint64 `json:"ids" binding:"required"`
	Action  string   `json:"action" binding:"required"`
	AlbumID *uint64  `json:"album_id"`
}

// ImageBatchResp 是批量操作结果。
type ImageBatchResp struct {
	Affected int `json:"affected"`
}

// RefererResp 是引用来源响应。
type RefererResp struct {
	ID          uint64    `json:"id"`
	Host        string    `json:"host"`
	URL         string    `json:"url"`
	Hits        int64     `json:"hits"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

// NewRefererResp 由模型构造引用响应。
func NewRefererResp(r *model.ImageReferer) *RefererResp {
	return &RefererResp{
		ID:          r.ID,
		Host:        r.RefererHost,
		URL:         r.RefererURL,
		Hits:        r.HitCount,
		FirstSeenAt: r.FirstSeenAt,
		LastSeenAt:  r.LastSeenAt,
	}
}

// RefererSummaryResp 是单图引用汇总。
type RefererSummaryResp struct {
	TotalHits int64 `json:"total_hits"`
	HostCount int64 `json:"host_count"`
}
