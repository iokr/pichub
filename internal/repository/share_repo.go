package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/pichub/pichub/internal/model"
)

// ShareRepo 分享仓储。
type ShareRepo struct{ db *gorm.DB }

// Create 创建分享。
func (r *ShareRepo) Create(ctx context.Context, s *model.Share) error {
	return conn(ctx, r.db).Create(s).Error
}

// GetByID 按 ID 查询分享。
func (r *ShareRepo) GetByID(ctx context.Context, id uint64) (*model.Share, error) {
	var s model.Share
	err := conn(ctx, r.db).First(&s, id).Error
	if notFound(err) {
		return nil, nil
	}
	return &s, err
}

// GetBySlug 按短链查询分享。
func (r *ShareRepo) GetBySlug(ctx context.Context, slug string) (*model.Share, error) {
	var s model.Share
	err := conn(ctx, r.db).Where("slug = ?", slug).First(&s).Error
	if notFound(err) {
		return nil, nil
	}
	return &s, err
}

// ListByUser 查询用户的分享。
func (r *ShareRepo) ListByUser(ctx context.Context, userID uint64) ([]*model.Share, error) {
	var list []*model.Share
	err := conn(ctx, r.db).Where("user_id = ?", userID).Order("id DESC").Find(&list).Error
	return list, err
}

// Delete 删除分享。
func (r *ShareRepo) Delete(ctx context.Context, id uint64) error {
	return conn(ctx, r.db).Delete(&model.Share{}, id).Error
}

// IncrViews 增加访问次数。
func (r *ShareRepo) IncrViews(ctx context.Context, id uint64) error {
	return conn(ctx, r.db).Model(&model.Share{}).Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + 1")).Error
}

// CountByUser 统计用户分享数。
func (r *ShareRepo) CountByUser(ctx context.Context, userID uint64) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.Share{}).Where("user_id = ?", userID).Count(&n).Error
	return n, err
}

// CountAll 统计全站分享数。
func (r *ShareRepo) CountAll(ctx context.Context) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.Share{}).Count(&n).Error
	return n, err
}

// DeleteByTarget 删除目标对象上的全部分享。
func (r *ShareRepo) DeleteByTarget(ctx context.Context, targetType string, targetID uint64) error {
	return conn(ctx, r.db).Where("target_type = ? AND target_id = ?", targetType, targetID).
		Delete(&model.Share{}).Error
}

// RefererRepo 引用统计仓储。
type RefererRepo struct{ db *gorm.DB }

