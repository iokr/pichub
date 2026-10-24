// Package service 实现业务逻辑。本包不依赖 Gin 与具体数据访问实现，
// 所有仓储以接口形式定义在此（消费方定义接口），由 bootstrap 注入。
package service

import (
	"context"
	"time"

	"github.com/pichub/pichub/internal/model"
	"github.com/pichub/pichub/internal/storage"
)

// Tx 是事务管理器抽象。
type Tx interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// UserRepo 用户仓储。
type UserRepo interface {
	Create(ctx context.Context, u *model.User) error
	GetByID(ctx context.Context, id uint64) (*model.User, error)
	GetByUsername(ctx context.Context, name string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Count(ctx context.Context) (int64, error)
	CountByIP(ctx context.Context, ip string, since time.Time) (int64, error)
	List(ctx context.Context, q model.UserQuery) ([]*model.User, int64, error)
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
	AddUsedBytes(ctx context.Context, id uint64, delta, quota int64) (bool, error)
	SubUsedBytes(ctx context.Context, id uint64, delta int64) error
	CountAdmins(ctx context.Context) (int64, error)
}

// UserGroupRepo 用户组仓储。
type UserGroupRepo interface {
	Create(ctx context.Context, g *model.UserGroup) error
	GetByID(ctx context.Context, id uint64) (*model.UserGroup, error)
	GetDefault(ctx context.Context) (*model.UserGroup, error)
	List(ctx context.Context) ([]*model.UserGroup, error)
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
	Delete(ctx context.Context, id uint64) error
	CountUsers(ctx context.Context, id uint64) (int64, error)
}

// ImageRepo 图片仓储。
type ImageRepo interface {
	Create(ctx context.Context, img *model.Image) error
	GetByID(ctx context.Context, id uint64) (*model.Image, error)
	GetByUserHash(ctx context.Context, userID uint64, hash string) (*model.Image, error)
	GetByHash(ctx context.Context, hash string) (*model.Image, error)
	GetByHashAny(ctx context.Context, hash string) (*model.Image, error)
	GetByStorageKey(ctx context.Context, key string) (*model.Image, error)
	GetByUserStorageKey(ctx context.Context, userID uint64, key string) (*model.Image, error)
	GetPublicByStorageKey(ctx context.Context, key string) (*model.Image, error)
	ListStorageObjects(ctx context.Context, userID uint64, prefix, after string, limit int) ([]*model.Image, error)
	List(ctx context.Context, q model.ImageQuery) ([]*model.Image, int64, error)
	SoftDelete(ctx context.Context, id uint64, deletedFlag int64, at time.Time) error
	Restore(ctx context.Context, id uint64) error
	HardDelete(ctx context.Context, id uint64) error
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
	IncrRefCount(ctx context.Context, id uint64, delta int) error
	ListTrashExpired(ctx context.Context, before time.Time, limit int) ([]*model.Image, error)
	CountByUser(ctx context.Context, userID uint64, status int8) (int64, error)
	SumSizeByUser(ctx context.Context, userID uint64) (int64, error)
	CountAll(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status int8) (int64, error)
	SumSizeAll(ctx context.Context) (int64, error)
	CountCreatedSince(ctx context.Context, since time.Time) (int64, error)
}

// AlbumRepo 相册仓储。
type AlbumRepo interface {
	Create(ctx context.Context, a *model.Album) error
	GetByID(ctx context.Context, id uint64) (*model.Album, error)
	ListByUser(ctx context.Context, userID uint64) ([]*model.Album, error)
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
	Delete(ctx context.Context, id uint64) error
	CountByUser(ctx context.Context, userID uint64) (int64, error)
	CountAll(ctx context.Context) (int64, error)
	IncrImageCount(ctx context.Context, id uint64, delta int) error
	DetachImages(ctx context.Context, albumID uint64) error
	RecalcImageCount(ctx context.Context, albumID uint64) error
}

// TagRepo 标签仓储。
type TagRepo interface {
	Create(ctx context.Context, t *model.Tag) error
	GetByID(ctx context.Context, id uint64) (*model.Tag, error)
	GetByUserName(ctx context.Context, userID uint64, name string) (*model.Tag, error)
	ListByUser(ctx context.Context, userID uint64) ([]*model.Tag, error)
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
	Delete(ctx context.Context, id uint64) error
	IncrImageCount(ctx context.Context, id uint64, delta int) error
	Attach(ctx context.Context, imageID uint64, tagIDs []uint64) error
	DetachAll(ctx context.Context, imageID uint64) error
	DetachByTag(ctx context.Context, tagID uint64) error
	ListByImage(ctx context.Context, imageID uint64) ([]uint64, error)
	ListImagesByTags(ctx context.Context, tagIDs []uint64) ([]uint64, error)
}

// APITokenRepo API 令牌仓储。
type APITokenRepo interface {
	Create(ctx context.Context, t *model.APIToken) error
	GetByID(ctx context.Context, id uint64) (*model.APIToken, error)
	GetByHash(ctx context.Context, hash string) (*model.APIToken, error)
	ListByUser(ctx context.Context, userID uint64) ([]*model.APIToken, error)
	Delete(ctx context.Context, id uint64) error
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
	CountByUser(ctx context.Context, userID uint64) (int64, error)
	IncrCallCount(ctx context.Context, id uint64, ip string) error
}

// S3CredentialRepo S3 凭据仓储。
type S3CredentialRepo interface {
	Create(ctx context.Context, c *model.S3Credential) error
	GetByID(ctx context.Context, id uint64) (*model.S3Credential, error)
	GetByAccessKey(ctx context.Context, ak string) (*model.S3Credential, error)
	ListByUser(ctx context.Context, userID uint64) ([]*model.S3Credential, error)
	Delete(ctx context.Context, id uint64) error
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
	IncrCallCount(ctx context.Context, id uint64, ip string) error
}

// ShareRepo 分享仓储。
type ShareRepo interface {
	Create(ctx context.Context, s *model.Share) error
	GetByID(ctx context.Context, id uint64) (*model.Share, error)
	GetBySlug(ctx context.Context, slug string) (*model.Share, error)
	ListByUser(ctx context.Context, userID uint64) ([]*model.Share, error)
	Delete(ctx context.Context, id uint64) error
	IncrViews(ctx context.Context, id uint64) error
	CountByUser(ctx context.Context, userID uint64) (int64, error)
	CountAll(ctx context.Context) (int64, error)
	DeleteByTarget(ctx context.Context, targetType string, targetID uint64) error
}

// RefererRepo 引用统计仓储。
type RefererRepo interface {
	UpsertBatch(ctx context.Context, items []*model.ImageReferer) error
	UpsertDailyBatch(ctx context.Context, date model.Date, items []*model.ImageReferer) error
	ListByImage(ctx context.Context, q model.RefererQuery) ([]*model.ImageReferer, int64, error)
	Summary(ctx context.Context, imageID uint64) (totalHits, hostCount int64, err error)
	Delete(ctx context.Context, id, imageID uint64) error
	DeleteByImage(ctx context.Context, imageID uint64) error
	TopHosts(ctx context.Context, userID uint64, since time.Time, limit int) ([]*model.HostStat, error)
	Trend(ctx context.Context, userID uint64, from, to time.Time) ([]*model.DayStat, error)
	AggregateDaily(ctx context.Context, date model.Date) error
	DeleteDailyBefore(ctx context.Context, date model.Date) error
	TotalHits(ctx context.Context) (int64, error)
	SumHitsByImage(ctx context.Context, imageID uint64) (int64, error)
	SumHitsByUser(ctx context.Context, userID uint64) (int64, error)
}

// StorageRepo 存储策略仓储。
type StorageRepo interface {
	Create(ctx context.Context, s *model.StorageConfig) error
	GetByID(ctx context.Context, id uint64) (*model.StorageConfig, error)
	GetDefault(ctx context.Context) (*model.StorageConfig, error)
	ListSystem(ctx context.Context) ([]*model.StorageConfig, error)
	ListByUser(ctx context.Context, userID uint64) ([]*model.StorageConfig, error)
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
	Delete(ctx context.Context, id uint64) error
	ClearDefault(ctx context.Context) error
	CountByUser(ctx context.Context, userID uint64) (int64, error)
}

// EmailRepo 邮件仓储。
type EmailRepo interface {
	CreateToken(ctx context.Context, t *model.EmailToken) error
	GetLatestToken(ctx context.Context, userID uint64, scene string) (*model.EmailToken, error)
	GetTokenByHash(ctx context.Context, hash string) (*model.EmailToken, error)
	MarkTokenUsed(ctx context.Context, id uint64, at time.Time) error
	DeleteExpiredTokens(ctx context.Context, before time.Time) error
	CreateLog(ctx context.Context, l *model.EmailLog) error
	UpdateLog(ctx context.Context, id uint64, fields map[string]any) error
	ListLogs(ctx context.Context, q model.EmailLogQuery) ([]*model.EmailLog, int64, error)
	DeleteLogsBefore(ctx context.Context, before time.Time) error
	CountTodaySuccess(ctx context.Context, since time.Time) (int64, error)
}

// SettingRepo 系统设置仓储。
type SettingRepo interface {
	GetAll(ctx context.Context) ([]*model.SystemSetting, error)
	Get(ctx context.Context, key string) (*model.SystemSetting, error)
	Set(ctx context.Context, key, value, valueType string, isSecret bool, desc string) error
	SetMany(ctx context.Context, items []*model.SystemSetting) error
}

// InviteRepo 邀请码仓储。
type InviteRepo interface {
	Create(ctx context.Context, i *model.Invite) error
	GetByCode(ctx context.Context, code string) (*model.Invite, error)
	GetByID(ctx context.Context, id uint64) (*model.Invite, error)
	List(ctx context.Context, page, size int) ([]*model.Invite, int64, error)
	Delete(ctx context.Context, id uint64) error
	IncrUsed(ctx context.Context, id uint64) (bool, error)
}

// LogRepo 操作日志仓储。
type LogRepo interface {
	Create(ctx context.Context, l *model.OperationLog) error
	List(ctx context.Context, q model.LogQuery) ([]*model.OperationLog, int64, error)
	DeleteBefore(ctx context.Context, before time.Time) error
}

// CounterRepo 每日上传用量仓储。
type CounterRepo interface {
	Incr(ctx context.Context, userID uint64, date model.Date, count int, bytes int64) error
	Get(ctx context.Context, userID uint64, date model.Date) (*model.UploadCounter, error)
	SumSince(ctx context.Context, userID uint64, from model.Date) (int64, int64, error)
	Add(ctx context.Context, userID uint64, date model.Date, bytes int64) error
}

// Repos 聚合业务层所需的全部仓储接口。
type Repos struct {
	Tx      Tx
	User    UserRepo
	Group   UserGroupRepo
	Image   ImageRepo
	Album   AlbumRepo
	Tag     TagRepo
	Token   APITokenRepo
	S3Cred  S3CredentialRepo
	Share   ShareRepo
	Referer RefererRepo
	Storage StorageRepo
	Email   EmailRepo
	Setting SettingRepo
	Invite  InviteRepo
	Log     LogRepo
	Counter CounterRepo
}

// StorageManager 是存储策略管理能力抽象。
type StorageManager interface {
	Resolve(ctx context.Context, user *model.User, requestedID uint64) (storage.Storage, *model.StorageConfig, error)
	Build(ctx context.Context, sc *model.StorageConfig) (storage.Storage, error)
	PublicURL(sc *model.StorageConfig, st storage.Storage, key string) string
	Test(ctx context.Context, sc *model.StorageConfig) error
	Invalidate(id uint64)
	AllowUserStrategy() bool
	FallbackDriver() string
	EnsureDefault(ctx context.Context) error
}

// Enqueuer 是异步任务入队抽象。
type Enqueuer interface {
	Enqueue(ctx context.Context, taskType string, payload any) error
}

// QueueStats 是队列统计能力抽象。
type QueueStats interface {
	Stats(ctx context.Context) (ready, delayed, dead int64, err error)
}
