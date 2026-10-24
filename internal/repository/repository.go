// Package repository 实现数据访问层，只操作 MySQL 并返回 model。
package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// Repos 聚合全部仓储实例。
type Repos struct {
	DB      *gorm.DB
	Tx      *TxManager
	User    *UserRepo
	Group   *UserGroupRepo
	Image   *ImageRepo
	Album   *AlbumRepo
	Tag     *TagRepo
	Token   *APITokenRepo
	S3Cred  *S3CredentialRepo
	Share   *ShareRepo
	Referer *RefererRepo
	Storage *StorageRepo
	Email   *EmailRepo
	Setting *SettingRepo
	Invite  *InviteRepo
	Log     *LogRepo
	Counter *CounterRepo
}

// New 构造函数化全部仓储。
func New(db *gorm.DB) *Repos {
	return &Repos{
		DB:      db,
		Tx:      &TxManager{db: db},
		User:    &UserRepo{db: db},
		Group:   &UserGroupRepo{db: db},
		Image:   &ImageRepo{db: db},
		Album:   &AlbumRepo{db: db},
		Tag:     &TagRepo{db: db},
		Token:   &APITokenRepo{db: db},
		S3Cred:  &S3CredentialRepo{db: db},
		Share:   &ShareRepo{db: db},
		Referer: &RefererRepo{db: db},
		Storage: &StorageRepo{db: db},
		Email:   &EmailRepo{db: db},
		Setting: &SettingRepo{db: db},
		Invite:  &InviteRepo{db: db},
		Log:     &LogRepo{db: db},
		Counter: &CounterRepo{db: db},
	}
}

// conn 返回事务上下文中的连接，未处于事务时返回普通连接。
func conn(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}

func notFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

// pageOffset 计算分页偏移，页码从 1 开始。
func pageOffset(page, size int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * size
}

func normPage(page, size, defSize, maxSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = defSize
	}
	if size > maxSize {
		size = maxSize
	}
	return page, size
}
