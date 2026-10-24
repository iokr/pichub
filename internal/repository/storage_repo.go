package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pichub/pichub/internal/model"
)

// clauseOnConflictDoNothing 用于忽略主键冲突的批量插入。
var clauseOnConflictDoNothing = clause.OnConflict{DoNothing: true}

// StorageRepo 存储策略仓储。
type StorageRepo struct{ db *gorm.DB }

// Create 创建存储策略。
func (r *StorageRepo) Create(ctx context.Context, s *model.StorageConfig) error {
	return conn(ctx, r.db).Create(s).Error
}

// GetByID 按 ID 查询策略。
func (r *StorageRepo) GetByID(ctx context.Context, id uint64) (*model.StorageConfig, error) {
	var s model.StorageConfig
	err := conn(ctx, r.db).First(&s, id).Error
	if notFound(err) {
		return nil, nil
	}
	return &s, err
}

// GetDefault 返回启用中的系统默认策略。
func (r *StorageRepo) GetDefault(ctx context.Context) (*model.StorageConfig, error) {
	var s model.StorageConfig
	err := conn(ctx, r.db).Where("is_default = 1 AND status = 1 AND user_id IS NULL").First(&s).Error
	if notFound(err) {
		return nil, nil
	}
	return &s, err
}

// ListSystem 查询系统级策略。
func (r *StorageRepo) ListSystem(ctx context.Context) ([]*model.StorageConfig, error) {
	var list []*model.StorageConfig
	err := conn(ctx, r.db).Where("user_id IS NULL").Order("id ASC").Find(&list).Error
	return list, err
}

// ListByUser 查询用户自建策略。
func (r *StorageRepo) ListByUser(ctx context.Context, userID uint64) ([]*model.StorageConfig, error) {
	var list []*model.StorageConfig
	err := conn(ctx, r.db).Where("user_id = ?", userID).Order("id ASC").Find(&list).Error
	return list, err
}

// UpdateFields 更新策略字段。
func (r *StorageRepo) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
	return conn(ctx, r.db).Model(&model.StorageConfig{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除策略。
func (r *StorageRepo) Delete(ctx context.Context, id uint64) error {
	return conn(ctx, r.db).Delete(&model.StorageConfig{}, id).Error
}

// ClearDefault 取消全部系统默认标记。
func (r *StorageRepo) ClearDefault(ctx context.Context) error {
	return conn(ctx, r.db).Model(&model.StorageConfig{}).
		Where("is_default = 1").Update("is_default", false).Error
}

// CountByUser 统计用户策略数。
func (r *StorageRepo) CountByUser(ctx context.Context, userID uint64) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.StorageConfig{}).Where("user_id = ?", userID).Count(&n).Error
	return n, err
}

// EmailRepo 邮件仓储。
type EmailRepo struct{ db *gorm.DB }

// CreateToken 创建邮箱令牌。
func (r *EmailRepo) CreateToken(ctx context.Context, t *model.EmailToken) error {
	return conn(ctx, r.db).Create(t).Error
}

// GetLatestToken 查询用户某场景最新的令牌。
func (r *EmailRepo) GetLatestToken(ctx context.Context, userID uint64, scene string) (*model.EmailToken, error) {
	var t model.EmailToken
	err := conn(ctx, r.db).Where("user_id = ? AND scene = ?", userID, scene).
		Order("id DESC").First(&t).Error
	if notFound(err) {
		return nil, nil
	}
	return &t, err
}

// GetTokenByHash 按哈希查询令牌。
func (r *EmailRepo) GetTokenByHash(ctx context.Context, hash string) (*model.EmailToken, error) {
	var t model.EmailToken
	err := conn(ctx, r.db).Where("token_hash = ?", hash).First(&t).Error
	if notFound(err) {
		return nil, nil
	}
	return &t, err
}

// MarkTokenUsed 标记令牌已使用。
func (r *EmailRepo) MarkTokenUsed(ctx context.Context, id uint64, at time.Time) error {
	return conn(ctx, r.db).Model(&model.EmailToken{}).Where("id = ?", id).
		Update("used_at", at).Error
}

// DeleteExpiredTokens 清理过期令牌。
func (r *EmailRepo) DeleteExpiredTokens(ctx context.Context, before time.Time) error {
	return conn(ctx, r.db).Where("expires_at < ?", before).Delete(&model.EmailToken{}).Error
}

// CreateLog 创建邮件日志。
func (r *EmailRepo) CreateLog(ctx context.Context, l *model.EmailLog) error {
	return conn(ctx, r.db).Create(l).Error
}

// UpdateLog 更新邮件日志。
func (r *EmailRepo) UpdateLog(ctx context.Context, id uint64, fields map[string]any) error {
	return conn(ctx, r.db).Model(&model.EmailLog{}).Where("id = ?", id).Updates(fields).Error
}

// ListLogs 分页查询邮件日志。
func (r *EmailRepo) ListLogs(ctx context.Context, q model.EmailLogQuery) ([]*model.EmailLog, int64, error) {
	page, size := normPage(q.Page, q.Size, 20, 100)
	db := conn(ctx, r.db).Model(&model.EmailLog{})
	if q.UserID > 0 {
		db = db.Where("user_id = ?", q.UserID)
	}
	if q.Status != nil {
		db = db.Where("status = ?", *q.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.EmailLog
	err := db.Order("id DESC").Offset(pageOffset(page, size)).Limit(size).Find(&list).Error
	return list, total, err
}

// DeleteLogsBefore 清理历史邮件日志。
func (r *EmailRepo) DeleteLogsBefore(ctx context.Context, before time.Time) error {
	return conn(ctx, r.db).Where("created_at < ?", before).Delete(&model.EmailLog{}).Error
}

// CountTodaySuccess 统计当日成功发送数（用于每日限额）。
func (r *EmailRepo) CountTodaySuccess(ctx context.Context, since time.Time) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.EmailLog{}).
		Where("status = 1 AND sent_at >= ?", since).Count(&n).Error
	return n, err
}
