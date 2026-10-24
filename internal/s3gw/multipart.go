package s3gw

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// multipartTTL 是分片会话的保留时长。
const multipartTTL = 24 * time.Hour

// multipartPart 是已上传的分片。
type multipartPart struct {
	Path string
	Size int64
	MD5  string
}

// multipartUpload 是一次分片上传会话。
type multipartUpload struct {
	ID        string
	UserID    uint64
	Bucket    string
	Key       string
	Dir       string
	CreatedAt time.Time

	mu    sync.Mutex
	parts map[int]*multipartPart
}

// multipartStore 管理进程内的分片上传会话。
type multipartStore struct {
	mu    sync.Mutex
	items map[string]*multipartUpload
}

func newMultipartStore() *multipartStore {
	return &multipartStore{items: map[string]*multipartUpload{}}
}

// create 创建分片上传会话。
func (s *multipartStore) create(userID uint64, bucket, key string) (*multipartUpload, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	id := hex.EncodeToString(buf)
	dir, err := os.MkdirTemp("", "pichub-s3mp-")
	if err != nil {
		return nil, err
	}
	up := &multipartUpload{
		ID:        id,
		UserID:    userID,
		Bucket:    bucket,
		Key:       key,
		Dir:       dir,
		CreatedAt: time.Now(),
		parts:     map[int]*multipartPart{},
	}
	s.mu.Lock()
	s.items[id] = up
	s.cleanupLocked(time.Now().Add(-multipartTTL))
	s.mu.Unlock()
	return up, nil
}

// get 返回未过期的分片会话。
func (s *multipartStore) get(id string) *multipartUpload {
	if id == "" {
		return nil
	}
	s.mu.Lock()
	up := s.items[id]
	s.mu.Unlock()
	if up == nil {
		return nil
	}
	if time.Since(up.CreatedAt) > multipartTTL {
		s.drop(id)
		return nil
	}
	return up
}

// drop 删除分片会话及其临时文件。
func (s *multipartStore) drop(id string) {
	s.mu.Lock()
	up := s.items[id]
	delete(s.items, id)
	s.mu.Unlock()
	if up != nil {
		_ = os.RemoveAll(up.Dir)
	}
}

// savePart 把一个已落盘的分片移入会话目录。
func (s *multipartStore) savePart(up *multipartUpload, number int, tmpPath string, size int64, md5hex string) error {
	up.mu.Lock()
	defer up.mu.Unlock()
	dst := filepath.Join(up.Dir, fmt.Sprintf("%05d.part", number))
	if err := os.Rename(tmpPath, dst); err != nil {
		// 跨文件系统时退化为复制。
		if cerr := copyFile(tmpPath, dst); cerr != nil {
			return cerr
		}
		_ = os.Remove(tmpPath)
	}
	up.parts[number] = &multipartPart{Path: dst, Size: size, MD5: md5hex}
	return nil
}

// assemble 按 Part 列表合并分片，返回临时文件与本分片 ETag。
func (s *multipartStore) assemble(up *multipartUpload, requested []completePart) (string, int64, string, error) {
	up.mu.Lock()
	numbers := make([]int, 0, len(up.parts))
	if len(requested) > 0 {
		for _, p := range requested {
			numbers = append(numbers, p.PartNumber)
		}
	} else {
		for n := range up.parts {
			numbers = append(numbers, n)
		}
		sort.Ints(numbers)
	}
	parts := make([]*multipartPart, 0, len(numbers))
	for i, n := range numbers {
		p := up.parts[n]
		if p == nil {
			up.mu.Unlock()
			return "", 0, "", fmt.Errorf("分片 %d 不存在", n)
		}
		if len(requested) > 0 {
			want := trimETag(requested[i].ETag)
			if want != "" && !strings.EqualFold(want, p.MD5) {
				up.mu.Unlock()
				return "", 0, "", fmt.Errorf("分片 %d 的 ETag 不匹配", n)
			}
		}
		parts = append(parts, p)
	}
	up.mu.Unlock()

	out, err := os.CreateTemp("", "pichub-s3merge-*.tmp")
	if err != nil {
		return "", 0, "", err
	}
	name := out.Name()
	var total int64
	etagHash := md5.New()
	for _, p := range parts {
		f, oerr := os.Open(p.Path)
		if oerr != nil {
			out.Close()
			os.Remove(name)
			return "", 0, "", oerr
		}
		n, cerr := io.Copy(out, f)
		f.Close()
		if cerr != nil {
			out.Close()
			os.Remove(name)
			return "", 0, "", cerr
		}
		total += n
		raw, derr := hex.DecodeString(p.MD5)
		if derr != nil {
			raw = []byte(p.MD5)
		}
		_, _ = etagHash.Write(raw)
	}
	if err := out.Close(); err != nil {
		os.Remove(name)
		return "", 0, "", err
	}
	etag := hex.EncodeToString(etagHash.Sum(nil)) + "-" + fmt.Sprint(len(parts))
	return name, total, etag, nil
}

func (s *multipartStore) cleanupLocked(before time.Time) {
	for id, up := range s.items {
		if up.CreatedAt.After(before) {
			continue
		}
		delete(s.items, id)
		_ = os.RemoveAll(up.Dir)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func trimETag(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
