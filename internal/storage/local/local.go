// Package local 实现本地磁盘存储驱动。
package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/storage"
)

// Storage 是本地磁盘存储。
type Storage struct {
	root     string
	domain   string
	dirPerm  os.FileMode
	filePerm os.FileMode
}

// New 创建本地存储驱动。
func New(cfg config.LocalStorageConfig, domain string) *Storage {
	dirPerm := os.FileMode(cfg.DirPermission)
	if dirPerm == 0 {
		dirPerm = 0o755
	}
	filePerm := os.FileMode(cfg.FilePermission)
	if filePerm == 0 {
		filePerm = 0o644
	}
	return &Storage{
		root:     cfg.RootPath,
		domain:   domain,
		dirPerm:  dirPerm,
		filePerm: filePerm,
	}
}

// Driver 返回驱动名称。
func (s *Storage) Driver() string { return storage.DriverLocal }

// absPath 计算并校验对象绝对路径，防止路径穿越。
func (s *Storage) absPath(key string) (string, error) {
	clean := filepath.Clean(strings.TrimPrefix(filepath.FromSlash(key), string(filepath.Separator)))
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", storage.ErrInvalidKey
	}
	root := filepath.Clean(s.root)
	abs := filepath.Join(root, clean)
	if !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		return "", storage.ErrInvalidKey
	}
	return abs, nil
}

// Put 原子写入对象：先写 .tmp 并 fsync，再 rename。
func (s *Storage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	abs, err := s.absPath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), s.dirPerm); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(abs), ".upload-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, s.filePerm); err != nil {
		return err
	}
	return os.Rename(tmpName, abs)
}

// Get 打开对象。
func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, *storage.FileInfo, error) {
	abs, err := s.absPath(key)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, storage.ErrNotFound
		}
		return nil, nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	return f, &storage.FileInfo{Size: st.Size(), ModTime: st.ModTime()}, nil
}

// Stat 获取对象信息。
func (s *Storage) Stat(ctx context.Context, key string) (*storage.FileInfo, error) {
	abs, err := s.absPath(key)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	return &storage.FileInfo{Size: st.Size(), ModTime: st.ModTime()}, nil
}

// Delete 删除对象。
func (s *Storage) Delete(ctx context.Context, key string) error {
	abs, err := s.absPath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// PublicURL 返回访问地址（本地驱动由 /i 路由直出）。
func (s *Storage) PublicURL(key string) string {
	if s.domain == "" {
		return ""
	}
	return join(s.domain, key)
}

// LocalPath 返回本地绝对路径，供 http.ServeContent 使用。
func (s *Storage) LocalPath(key string) (string, bool) {
	abs, err := s.absPath(key)
	if err != nil {
		return "", false
	}
	return abs, true
}

// Test 写入并删除探针文件验证可用性。
func (s *Storage) Test(ctx context.Context) error {
	if err := os.MkdirAll(s.root, s.dirPerm); err != nil {
		return err
	}
	probe := fmt.Sprintf(".pichub-probe-%d", time.Now().UnixNano())
	path := filepath.Join(s.root, probe)
	if err := os.WriteFile(path, []byte("ok"), s.filePerm); err != nil {
		return err
	}
	return os.Remove(path)
}

func join(base, key string) string {
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
