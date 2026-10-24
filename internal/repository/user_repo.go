package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/pichub/pichub/internal/model"
)

// UserRepo 用户仓储。
type UserRepo struct{ db *gorm.DB }

// Create 创建用户。
func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	return conn(ctx, r.db).Create(u).Error
}

// GetByID 按 ID 查询。
func (r *UserRepo) GetByID(ctx context.Context, id uint64) (*model.User, error) {
	var u model.User
	err := conn(ctx, r.db).First(&u, id).Error
	if notFound(err) {
		return nil, nil
	}
	return &u, err
}

// GetByUsername 按用户名查询（大小写不敏感）。
func (r *UserRepo) GetByUsername(ctx context.Context, name string) (*model.User, error) {
	var u model.User
	err := conn(ctx, r.db).Where("username = ?", name).First(&u).Error
	if notFound(err) {
		return nil, nil
	}
	return &u, err
}

// GetByEmail 按邮箱查询（大小写不敏感）。
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := conn(ctx, r.db).Where("email = ?", email).First(&u).Error
	if notFound(err) {
		return nil, nil
	}
	return &u, err
}

// Count 统计用户总数。
func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.User{}).Count(&n).Error
	return n, err
}

// CountByIP 统计指定 IP 在 since 之后的注册数。
func (r *UserRepo) CountByIP(ctx context.Context, ip string, since time.Time) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.User{}).
		Where("register_ip = ? AND created_at >= ?", ip, since).Count(&n).Error
	return n, err
}

// List 分页查询用户。
func (r *UserRepo) List(ctx context.Context, q model.UserQuery) ([]*model.User, int64, error) {
	page, size := normPage(q.Page, q.Size, 20, 100)
	db := conn(ctx, r.db).Model(&model.User{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		db = db.Where("username LIKE ? OR email LIKE ? OR nickname LIKE ?", kw, kw, kw)
	}
	if q.Status != nil {
		db = db.Where("status = ?", *q.Status)
	}
	if q.Role != nil {
		db = db.Where("role = ?", *q.Role)
	}
	if q.GroupID > 0 {
		db = db.Where("group_id = ?", q.GroupID)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.User
	err := db.Order("id DESC").Offset(pageOffset(page, size)).Limit(size).Find(&list).Error
	return list, total, err
}

// UpdateFields 按字段更新。
func (r *UserRepo) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
	return conn(ctx, r.db).Model(&model.User{}).Where("id = ?", id).Updates(fields).Error
}

// AddUsedBytes 原子增加已用容量，超出配额返回 false。
func (r *UserRepo) AddUsedBytes(ctx context.Context, id uint64, delta, quota int64) (bool, error) {
	db := conn(ctx, r.db).Model(&model.User{}).Where("id = ?", id)
	if quota > 0 {
		db = db.Where("used_bytes + ? <= ?", delta, quota)
	}
	res := db.UpdateColumn("used_bytes", gorm.Expr("used_bytes + ?", delta))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// SubUsedBytes 原子减少已用容量，不会小于 0。
func (r *UserRepo) SubUsedBytes(ctx context.Context, id uint64, delta int64) error {
	return conn(ctx, r.db).Model(&model.User{}).Where("id = ?", id).
		UpdateColumn("used_bytes", gorm.Expr("GREATEST(used_bytes - ?, 0)", delta)).Error
}

// CountAdmins 统计管理员数量。
func (r *UserRepo) CountAdmins(ctx context.Context) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.User{}).Where("role = ?", model.RoleAdmin).Count(&n).Error
	return n, err
}

// UserGroupRepo 用户组仓储。
type UserGroupRepo struct{ db *gorm.DB }

// Create 创建用户组。
func (r *UserGroupRepo) Create(ctx context.Context, g *model.UserGroup) error {
	return conn(ctx, r.db).Create(g).Error
}

// GetByID 按 ID 查询用户组。
func (r *UserGroupRepo) GetByID(ctx context.Context, id uint64) (*model.UserGroup, error) {
	var g model.UserGroup
	err := conn(ctx, r.db).First(&g, id).Error
	if notFound(err) {
		return nil, nil
	}
	return &g, err
}

// GetDefault 返回默认用户组。
func (r *UserGroupRepo) GetDefault(ctx context.Context) (*model.UserGroup, error) {
	var g model.UserGroup
	err := conn(ctx, r.db).Where("is_default = 1 AND status = 1").First(&g).Error
	if notFound(err) {
		return nil, nil
	}
	return &g, err
}

// List 查询全部用户组。
func (r *UserGroupRepo) List(ctx context.Context) ([]*model.UserGroup, error) {
	var list []*model.UserGroup
	err := conn(ctx, r.db).Order("id ASC").Find(&list).Error
	return list, err
}

// UpdateFields 更新用户组字段。
func (r *UserGroupRepo) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
	return conn(ctx, r.db).Model(&model.UserGroup{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除用户组。
func (r *UserGroupRepo) Delete(ctx context.Context, id uint64) error {
	return conn(ctx, r.db).Delete(&model.UserGroup{}, id).Error
}

// CountUsers 统计组内用户数。
func (r *UserGroupRepo) CountUsers(ctx context.Context, id uint64) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.User{}).Where("group_id = ?", id).Count(&n).Error
	return n, err
}
