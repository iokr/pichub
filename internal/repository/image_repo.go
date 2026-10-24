package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/pichub/pichub/internal/model"
)

// ImageRepo 图片仓储。
type ImageRepo struct{ db *gorm.DB }

// Create 创建图片记录。
func (r *ImageRepo) Create(ctx context.Context, img *model.Image) error {
	return conn(ctx, r.db).Create(img).Error
}

// GetByID 按 ID 查询（含回收站）。
func (r *ImageRepo) GetByID(ctx context.Context, id uint64) (*model.Image, error) {
	var img model.Image
	err := conn(ctx, r.db).First(&img, id).Error
	if notFound(err) {
		return nil, nil
	}
	return &img, err
}

// GetByUserHash 查询用户未删除的同哈希图片（秒传用）。
func (r *ImageRepo) GetByUserHash(ctx context.Context, userID uint64, hash string) (*model.Image, error) {
	var img model.Image
	err := conn(ctx, r.db).
		Where("user_id = ? AND hash = ? AND deleted_flag = 0 AND status = 1", userID, hash).
		First(&img).Error
	if notFound(err) {
		return nil, nil
	}
	return &img, err
}

// GetByHash 查询可公开访问的同哈希图片。
func (r *ImageRepo) GetByHash(ctx context.Context, hash string) (*model.Image, error) {
	var img model.Image
	err := conn(ctx, r.db).
		Where("hash = ? AND status = 1 AND is_public = 1", hash).
		Order("id ASC").First(&img).Error
	if notFound(err) {
		return nil, nil
	}
	return &img, err
}

// GetByHashAny 查询任意存活图片（含私有），用于签名访问与 S3 网关。
func (r *ImageRepo) GetByHashAny(ctx context.Context, hash string) (*model.Image, error) {
	var img model.Image
	err := conn(ctx, r.db).
		Where("hash = ? AND status = 1", hash).
		Order("id DESC").First(&img).Error
	if notFound(err) {
		return nil, nil
	}
	return &img, err
}

// GetByStorageKey 按存储 Key 查询存活图片。
func (r *ImageRepo) GetByStorageKey(ctx context.Context, key string) (*model.Image, error) {
	var img model.Image
	err := conn(ctx, r.db).
		Where("storage_key = ? AND status = 1", key).
		Order("id DESC").First(&img).Error
	if notFound(err) {
		return nil, nil
	}
	return &img, err
}

// GetByUserStorageKey 按用户与存储 Key 查询存活图片（S3 网关用）。
func (r *ImageRepo) GetByUserStorageKey(ctx context.Context, userID uint64, key string) (*model.Image, error) {
	var img model.Image
	err := conn(ctx, r.db).
		Where("user_id = ? AND storage_key = ? AND status = ?", userID, key, model.ImageStatusNormal).
		Order("id DESC").First(&img).Error
	if notFound(err) {
		return nil, nil
	}
	return &img, err
}

// GetPublicByStorageKey 按存储 Key 查询公开图片（S3 网关匿名下载用）。
func (r *ImageRepo) GetPublicByStorageKey(ctx context.Context, key string) (*model.Image, error) {
	var img model.Image
	err := conn(ctx, r.db).
		Where("storage_key = ? AND is_public = 1 AND status = ?", key, model.ImageStatusNormal).
		Order("id DESC").First(&img).Error
	if notFound(err) {
		return nil, nil
	}
	return &img, err
}

// ListStorageObjects 按存储 Key 前缀与游标查询存活图片（S3 网关列举用）。
func (r *ImageRepo) ListStorageObjects(ctx context.Context, userID uint64, prefix, after string, limit int) ([]*model.Image, error) {
	if limit <= 0 {
		limit = 1000
	}
	if limit > 2000 {
		limit = 2000
	}
	db := conn(ctx, r.db).Model(&model.Image{}).
		Where("user_id = ? AND status = ?", userID, model.ImageStatusNormal)
	if prefix != "" {
		db = db.Where("storage_key LIKE ?", escapeLike(prefix)+"%")
	}
	if after != "" {
		db = db.Where("storage_key > ?", after)
	}
	var list []*model.Image
	err := db.Order("storage_key ASC").Limit(limit).Find(&list).Error
	return list, err
}

