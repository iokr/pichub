// Package manager 负责存储策略解析、驱动构造与实例缓存。
package manager

import (
	"fmt"

	"github.com/pichub/pichub/internal/storage"
	"github.com/pichub/pichub/internal/storage/local"
	"github.com/pichub/pichub/internal/storage/s3"
	"github.com/pichub/pichub/internal/storage/webdav"
)

// NewDriver 根据配置创建存储驱动实例。
func NewDriver(dc storage.DriverConfig) (storage.Storage, error) {
	switch dc.Driver {
	case storage.DriverLocal, "":
		return local.New(dc.Local, dc.Domain), nil
	case storage.DriverS3:
		return s3.New(storage.DriverS3, dc.S3, dc.Domain)
	case storage.DriverOSS:
		return s3.New(storage.DriverOSS, dc.OSS, dc.Domain)
	case storage.DriverCOS:
		return s3.New(storage.DriverCOS, dc.COS, dc.Domain)
	case storage.DriverWebDAV:
		return webdav.New(dc.WebDAV, dc.Domain)
	default:
		return nil, fmt.Errorf("%w: %s", storage.ErrUnsupported, dc.Driver)
	}
}
