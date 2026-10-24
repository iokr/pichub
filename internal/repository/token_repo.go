package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/pichub/pichub/internal/model"
)

// APITokenRepo API Token 仓储。
type APITokenRepo struct{ db *gorm.DB }

// Create 创建令牌。
func (r *APITokenRepo) Create(ctx context.Context, t *model.APIToken) error {
	return conn(ctx, r.db).Create(t).Error
}

// GetByID 按 ID 查询令牌。
func (r *APITokenRepo) GetByID(ctx context.Context, id uint64) (*model.APIToken, error) {
	var t model.APIToken
	err := conn(ctx, r.db).First(&t, id).Error
	if notFound(err) {
		return nil, nil
	}
	return &t, err
}

// GetByHash 按哈希查询启用中的令牌。
func (r *APITokenRepo) GetByHash(ctx context.Context, hash string) (*model.APIToken, error) {
	var t model.APIToken
	err := conn(ctx, r.db).Where("token_hash = ? AND status = 1", hash).First(&t).Error
	if notFound(err) {
		return nil, nil
	}
	return &t, err
}

// ListByUser 查询用户的令牌。
func (r *APITokenRepo) ListByUser(ctx context.Context, userID uint64) ([]*model.APIToken, error) {
	var list []*model.APIToken
	err := conn(ctx, r.db).Where("user_id = ?", userID).Order("id DESC").Find(&list).Error
	return list, err
}

// Delete 删除令牌。
func (r *APITokenRepo) Delete(ctx context.Context, id uint64) error {
	return conn(ctx, r.db).Delete(&model.APIToken{}, id).Error
}

// UpdateFields 更新令牌字段。
func (r *APITokenRepo) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
	return conn(ctx, r.db).Model(&model.APIToken{}).Where("id = ?", id).Updates(fields).Error
}

// CountByUser 统计用户令牌数。
func (r *APITokenRepo) CountByUser(ctx context.Context, userID uint64) (int64, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.APIToken{}).Where("user_id = ?", userID).Count(&n).Error
	return n, err
}

// IncrCallCount 增加调用计数。
func (r *APITokenRepo) IncrCallCount(ctx context.Context, id uint64, ip string) error {
	return conn(ctx, r.db).Model(&model.APIToken{}).Where("id = ?", id).Updates(map[string]any{
		"call_count":   gorm.Expr("call_count + 1"),
		"last_used_at": model.Now(),
		"last_used_ip": ip,
	}).Error
}

// S3CredentialRepo S3 凭据仓储。
type S3CredentialRepo struct{ db *gorm.DB }

// Create 创建凭据。
func (r *S3CredentialRepo) Create(ctx context.Context, c *model.S3Credential) error {
	return conn(ctx, r.db).Create(c).Error
}

// GetByID 按 ID 查询凭据。
func (r *S3CredentialRepo) GetByID(ctx context.Context, id uint64) (*model.S3Credential, error) {
	var c model.S3Credential
	err := conn(ctx, r.db).First(&c, id).Error
	if notFound(err) {
		return nil, nil
	}
	return &c, err
}

// GetByAccessKey 按 AccessKey 查询凭据。
func (r *S3CredentialRepo) GetByAccessKey(ctx context.Context, ak string) (*model.S3Credential, error) {
	var c model.S3Credential
	err := conn(ctx, r.db).Where("access_key = ?", ak).First(&c).Error
	if notFound(err) {
		return nil, nil
	}
	return &c, err
}

// ListByUser 查询用户的凭据。
func (r *S3CredentialRepo) ListByUser(ctx context.Context, userID uint64) ([]*model.S3Credential, error) {
	var list []*model.S3Credential
	err := conn(ctx, r.db).Where("user_id = ?", userID).Order("id DESC").Find(&list).Error
	return list, err
}

// Delete 删除凭据。
func (r *S3CredentialRepo) Delete(ctx context.Context, id uint64) error {
	return conn(ctx, r.db).Delete(&model.S3Credential{}, id).Error
}

// UpdateFields 更新凭据字段。
func (r *S3CredentialRepo) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
	return conn(ctx, r.db).Model(&model.S3Credential{}).Where("id = ?", id).Updates(fields).Error
}

// IncrCallCount 增加调用计数。
func (r *S3CredentialRepo) IncrCallCount(ctx context.Context, id uint64, ip string) error {
	return conn(ctx, r.db).Model(&model.S3Credential{}).Where("id = ?", id).Updates(map[string]any{
		"call_count":   gorm.Expr("call_count + 1"),
		"last_used_at": model.Now(),
		"last_used_ip": ip,
	}).Error
}
