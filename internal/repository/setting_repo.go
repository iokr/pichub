package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pichub/pichub/internal/model"
)

// SettingRepo 系统配置仓储。
type SettingRepo struct{ db *gorm.DB }

// GetAll 返回全部配置项。
func (r *SettingRepo) GetAll(ctx context.Context) ([]*model.SystemSetting, error) {
	var list []*model.SystemSetting
	err := conn(ctx, r.db).Order("id ASC").Find(&list).Error
	return list, err
}

// Get 查询单个配置项。
func (r *SettingRepo) Get(ctx context.Context, key string) (*model.SystemSetting, error) {
	var s model.SystemSetting
	err := conn(ctx, r.db).Where("cfg_key = ?", key).First(&s).Error
	if notFound(err) {
		return nil, nil
	}
	return &s, err
}

// Set 写入或更新单个配置项。
func (r *SettingRepo) Set(ctx context.Context, key, value, valueType string, isSecret bool, desc string) error {
	s := &model.SystemSetting{
		CfgKey: key, CfgValue: value, ValueType: valueType,
		IsSecret: isSecret, Description: desc,
	}
	return conn(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cfg_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"cfg_value", "value_type", "is_secret", "description"}),
	}).Create(s).Error
}

// SetMany 批量写入配置项。
func (r *SettingRepo) SetMany(ctx context.Context, items []*model.SystemSetting) error {
	if len(items) == 0 {
		return nil
	}
	return conn(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cfg_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"cfg_value", "value_type", "is_secret", "description"}),
	}).Create(&items).Error
}

// InviteRepo 邀请码仓储。
type InviteRepo struct{ db *gorm.DB }

// Create 创建邀请码。
func (r *InviteRepo) Create(ctx context.Context, i *model.Invite) error {
	return conn(ctx, r.db).Create(i).Error
}

// GetByCode 按邀请码查询。
func (r *InviteRepo) GetByCode(ctx context.Context, code string) (*model.Invite, error) {
	var i model.Invite
	err := conn(ctx, r.db).Where("code = ?", code).First(&i).Error
	if notFound(err) {
		return nil, nil
	}
	return &i, err
}

// GetByID 按 ID 查询。
func (r *InviteRepo) GetByID(ctx context.Context, id uint64) (*model.Invite, error) {
	var i model.Invite
	err := conn(ctx, r.db).First(&i, id).Error
	if notFound(err) {
		return nil, nil
	}
	return &i, err
}

// List 分页查询邀请码。
func (r *InviteRepo) List(ctx context.Context, page, size int) ([]*model.Invite, int64, error) {
	page, size = normPage(page, size, 20, 100)
	db := conn(ctx, r.db).Model(&model.Invite{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.Invite
	err := db.Order("id DESC").Offset(pageOffset(page, size)).Limit(size).Find(&list).Error
	return list, total, err
}

// Delete 删除邀请码。
func (r *InviteRepo) Delete(ctx context.Context, id uint64) error {
	return conn(ctx, r.db).Delete(&model.Invite{}, id).Error
}

// IncrUsed 原子增加使用次数，超出上限返回 false。
func (r *InviteRepo) IncrUsed(ctx context.Context, id uint64) (bool, error) {
	res := conn(ctx, r.db).Model(&model.Invite{}).Where("id = ? AND (max_uses = 0 OR used_count < max_uses)", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1"))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// LogRepo 操作日志仓储。
type LogRepo struct{ db *gorm.DB }

// Create 写入操作日志。
func (r *LogRepo) Create(ctx context.Context, l *model.OperationLog) error {
	return conn(ctx, r.db).Create(l).Error
}

// List 分页查询操作日志。
func (r *LogRepo) List(ctx context.Context, q model.LogQuery) ([]*model.OperationLog, int64, error) {
	page, size := normPage(q.Page, q.Size, 20, 100)
	db := conn(ctx, r.db).Model(&model.OperationLog{})
	if q.UserID > 0 {
		db = db.Where("user_id = ?", q.UserID)
	}
	if q.Action != "" {
		db = db.Where("action = ?", q.Action)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.OperationLog
	err := db.Order("id DESC").Offset(pageOffset(page, size)).Limit(size).Find(&list).Error
	return list, total, err
}

// DeleteBefore 清理历史操作日志。
func (r *LogRepo) DeleteBefore(ctx context.Context, before time.Time) error {
	return conn(ctx, r.db).Where("created_at < ?", before).Delete(&model.OperationLog{}).Error
}

// CounterRepo 每日上传用量仓储。
type CounterRepo struct{ db *gorm.DB }

// Incr 累加当日上传计数。
func (r *CounterRepo) Incr(ctx context.Context, userID uint64, date model.Date, count int, bytes int64) error {
	c := &model.UploadCounter{UserID: userID, StatDate: date, Count: count, Bytes: bytes}
	return conn(ctx, r.db).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "stat_date"}},
		DoUpdates: clause.Assignments(map[string]any{
			"count":      gorm.Expr("count + ?", count),
			"bytes":      gorm.Expr("bytes + ?", bytes),
			"updated_at": model.Now(),
		}),
	}).Create(c).Error
}

// Get 查询某日用量。
func (r *CounterRepo) Get(ctx context.Context, userID uint64, date model.Date) (*model.UploadCounter, error) {
	var c model.UploadCounter
	err := conn(ctx, r.db).Where("user_id = ? AND stat_date = ?", userID, string(date)).First(&c).Error
	if notFound(err) {
		return nil, nil
	}
	return &c, err
}

// SumSince 统计指定日期之后的累计上传数。
func (r *CounterRepo) SumSince(ctx context.Context, userID uint64, from model.Date) (int64, int64, error) {
	type row struct {
		Count int64
		Bytes int64
	}
	var res row
	err := conn(ctx, r.db).Model(&model.UploadCounter{}).
		Select("COALESCE(SUM(count),0) AS count, COALESCE(SUM(bytes),0) AS bytes").
		Where("user_id = ? AND stat_date >= ?", userID, string(from)).Scan(&res).Error
	return res.Count, res.Bytes, err
}

// Add 累加任意用户当日上传计数（S3 网关与普通上传共用）。
func (r *CounterRepo) Add(ctx context.Context, userID uint64, date model.Date, bytes int64) error {
	return r.Incr(ctx, userID, date, 1, bytes)
}
