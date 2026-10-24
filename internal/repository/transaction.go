package repository

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

// TxManager 基于 GORM 的事务管理器。
type TxManager struct {
	db *gorm.DB
}

// Do 在事务中执行 fn，fn 内部通过 context 获取事务连接。
func (t *TxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		// 已在事务中，直接复用。
		return fn(ctx)
	}
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}