// escapeLike 转义 LIKE 通配符，避免用户输入被当作模式。
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// List 分页/游标查询图片。
func (r *ImageRepo) List(ctx context.Context, q model.ImageQuery) ([]*model.Image, int64, error) {
	page, size := normPage(q.Page, q.Size, 24, 200)
	db := conn(ctx, r.db).Model(&model.Image{}).Where("user_id = ?", q.UserID)

	if q.TrashOnly {
		db = db.Where("status = ?", model.ImageStatusTrash)
	} else if q.Status != nil {
		db = db.Where("status = ?", *q.Status)
	} else {
		db = db.Where("status = ?", model.ImageStatusNormal)
	}
	if q.AlbumID != nil {
		db = db.Where("album_id = ?", *q.AlbumID)
	}
	if q.TagID > 0 {
		sub := conn(ctx, r.db).Model(&model.ImageTag{}).Select("image_id").Where("tag_id = ?", q.TagID)
		db = db.Where("id IN (?)", sub)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		db = db.Where("(display_name LIKE ? OR origin_name LIKE ? OR hash LIKE ?)", kw, kw, kw)
	}
	if q.Ext != "" {
		db = db.Where("ext = ?", q.Ext)
	}
	if q.Referred {
		db = db.Where("EXISTS (SELECT 1 FROM image_referers ir WHERE ir.image_id = images.id)")
	}
	if q.IsPublic != nil {
		db = db.Where("is_public = ?", *q.IsPublic)
	}

	var total int64
	if q.Cursor == 0 {
		if err := db.Count(&total).Error; err != nil {
			return nil, 0, err
		}
	}

	order := "created_at DESC, id DESC"
	if q.TrashOnly {
		order = "deleted_at DESC, id DESC"
	}
	if q.Cursor > 0 {
		db = db.Where("id < ?", q.Cursor)
	}

	var list []*model.Image
	err := db.Order(order).
		Offset(func() int {
			if q.Cursor > 0 {
				return 0
			}
			return pageOffset(page, size)
		}()).
		Limit(size).Find(&list).Error
	return list, total, err
}

// SoftDelete 移入回收站。
func (r *ImageRepo) SoftDelete(ctx context.Context, id uint64, deletedFlag int64, at time.Time) error {
	return conn(ctx, r.db).Model(&model.Image{}).Where("id = ?", id).Updates(map[string]any{
		"status":       model.ImageStatusTrash,
		"deleted_flag": deletedFlag,
		"deleted_at":   at,
	}).Error
}

// Restore 从回收站还原。
func (r *ImageRepo) Restore(ctx context.Context, id uint64) error {
	return conn(ctx, r.db).Model(&model.Image{}).Where("id = ?", id).Updates(map[string]any{
		"status":       model.ImageStatusNormal,
		"deleted_flag": 0,
		"deleted_at":   nil,
	}).Error
}

// HardDelete 物理删除记录。
func (r *ImageRepo) HardDelete(ctx context.Context, id uint64) error {
	return conn(ctx, r.db).Delete(&model.Image{}, id).Error
}

// UpdateFields 按字段更新。
func (r *ImageRepo) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
	return conn(ctx, r.db).Model(&model.Image{}).Where("id = ?", id).Updates(fields).Error
}

// IncrRefCount 原子增减引用计数。
func (r *ImageRepo) IncrRefCount(ctx context.Context, id uint64, delta int) error {
	return conn(ctx, r.db).Model(&model.Image{}).Where("id = ?", id).
		UpdateColumn("ref_count", gorm.Expr("GREATEST(ref_count + ?, 0)", delta)).Error
}

// ListTrashExpired 查询超过保留期的回收站图片。
func (r *ImageRepo) ListTrashExpired(ctx context.Context, before time.Time, limit int) ([]*model.Image, error) {
	var list []*model.Image
	err := conn(ctx, r.db).
		Where("status = ? AND deleted_at IS NOT NULL AND deleted_at < ?", model.ImageStatusTrash, before).
		Limit(limit).Find(&list).Error
	return list, err
}

// CountByUser 统计用户图片数。
func (r *ImageRepo) CountByUser(ctx context.Context, userID uint64, status int8) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.Image{}).
		Where("user_id = ? AND status = ?", userID, status).Count(&n).Error
	return n, err
}

// SumSizeByUser 统计用户图片占用空间。
func (r *ImageRepo) SumSizeByUser(ctx context.Context, userID uint64) (int64, error) {
	var sum int64
	err := conn(ctx, r.db).Model(&model.Image{}).
		Select("COALESCE(SUM(size),0)").Where("user_id = ?", userID).Scan(&sum).Error
	return sum, err
}

// CountAll 统计全站图片数（不含回收站）。
func (r *ImageRepo) CountAll(ctx context.Context) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.Image{}).Where("status = 1").Count(&n).Error
	return n, err
}

// CountByStatus 按状态统计图片数。
func (r *ImageRepo) CountByStatus(ctx context.Context, status int8) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.Image{}).Where("status = ?", status).Count(&n).Error
	return n, err
}

// SumSizeAll 统计全站占用空间。
func (r *ImageRepo) SumSizeAll(ctx context.Context) (int64, error) {
	var sum int64
	err := conn(ctx, r.db).Model(&model.Image{}).Select("COALESCE(SUM(size),0)").Scan(&sum).Error
	return sum, err
}

// CountCreatedSince 统计时间之后的图片数。
func (r *ImageRepo) CountCreatedSince(ctx context.Context, since time.Time) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.Image{}).
		Where("created_at >= ? AND status = 1", since).Count(&n).Error
	return n, err
}
