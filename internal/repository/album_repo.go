package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/pichub/pichub/internal/model"
)

// AlbumRepo 相册仓储。
type AlbumRepo struct{ db *gorm.DB }

// Create 创建相册。
func (r *AlbumRepo) Create(ctx context.Context, a *model.Album) error {
	return conn(ctx, r.db).Create(a).Error
}

// GetByID 按 ID 查询相册。
func (r *AlbumRepo) GetByID(ctx context.Context, id uint64) (*model.Album, error) {
	var a model.Album
	err := conn(ctx, r.db).First(&a, id).Error
	if notFound(err) {
		return nil, nil
	}
	return &a, err
}

// ListByUser 查询用户全部相册。
func (r *AlbumRepo) ListByUser(ctx context.Context, userID uint64) ([]*model.Album, error) {
	var list []*model.Album
	err := conn(ctx, r.db).Where("user_id = ?", userID).
		Order("sort_order ASC, id DESC").Find(&list).Error
	return list, err
}

// UpdateFields 更新相册字段。
func (r *AlbumRepo) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
	return conn(ctx, r.db).Model(&model.Album{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除相册。
func (r *AlbumRepo) Delete(ctx context.Context, id uint64) error {
	return conn(ctx, r.db).Delete(&model.Album{}, id).Error
}

// CountByUser 统计用户相册数。
func (r *AlbumRepo) CountByUser(ctx context.Context, userID uint64) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.Album{}).Where("user_id = ?", userID).Count(&n).Error
	return n, err
}

// CountAll 统计全站相册数。
func (r *AlbumRepo) CountAll(ctx context.Context) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.Album{}).Count(&n).Error
	return n, err
}

// IncrImageCount 调整相册图片计数（冗余字段）。
func (r *AlbumRepo) IncrImageCount(ctx context.Context, id uint64, delta int) error {
	return conn(ctx, r.db).Model(&model.Album{}).Where("id = ?", id).
		UpdateColumn("image_count", gorm.Expr("GREATEST(image_count + ?, 0)", delta)).Error
}

// DetachImages 解绑相册内全部图片。
func (r *AlbumRepo) DetachImages(ctx context.Context, albumID uint64) error {
	return conn(ctx, r.db).Model(&model.Image{}).Where("album_id = ?", albumID).
		Update("album_id", nil).Error
}

// RecalcImageCount 重算相册图片数。
func (r *AlbumRepo) RecalcImageCount(ctx context.Context, albumID uint64) error {
	var n int64
	if err := conn(ctx, r.db).Model(&model.Image{}).
		Where("album_id = ? AND status = 1", albumID).Count(&n).Error; err != nil {
		return err
	}
	return conn(ctx, r.db).Model(&model.Album{}).Where("id = ?", albumID).
		UpdateColumn("image_count", n).Error
}

// TagRepo 标签仓储。
type TagRepo struct{ db *gorm.DB }

// Create 创建标签。
func (r *TagRepo) Create(ctx context.Context, t *model.Tag) error {
	return conn(ctx, r.db).Create(t).Error
}

// GetByID 按 ID 查询标签。
func (r *TagRepo) GetByID(ctx context.Context, id uint64) (*model.Tag, error) {
	var t model.Tag
	err := conn(ctx, r.db).First(&t, id).Error
	if notFound(err) {
		return nil, nil
	}
	return &t, err
}

// GetByUserName 按用户与名称查询标签。
func (r *TagRepo) GetByUserName(ctx context.Context, userID uint64, name string) (*model.Tag, error) {
	var t model.Tag
	err := conn(ctx, r.db).Where("user_id = ? AND name = ?", userID, name).First(&t).Error
	if notFound(err) {
		return nil, nil
	}
	return &t, err
}

// ListByUser 查询用户全部标签。
func (r *TagRepo) ListByUser(ctx context.Context, userID uint64) ([]*model.Tag, error) {
	var list []*model.Tag
	err := conn(ctx, r.db).Where("user_id = ?", userID).Order("id ASC").Find(&list).Error
	return list, err
}

// UpdateFields 更新标签字段。
func (r *TagRepo) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
	return conn(ctx, r.db).Model(&model.Tag{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除标签。
func (r *TagRepo) Delete(ctx context.Context, id uint64) error {
	return conn(ctx, r.db).Delete(&model.Tag{}, id).Error
}

// IncrImageCount 调整标签图片计数。
func (r *TagRepo) IncrImageCount(ctx context.Context, id uint64, delta int) error {
	return conn(ctx, r.db).Model(&model.Tag{}).Where("id = ?", id).
		UpdateColumn("image_count", gorm.Expr("GREATEST(image_count + ?, 0)", delta)).Error
}

// Attach 关联图片与标签（重复忽略）。
func (r *TagRepo) Attach(ctx context.Context, imageID uint64, tagIDs []uint64) error {
	if len(tagIDs) == 0 {
		return nil
	}
	rows := make([]model.ImageTag, 0, len(tagIDs))
	for _, tid := range tagIDs {
		rows = append(rows, model.ImageTag{ImageID: imageID, TagID: tid})
	}
	return conn(ctx, r.db).Clauses(clauseOnConflictDoNothing).Create(&rows).Error
}

// DetachAll 解绑图片的全部标签。
func (r *TagRepo) DetachAll(ctx context.Context, imageID uint64) error {
	return conn(ctx, r.db).Where("image_id = ?", imageID).Delete(&model.ImageTag{}).Error
}

// DetachByTag 解绑某标签的全部图片。
func (r *TagRepo) DetachByTag(ctx context.Context, tagID uint64) error {
	return conn(ctx, r.db).Where("tag_id = ?", tagID).Delete(&model.ImageTag{}).Error
}

// ListByImage 查询图片的标签 ID。
func (r *TagRepo) ListByImage(ctx context.Context, imageID uint64) ([]uint64, error) {
	var ids []uint64
	err := conn(ctx, r.db).Model(&model.ImageTag{}).
		Where("image_id = ?", imageID).Pluck("tag_id", &ids).Error
	return ids, err
}

// ListImagesByTags 查询包含任一标签的图片 ID。
func (r *TagRepo) ListImagesByTags(ctx context.Context, tagIDs []uint64) ([]uint64, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}
	var ids []uint64
	err := conn(ctx, r.db).Model(&model.ImageTag{}).
		Where("tag_id IN ?", tagIDs).Distinct().Pluck("image_id", &ids).Error
	return ids, err
}