// UpsertBatch 批量累加引用计数。
func (r *RefererRepo) UpsertBatch(ctx context.Context, items []*model.ImageReferer) error {
	if len(items) == 0 {
		return nil
	}
	db := conn(ctx, r.db)
	for _, it := range items {
		err := db.Exec(`INSERT INTO image_referers
			(image_id, referer_hash, referer_url, referer_host, hit_count, first_seen_at, last_seen_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				hit_count = hit_count + VALUES(hit_count),
				referer_url = VALUES(referer_url),
				last_seen_at = VALUES(last_seen_at)`,
			it.ImageID, it.RefererHash, it.RefererURL, it.RefererHost, it.HitCount, it.FirstSeenAt, it.LastSeenAt).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// UpsertDailyBatch 批量累加按天引用计数。
func (r *RefererRepo) UpsertDailyBatch(ctx context.Context, date model.Date, items []*model.ImageReferer) error {
	if len(items) == 0 {
		return nil
	}
	db := conn(ctx, r.db)
	for _, it := range items {
		err := db.Exec(`INSERT INTO image_referer_daily (image_id, stat_date, referer_host, hit_count)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE hit_count = hit_count + VALUES(hit_count)`,
			it.ImageID, string(date), it.RefererHost, it.HitCount).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// ListByImage 分页查询图片引用来源。
func (r *RefererRepo) ListByImage(ctx context.Context, q model.RefererQuery) ([]*model.ImageReferer, int64, error) {
	page, size := normPage(q.Page, q.Size, 20, 200)
	db := conn(ctx, r.db).Model(&model.ImageReferer{}).Where("image_id = ?", q.ImageID)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "hit_count DESC, id DESC"
	if q.OrderBy == "time" {
		order = "last_seen_at DESC"
	}
	var list []*model.ImageReferer
	err := db.Order(order).Offset(pageOffset(page, size)).Limit(size).Find(&list).Error
	return list, total, err
}

// Summary 统计单图引用汇总。
func (r *RefererRepo) Summary(ctx context.Context, imageID uint64) (totalHits, hostCount int64, err error) {
	type row struct {
		Total int64
		Hosts int64
	}
	var res row
	err = conn(ctx, r.db).Model(&model.ImageReferer{}).
		Select("COALESCE(SUM(hit_count),0) AS total, COUNT(*) AS hosts").
		Where("image_id = ?", imageID).Scan(&res).Error
	return res.Total, res.Hosts, err
}

// Delete 删除单条引用记录。
func (r *RefererRepo) Delete(ctx context.Context, id, imageID uint64) error {
	return conn(ctx, r.db).Where("id = ? AND image_id = ?", id, imageID).
		Delete(&model.ImageReferer{}).Error
}

// DeleteByImage 删除图片的全部引用。
func (r *RefererRepo) DeleteByImage(ctx context.Context, imageID uint64) error {
	return conn(ctx, r.db).Where("image_id = ?", imageID).Delete(&model.ImageReferer{}).Error
}

// TopHosts 统计用户图片的 TOP 引用域名。
func (r *RefererRepo) TopHosts(ctx context.Context, userID uint64, since time.Time, limit int) ([]*model.HostStat, error) {
	var list []*model.HostStat
	err := conn(ctx, r.db).Table("image_referers ir").
		Select("ir.referer_host AS referer_host, SUM(ir.hit_count) AS hits, COUNT(DISTINCT ir.image_id) AS images").
		Joins("JOIN images i ON i.id = ir.image_id AND i.user_id = ? AND i.status = 1", userID).
		Where("ir.last_seen_at >= ?", since).
		Group("ir.referer_host").
		Order("hits DESC").
		Limit(limit).Scan(&list).Error
	return list, err
}

// Trend 按天统计引用趋势。
func (r *RefererRepo) Trend(ctx context.Context, userID uint64, from, to time.Time) ([]*model.DayStat, error) {
	var list []*model.DayStat
	err := conn(ctx, r.db).Table("image_referer_daily d").
		Select("d.stat_date AS stat_date, SUM(d.hit_count) AS hits").
		Joins("JOIN images i ON i.id = d.image_id AND i.user_id = ?", userID).
		Where("d.stat_date BETWEEN ? AND ?", from.Format("2006-01-02"), to.Format("2006-01-02")).
		Group("d.stat_date").Order("d.stat_date ASC").Scan(&list).Error
	return list, err
}

// AggregateDaily 将引用记录按天聚合。
func (r *RefererRepo) AggregateDaily(ctx context.Context, date model.Date) error {
	return conn(ctx, r.db).Exec(`INSERT INTO image_referer_daily (image_id, stat_date, referer_host, hit_count)
		SELECT image_id, ?, referer_host, hit_count FROM image_referers WHERE hit_count > 0
		ON DUPLICATE KEY UPDATE hit_count = image_referer_daily.hit_count + VALUES(hit_count)`,
		string(date)).Error
}

// DeleteDailyBefore 清理过期的按天聚合数据。
func (r *RefererRepo) DeleteDailyBefore(ctx context.Context, date model.Date) error {
	return conn(ctx, r.db).Where("stat_date < ?", string(date)).Delete(&model.ImageRefererDaily{}).Error
}

// TotalHits 统计全站引用总次数。
func (r *RefererRepo) TotalHits(ctx context.Context) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.ImageReferer{}).
		Select("COALESCE(SUM(hit_count),0)").Scan(&n).Error
	return n, err
}

// SumHitsByImage 统计单图引用总次数。
func (r *RefererRepo) SumHitsByImage(ctx context.Context, imageID uint64) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.ImageReferer{}).
		Select("COALESCE(SUM(hit_count),0)").Where("image_id = ?", imageID).Scan(&n).Error
	return n, err
}

// SumHitsByUser 统计用户全部图片的引用总次数。
func (r *RefererRepo) SumHitsByUser(ctx context.Context, userID uint64) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Table("image_referers ir").
		Select("COALESCE(SUM(ir.hit_count),0)").
		Joins("JOIN images i ON i.id = ir.image_id AND i.user_id = ?", userID).
		Scan(&n).Error
	return n, err
}
