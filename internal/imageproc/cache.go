package imageproc

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// cacheExts 是缓存文件可能使用的扩展名，按优先级尝试。
var cacheExts = []string{"webp", "jpg", "png", "gif"}

// Cache 是基于磁盘的图片处理结果缓存，按 TTL 与容量上限清理。
type Cache struct {
	dir     string
	maxSize int64
	ttl     time.Duration

	mu          sync.Mutex
	lastCleanup time.Time
}

// NewCache 创建缓存目录。
func NewCache(dir string, maxSize int64, ttl time.Duration) (*Cache, error) {
	if dir == "" {
		return nil, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Cache{dir: dir, maxSize: maxSize, ttl: ttl}, nil
}

// Dir 返回缓存根目录。
func (c *Cache) Dir() string { return c.dir }

// path 返回缓存文件路径：dir/ab/<key>.<ext>。
func (c *Cache) path(key, ext string) string {
	prefix := "xx"
	if len(key) >= 2 {
		prefix = key[:2]
	}
	return filepath.Join(c.dir, prefix, key+"."+strings.TrimPrefix(ext, "."))
}

// Get 读取指定格式的缓存，命中时刷新访问时间。
func (c *Cache) Get(key, format string) ([]byte, bool) {
	if c == nil {
		return nil, false
	}
	p := c.path(key, ExtOf(format))
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	now := time.Now()
	_ = os.Chtimes(p, now, now)
	return data, true
}

// GetAny 依次尝试各已支持格式读取缓存，返回数据与实际格式。
func (c *Cache) GetAny(key string) ([]byte, string, bool) {
	if c == nil {
		return nil, "", false
	}
	for _, ext := range cacheExts {
		if data, ok := c.Get(key, ext); ok {
			return data, FormatOfExt(ext), true
		}
	}
	return nil, "", false
}

// Put 写入缓存文件，并择机触发后台清理。
func (c *Cache) Put(key, format string, data []byte) error {
	if c == nil || len(data) == 0 {
		return nil
	}
	if c.maxSize > 0 && int64(len(data)) > c.maxSize {
		return nil
	}
	p := c.path(key, ExtOf(format))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".cache-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, p); err != nil {
		return err
	}
	c.maybeCleanup()
	return nil
}

// maybeCleanup 每 30 分钟最多触发一次后台清理。
func (c *Cache) maybeCleanup() {
	c.mu.Lock()
	if time.Since(c.lastCleanup) < 30*time.Minute {
		c.mu.Unlock()
		return
	}
	c.lastCleanup = time.Now()
	c.mu.Unlock()
	go func() { _, _, _ = c.Cleanup() }()
}

type cacheEntry struct {
	path    string
	size    int64
	modTime time.Time
}

// Cleanup 清理过期缓存，并在总量超限时按最久未访问淘汰，返回删除数量与释放字节数。
func (c *Cache) Cleanup() (removed int, freed int64, err error) {
	if c == nil {
		return 0, 0, nil
	}
	var entries []cacheEntry
	now := time.Now()
	walkErr := filepath.WalkDir(c.dir, func(path string, d os.DirEntry, werr error) error {
		if werr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		// 清理中断的临时文件。
		if strings.HasPrefix(d.Name(), ".cache-") {
			if now.Sub(info.ModTime()) > time.Hour {
				if os.Remove(path) == nil {
					removed++
					freed += info.Size()
				}
			}
			return nil
		}
		if c.ttl > 0 && now.Sub(info.ModTime()) > c.ttl {
			if os.Remove(path) == nil {
				removed++
				freed += info.Size()
			}
			return nil
		}
		entries = append(entries, cacheEntry{path: path, size: info.Size(), modTime: info.ModTime()})
		return nil
	})
	if walkErr != nil {
		return removed, freed, walkErr
	}
	if c.maxSize <= 0 {
		return removed, freed, nil
	}
	var total int64
	for _, e := range entries {
		total += e.size
	}
	if total <= c.maxSize {
		return removed, freed, nil
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].modTime.Before(entries[j].modTime) })
	target := c.maxSize * 9 / 10
	for _, e := range entries {
		if total <= target {
			break
		}
		if os.Remove(e.path) == nil {
			total -= e.size
			removed++
			freed += e.size
		}
	}
	return removed, freed, nil
}
