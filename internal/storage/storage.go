// Package storage 定义存储驱动抽象、工厂与策略管理器。
package storage

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/pichub/pichub/internal/config"
)

// 驱动名称。
const (
	DriverLocal  = "local"
	DriverS3     = "s3"
	DriverOSS    = "oss"
	DriverCOS    = "cos"
	DriverWebDAV = "webdav"
)

// 错误定义。
var (
	ErrInvalidKey  = errors.New("invalid storage key")
	ErrNotFound    = errors.New("object not found")
	ErrUnsupported = errors.New("unsupported storage driver")
)

// FileInfo 对象元信息。
type FileInfo struct {
	Size        int64
	ContentType string
	ModTime     time.Time
}

// Storage 是统一的存储驱动接口。
type Storage interface {
	// Driver 返回驱动名称。
	Driver() string
	// Put 写入对象；实现需保证原子性（先写临时文件再重命名）。
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	// Get 读取对象。
	Get(ctx context.Context, key string) (io.ReadCloser, *FileInfo, error)
	// Stat 获取对象元信息。
	Stat(ctx context.Context, key string) (*FileInfo, error)
	// Delete 删除对象。
	Delete(ctx context.Context, key string) error
	// PublicURL 返回对象的公开访问地址。
	PublicURL(key string) string
	// Test 连通性测试。
	Test(ctx context.Context) error
}

// LocalPathProvider 由支持直接读取本地路径的驱动实现，便于 http.ServeContent。
type LocalPathProvider interface {
	LocalPath(key string) (string, bool)
}

// DriverConfig 是构造驱动实例所需的配置。
type DriverConfig struct {
	Driver     string
	Domain     string
	PathPrefix string
	Local      config.LocalStorageConfig
	S3         config.ObjectStorageConfig
	OSS        config.ObjectStorageConfig
	COS        config.ObjectStorageConfig
	WebDAV     config.WebDAVConfig
}

// JoinURL 拼接域名与 Key，避免重复斜杠。
func JoinURL(base, key string) string {
	if base == "" {
		return key
	}
	for len(base) > 0 && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	if key == "" {
		return base
	}
	if key[0] == '/' {
		return base + key
	}
	return base + "/" + key
}
