// Package webdav 实现 WebDAV 存储驱动。
package webdav

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/storage"
)

// Storage 是 WebDAV 存储驱动。
type Storage struct {
	baseURL   string
	publicURL string
	domain    string
	username  string
	password  string
	client    *http.Client
}

// New 创建 WebDAV 驱动。
func New(cfg config.WebDAVConfig, domain string) (*Storage, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("webdav storage: url is required")
	}
	return &Storage{
		baseURL:   strings.TrimRight(cfg.URL, "/"),
		publicURL: cfg.PublicURL,
		domain:    domain,
		username:  cfg.Username,
		password:  cfg.Password,
		client:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Driver 返回驱动名称。
func (s *Storage) Driver() string { return storage.DriverWebDAV }

func (s *Storage) objectURL(key string) string {
	segs := strings.Split(strings.Trim(key, "/"), "/")
	for i, seg := range segs {
		segs[i] = url.PathEscape(seg)
	}
	return s.baseURL + "/" + strings.Join(segs, "/")
}

func (s *Storage) newRequest(ctx context.Context, method, target string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	if s.username != "" {
		req.SetBasicAuth(s.username, s.password)
	}
	return req, nil
}

func (s *Storage) mkdirAll(ctx context.Context, key string) error {
	dir := path.Dir(strings.Trim(key, "/"))
	if dir == "." || dir == "/" {
		return nil
	}
	parts := strings.Split(dir, "/")
	current := ""
	for _, p := range parts {
		current = current + "/" + p
		target := s.baseURL + current
		req, err := s.newRequest(ctx, "MKCOL", target, nil)
		if err != nil {
			return err
		}
		resp, err := s.client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		// 405 表示目录已存在。
		if resp.StatusCode >= 300 && resp.StatusCode != http.StatusMethodNotAllowed {
			return fmt.Errorf("webdav mkcol %s: status %d", current, resp.StatusCode)
		}
	}
	return nil
}

// Put 上传对象。
func (s *Storage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	if err := s.mkdirAll(ctx, key); err != nil {
		return err
	}
	req, err := s.newRequest(ctx, http.MethodPut, s.objectURL(key), r)
	if err != nil {
		return err
	}
	req.ContentLength = size
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webdav put: status %d", resp.StatusCode)
	}
	return nil
}

// Get 下载对象。
func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, *storage.FileInfo, error) {
	req, err := s.newRequest(ctx, http.MethodGet, s.objectURL(key), nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, nil, storage.ErrNotFound
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("webdav get: status %d", resp.StatusCode)
	}
	return resp.Body, &storage.FileInfo{
		Size:        resp.ContentLength,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

// Stat 获取对象元信息。
func (s *Storage) Stat(ctx context.Context, key string) (*storage.FileInfo, error) {
	req, err := s.newRequest(ctx, http.MethodHead, s.objectURL(key), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, storage.ErrNotFound
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("webdav head: status %d", resp.StatusCode)
	}
	return &storage.FileInfo{Size: resp.ContentLength, ContentType: resp.Header.Get("Content-Type")}, nil
}

// Delete 删除对象。
func (s *Storage) Delete(ctx context.Context, key string) error {
	req, err := s.newRequest(ctx, http.MethodDelete, s.objectURL(key), nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("webdav delete: status %d", resp.StatusCode)
	}
	return nil
}

// PublicURL 返回对象公开地址。
func (s *Storage) PublicURL(key string) string {
	base := s.publicURL
	if base == "" {
		base = s.domain
	}
	if base == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(key, "/")
}

// Test 访问根目录验证连通性。
func (s *Storage) Test(ctx context.Context) error {
	req, err := s.newRequest(ctx, http.MethodOptions, s.baseURL, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webdav: status %d", resp.StatusCode)
	}
	return nil
}
