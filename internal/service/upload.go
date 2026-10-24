package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pichub/pichub/internal/auth"
	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/imageproc"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
	"github.com/pichub/pichub/internal/storage"
	"github.com/pichub/pichub/internal/storage/manager"
)

// chunkSessionTTL 是分片会话的保留时长。
const chunkSessionTTL = 24 * time.Hour

// defaultMaxFileSize 是单文件大小的兜底上限。
const defaultMaxFileSize = 10 << 20

// UploadOptions 是上传请求的附加选项。
type UploadOptions struct {
	Filename  string
	AlbumID   *uint64
	Tags      string
	IsPublic  *bool
	StorageID uint64
	ObjectKey string
	IP        string
	UserAgent string
}

// chunkSession 是进程内的分片上传会话。
type chunkSession struct {
	ID        string
	UserID    uint64
	Name      string
	Size      int64
	Hash      string
	Count     int
	AlbumID   *uint64
	Tags      string
	IsPublic  *bool
	StorageID uint64
	Dir       string
	CreatedAt time.Time

	mu       sync.Mutex
	received map[int]int64
}

// UploadService 处理图片上传、秒传、远程拉取与分片上传。
type UploadService struct {
	cfg       *config.Config
	log       *logger.Logger
	repos     Repos
	settings  *SettingsService
	images    *ImageService
	storages  StorageManager
	client    *http.Client
	chunkRoot string

	chunkMu sync.Mutex
	chunks  map[string]*chunkSession
}

// NewUploadService 创建上传服务。
func NewUploadService(d Deps, settings *SettingsService, images *ImageService) *UploadService {
	s := &UploadService{
		cfg:       d.Config,
		log:       d.Log,
		repos:     d.Repos,
		settings:  settings,
		images:    images,
		storages:  d.Storages,
		chunkRoot: filepath.Join(os.TempDir(), "pichub-chunks"),
		chunks:    make(map[string]*chunkSession),
	}
	s.client = &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 禁止重定向，避免绕过内网地址校验。
			return http.ErrUseLastResponse
		},
	}
	return s
}

// uploadLimits 是解析后的上传限制。
type uploadLimits struct {
	maxFileSize    int64
	dailyLimit     int
	exts           map[string]bool
	allowSVG       bool
	dedupEnabled   bool
	dedupScope     string
	defaultPublic  bool
	keepOriginName bool
}

func (l *uploadLimits) allowsExt(ext string) bool {
	ext = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))
	if ext == "jpeg" {
		ext = "jpg"
	}
	if ext == "tif" {
		ext = "tiff"
	}
	if ext == "svg" && !l.allowSVG {
		return false
	}
	if l.exts[ext] {
		return true
	}
	if ext == "jpg" {
		return l.exts["jpeg"] || l.exts["jpg"]
	}
	return false
}

// UploadObject 处理 S3 网关 PutObject：客户端对象 Key 直接作为存储 Key。
func (s *UploadService) UploadObject(ctx context.Context, user *model.User, objectKey string, r io.Reader, size int64, ip, ua string) (*dto.UploadResp, error) {
	key := normalizeObjectKey(objectKey)
	if key == "" {
		return nil, errs.ErrInvalidParam.WithDetail("对象 Key 无效")
	}
	if err := s.checkUploadPermission(ctx, user); err != nil {
		return nil, err
	}
	limits := s.newLimits(ctx, user)
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(key), "."))
	if ext != "" && !limits.allowsExt(ext) {
		return nil, errs.ErrInvalidParam.WithDetail("不支持的文件类型: " + ext)
	}
	if size > 0 && limits.maxFileSize > 0 && size > limits.maxFileSize {
		return nil, errs.ErrFileTooLarge.WithDetail(fmt.Sprintf("单文件不能超过 %d MB", limits.maxFileSize>>20))
	}
	tmp, n, hash, err := s.saveTemp(r, limits.maxFileSize)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp)
	return s.persist(ctx, user, tmp, n, hash, UploadOptions{
		Filename:  path.Base(key),
		ObjectKey: key,
		IP:        ip,
		UserAgent: ua,
	})
}

// Upload 从数据流读取并保存图片。
func (s *UploadService) Upload(ctx context.Context, user *model.User, r io.Reader, size int64, opts UploadOptions) (*dto.UploadResp, error) {
	if err := s.checkUploadPermission(ctx, user); err != nil {
		return nil, err
	}
	limits := s.newLimits(ctx, user)
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(opts.Filename), "."))
	if ext != "" && !limits.allowsExt(ext) {
		return nil, errs.ErrInvalidParam.WithDetail("不支持的文件类型: " + ext)
	}
	if size > 0 && limits.maxFileSize > 0 && size > limits.maxFileSize {
		return nil, errs.ErrFileTooLarge.WithDetail(fmt.Sprintf("单文件不能超过 %d MB", limits.maxFileSize>>20))
	}
	tmp, n, hash, err := s.saveTemp(r, limits.maxFileSize)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp)
	return s.persist(ctx, user, tmp, n, hash, opts)
}

// Check 执行秒传检测（仅本用户内去重）。
func (s *UploadService) Check(ctx context.Context, user *model.User, req *dto.UploadCheckReq) (*dto.UploadCheckResp, error) {
	hash := strings.ToLower(strings.TrimSpace(req.Hash))
	if !validSHA256(hash) {
		return nil, errs.ErrInvalidParam.WithDetail("文件哈希格式无效")
	}
	img, err := s.repos.Image.GetByUserHash(ctx, user.ID, hash)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if img == nil {
		return &dto.UploadCheckResp{Exists: false}, nil
	}
	return &dto.UploadCheckResp{Exists: true, Image: s.resp(img, true)}, nil
}

// UploadURL 拉取远程图片并保存。
func (s *UploadService) UploadURL(ctx context.Context, user *model.User, req *dto.UploadURLReq, ip, ua string) (*dto.UploadResp, error) {
	if err := s.checkUploadPermission(ctx, user); err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(req.URL)
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errs.ErrInvalidParam.WithDetail("无效的远程地址")
	}
	if err := s.checkRemoteHost(ctx, u.Hostname()); err != nil {
		return nil, err
	}
	limits := s.newLimits(ctx, user)

	rctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	hreq, err := http.NewRequestWithContext(rctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errs.ErrInvalidParam.WithDetail("无效的远程地址")
	}
	hreq.Header.Set("User-Agent", "PicHub/1.0")
	resp, err := s.client.Do(hreq)
	if err != nil {
		return nil, errs.ErrInvalidParam.WithDetail("远程文件下载失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errs.ErrInvalidParam.WithDetail("远程文件下载失败: " + resp.Status)
	}
	if limits.maxFileSize > 0 && resp.ContentLength > limits.maxFileSize {
		return nil, errs.ErrFileTooLarge.WithDetail(fmt.Sprintf("单文件不能超过 %d MB", limits.maxFileSize>>20))
	}
	tmp, size, hash, err := s.saveTemp(resp.Body, limits.maxFileSize)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp)

	name := path.Base(u.Path)
	if name == "" || name == "." || name == "/" {
		name = "remote"
	}
	if filepath.Ext(name) == "" {
		if ct := resp.Header.Get("Content-Type"); ct != "" {
			if exts, eerr := mime.ExtensionsByType(ct); eerr == nil && len(exts) > 0 {
				name += exts[0]
			}
		}
	}
	opts := UploadOptions{
		Filename:  name,
		AlbumID:   req.AlbumID,
		Tags:      req.Tags,
		IsPublic:  req.IsPublic,
		StorageID: req.StorageID,
		IP:        ip,
		UserAgent: ua,
	}
	return s.persist(ctx, user, tmp, size, hash, opts)
}

// ChunkInit 初始化分片上传会话。
func (s *UploadService) ChunkInit(ctx context.Context, user *model.User, req *dto.ChunkInitReq) (*dto.ChunkInitResp, error) {
	if err := s.checkUploadPermission(ctx, user); err != nil {
		return nil, err
	}
	limits := s.newLimits(ctx, user)
	if req.Size <= 0 {
		return nil, errs.ErrInvalidParam.WithDetail("文件大小无效")
	}
	if limits.maxFileSize > 0 && req.Size > limits.maxFileSize {
		return nil, errs.ErrFileTooLarge.WithDetail(fmt.Sprintf("单文件不能超过 %d MB", limits.maxFileSize>>20))
	}
	maxChunks := s.cfg.Upload.MaxChunkCount
	if maxChunks <= 0 {
		maxChunks = 1000
	}
	if req.Chunks <= 0 || req.Chunks > maxChunks {
		return nil, errs.ErrInvalidParam.WithDetail("分片数量无效")
	}
	hash := strings.ToLower(strings.TrimSpace(req.Hash))
	if !validSHA256(hash) {
		return nil, errs.ErrInvalidParam.WithDetail("文件哈希格式无效")
	}
	id, err := auth.NewOpaqueToken(16)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	dir := filepath.Join(s.chunkRoot, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	sess := &chunkSession{
		ID:        id,
		UserID:    user.ID,
		Name:      req.Name,
		Size:      req.Size,
		Hash:      hash,
		Count:     req.Chunks,
		AlbumID:   req.AlbumID,
		Tags:      req.Tags,
		IsPublic:  req.IsPublic,
		StorageID: req.StorageID,
		Dir:       dir,
		CreatedAt: model.Now(),
		received:  map[int]int64{},
	}
	s.chunkMu.Lock()
	s.chunks[id] = sess
	s.chunkMu.Unlock()
	s.cleanupChunksLocked(model.Now().Add(-chunkSessionTTL))
	return &dto.ChunkInitResp{
		UploadID:  id,
		Uploaded:  []int{},
		ChunkSize: s.chunkSize(),
	}, nil
}

// UploadPart 保存一个分片。
func (s *UploadService) UploadPart(ctx context.Context, user *model.User, uploadID string, index int, r io.Reader, size int64) error {
	sess, err := s.session(user.ID, uploadID)
	if err != nil {
		return err
	}
	if index < 0 || index >= sess.Count {
		return errs.ErrInvalidParam.WithDetail("分片序号无效")
	}
	chunkSize := s.chunkSize()
	limit := chunkSize
	if index == sess.Count-1 && sess.Size > 0 {
		if rest := sess.Size - int64(index)*chunkSize; rest > 0 && rest < limit {
			limit = rest
		}
	}
	if limit <= 0 {
		limit = chunkSize
	}
	if size > 0 && size > limit {
		return errs.ErrInvalidParam.WithDetail("分片大小超出限制")
	}
	partPath := filepath.Join(sess.Dir, fmt.Sprintf("%06d.part", index))
	f, err := os.Create(partPath)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	n, cerr := io.Copy(f, io.LimitReader(r, limit+1))
	if cerr != nil {
		f.Close()
		os.Remove(partPath)
		return errs.ErrInternal.Wrap(cerr)
	}
	if err := f.Close(); err != nil {
		os.Remove(partPath)
		return errs.ErrInternal.Wrap(err)
	}
	if n > limit {
		os.Remove(partPath)
		return errs.ErrInvalidParam.WithDetail("分片大小超出限制")
	}
	if n == 0 {
		os.Remove(partPath)
		return errs.ErrInvalidParam.WithDetail("分片内容为空")
	}
	sess.mu.Lock()
	sess.received[index] = n
	sess.mu.Unlock()
	return nil
}

// ChunkComplete 合并分片并保存图片。
func (s *UploadService) ChunkComplete(ctx context.Context, user *model.User, uploadID, ip, ua string) (*dto.UploadResp, error) {
	sess, err := s.session(user.ID, uploadID)
	if err != nil {
		return nil, err
	}
	defer s.dropSession(uploadID)

	sess.mu.Lock()
	for i := 0; i < sess.Count; i++ {
		if _, ok := sess.received[i]; !ok {
			sess.mu.Unlock()
			return nil, errs.ErrInvalidParam.WithDetail(fmt.Sprintf("分片 %d 尚未上传", i))
		}
	}
	sess.mu.Unlock()

	tmp, err := os.CreateTemp("", "pichub-merge-*.tmp")
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	h := sha256.New()
	var total int64
	for i := 0; i < sess.Count; i++ {
		partPath := filepath.Join(sess.Dir, fmt.Sprintf("%06d.part", i))
		pf, oerr := os.Open(partPath)
		if oerr != nil {
			tmp.Close()
			return nil, errs.ErrInternal.Wrap(oerr)
		}
		n, cerr := io.Copy(io.MultiWriter(tmp, h), pf)
		pf.Close()
		if cerr != nil {
			tmp.Close()
			return nil, errs.ErrInternal.Wrap(cerr)
		}
		total += n
	}
	if err := tmp.Close(); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if sess.Size > 0 && total != sess.Size {
		return nil, errs.ErrInvalidParam.WithDetail("分片合并后大小与声明不一致")
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if sess.Hash != "" && !strings.EqualFold(sum, sess.Hash) {
		return nil, errs.ErrInvalidParam.WithDetail("文件哈希校验失败")
	}
	opts := UploadOptions{
		Filename:  sess.Name,
		AlbumID:   sess.AlbumID,
		Tags:      sess.Tags,
		IsPublic:  sess.IsPublic,
		StorageID: sess.StorageID,
		IP:        ip,
		UserAgent: ua,
	}
	return s.persist(ctx, user, tmpName, total, sum, opts)
}

// CleanupChunks 清理超时的分片会话，返回清理数量。
func (s *UploadService) CleanupChunks(before time.Time) int {
	s.chunkMu.Lock()
	defer s.chunkMu.Unlock()
	return s.cleanupChunksLocked(before)
}

func (s *UploadService) cleanupChunksLocked(before time.Time) int {
	n := 0
	for id, sess := range s.chunks {
		if sess.CreatedAt.After(before) {
			continue
		}
		delete(s.chunks, id)
		_ = os.RemoveAll(sess.Dir)
		n++
	}
	return n
}

func (s *UploadService) session(userID uint64, id string) (*chunkSession, error) {
	s.chunkMu.Lock()
	sess := s.chunks[id]
	s.chunkMu.Unlock()
	if sess == nil || sess.UserID != userID {
		return nil, errs.ErrNotFound.WithMessage("分片上传会话不存在")
	}
	if sess.CreatedAt.Before(model.Now().Add(-chunkSessionTTL)) {
		s.dropSession(id)
		return nil, errs.ErrNotFound.WithMessage("分片上传会话已过期")
	}
	return sess, nil
}

func (s *UploadService) dropSession(id string) {
	s.chunkMu.Lock()
	sess := s.chunks[id]
	delete(s.chunks, id)
	s.chunkMu.Unlock()
	if sess != nil {
		_ = os.RemoveAll(sess.Dir)
	}
}

func (s *UploadService) chunkSize() int64 {
	if s.cfg.Upload.ChunkSize > 0 {
		return s.cfg.Upload.ChunkSize
	}
	return 5 << 20
}

// persist 是上传的统一落库流程。
func (s *UploadService) persist(ctx context.Context, user *model.User, tmpPath string, size int64, hash string, opts UploadOptions) (*dto.UploadResp, error) {
	if err := s.checkUploadPermission(ctx, user); err != nil {
		return nil, err
	}
	limits := s.newLimits(ctx, user)
	if size <= 0 {
		fi, err := os.Stat(tmpPath)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		size = fi.Size()
	}
	if limits.maxFileSize > 0 && size > limits.maxFileSize {
		return nil, errs.ErrFileTooLarge.WithDetail(fmt.Sprintf("单文件不能超过 %d MB", limits.maxFileSize>>20))
	}
	format, ext, width, height, err := s.inspectFile(tmpPath, opts.Filename, limits)
	if err != nil {
		return nil, err
	}
	mimeType := imageproc.MimeOf(format)
	if format == "ico" {
		mimeType = "image/x-icon"
	}

	// 用户级秒传：命中已有记录时直接返回。
	if existing, err := s.repos.Image.GetByUserHash(ctx, user.ID, hash); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	} else if existing != nil {
		if opts.ObjectKey == "" || existing.StorageKey == opts.ObjectKey {
			return s.resp(existing, true), nil
		}
		// S3 网关下同内容更换对象 Key：把既有记录迁移到新 Key。
		if merr := s.moveObjectKey(ctx, user, existing, opts.ObjectKey, tmpPath, size, mimeType); merr != nil {
			return nil, merr
		}
		return s.resp(existing, true), nil
	}

	today := model.Today()
	if limits.dailyLimit > 0 {
		counter, err := s.repos.Counter.Get(ctx, user.ID, today)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if counter != nil && counter.Count >= limits.dailyLimit {
			return nil, errs.ErrDailyLimit.WithMessage(fmt.Sprintf("今日上传已达上限（%d 张）", limits.dailyLimit))
		}
	}

	// 全局秒传：复用已存在的公开物理文件，仅新建记录。
	var reused *model.Image
	if limits.dedupEnabled && limits.dedupScope == "global" {
		other, err := s.repos.Image.GetByHash(ctx, hash)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		// S3 网关要求对象 Key 与客户端一致，不能复用其他 Key 的物理文件。
		if other != nil && opts.ObjectKey != "" && other.StorageKey != opts.ObjectKey {
			other = nil
		}
		reused = other
	}

	isPublic := limits.defaultPublic
	if opts.IsPublic != nil {
		isPublic = *opts.IsPublic
	}
	var album *model.Album
	if opts.AlbumID != nil && *opts.AlbumID > 0 {
		a, err := s.repos.Album.GetByID(ctx, *opts.AlbumID)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if a == nil || a.UserID != user.ID {
			return nil, errs.ErrInvalidParam.WithDetail("相册不存在")
		}
		album = a
	}

	ok, err := s.repos.User.AddUsedBytes(ctx, user.ID, size, user.QuotaBytes)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if !ok {
		return nil, errs.ErrQuotaExceeded
	}
	release := func() {
		if err := s.repos.User.SubUsedBytes(context.Background(), user.ID, size); err != nil && s.log != nil {
			s.log.Warn("rollback used bytes failed", logger.Err(err), logger.Uint64("user_id", user.ID))
		}
	}

	name := strings.TrimSpace(filepath.Base(opts.Filename))
	if name == "" || name == "." {
		name = hash + "." + ext
	}
	display := strings.TrimSuffix(name, filepath.Ext(name))
	if display == "" {
		display = hash
	}

	img := &model.Image{
		UserID:      user.ID,
		Hash:        hash,
		OriginName:  trimTo(name, 255),
		DisplayName: trimTo(display, 255),
		Ext:         ext,
		Mime:        mimeType,
		Size:        size,
		Width:       width,
		Height:      height,
		IsPublic:    isPublic,
		Status:      model.ImageStatusNormal,
	}
	if album != nil {
		img.AlbumID = &album.ID
	}

	var (
		st storage.Storage
		sc *model.StorageConfig
	)
	if reused != nil {
		img.StorageKey = reused.StorageKey
		img.StorageDriver = reused.StorageDriver
		img.URL = reused.URL
		if img.Ext == "" {
			img.Ext = reused.Ext
		}
		if err := s.repos.Image.IncrRefCount(ctx, reused.ID, 1); err != nil {
			release()
			return nil, errs.ErrInternal.Wrap(err)
		}
	} else {
		st, sc, err = s.resolveStorage(ctx, user, opts.StorageID)
		if err != nil {
			release()
			return nil, err
		}
		key := opts.ObjectKey
		if key == "" {
			key = s.buildKey(user, sc, hash, ext)
		}
		f, oerr := os.Open(tmpPath)
		if oerr != nil {
			release()
			return nil, errs.ErrInternal.Wrap(oerr)
		}
		perr := st.Put(ctx, key, f, size, mimeType)
		f.Close()
		if perr != nil {
			release()
			return nil, errs.ErrInternal.Wrap(perr)
		}
		img.StorageKey = key
		img.StorageDriver = st.Driver()
		img.URL = s.assetURL(st, key, hash, ext)
	}

	err = s.repos.Tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repos.Image.Create(ctx, img); err != nil {
			return err
		}
		if img.AlbumID != nil {
			if err := s.repos.Album.IncrImageCount(ctx, *img.AlbumID, 1); err != nil {
				return err
			}
		}
		return s.repos.Counter.Incr(ctx, user.ID, today, 1, size)
	})
	if err != nil {
		release()
		if reused != nil {
			_ = s.repos.Image.IncrRefCount(context.Background(), reused.ID, -1)
		} else if st != nil {
			_ = st.Delete(context.Background(), img.StorageKey)
		}
		return nil, errs.ErrInternal.Wrap(err)
	}
	if s.images != nil && strings.TrimSpace(opts.Tags) != "" {
		if _, terr := s.images.tags.SyncImageTags(ctx, user.ID, img.ID, splitTags(opts.Tags)); terr != nil && s.log != nil {
			s.log.Warn("sync image tags failed", logger.Err(terr), logger.Uint64("image_id", img.ID))
		}
	}
	return s.resp(img, reused != nil), nil
}

// moveObjectKey 把既有图片记录迁移到新的存储 Key（S3 网关重复内容场景）。
func (s *UploadService) moveObjectKey(ctx context.Context, user *model.User, img *model.Image, newKey, tmpPath string, size int64, mimeType string) error {
	st, _, err := s.resolveStorage(ctx, user, 0)
	if err != nil {
		return err
	}
	f, oerr := os.Open(tmpPath)
	if oerr != nil {
		return errs.ErrInternal.Wrap(oerr)
	}
	defer f.Close()
	if perr := st.Put(ctx, newKey, f, size, mimeType); perr != nil {
		return errs.ErrInternal.Wrap(perr)
	}
	url := s.assetURL(st, newKey, img.Hash, img.Ext)
	if uerr := s.repos.Image.UpdateFields(ctx, img.ID, map[string]any{
		"storage_key":    newKey,
		"storage_driver": st.Driver(),
		"url":            url,
	}); uerr != nil {
		_ = st.Delete(ctx, newKey)
		return errs.ErrInternal.Wrap(uerr)
	}
	img.StorageKey = newKey
	img.StorageDriver = st.Driver()
	img.URL = url
	return nil
}

// normalizeObjectKey 规范化 S3 对象 Key，拒绝路径穿越。
func normalizeObjectKey(raw string) string {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if raw == "" {
		return ""
	}
	key := strings.TrimLeft(path.Clean("/"+raw), "/")
	if key == "" || key == "." {
		return ""
	}
	if len(key) > 480 {
		return ""
	}
	return key
}

// resolveStorage 解析上传使用的存储策略并校验归属。
func (s *UploadService) resolveStorage(ctx context.Context, user *model.User, requestedID uint64) (storage.Storage, *model.StorageConfig, error) {
	if s.storages == nil {
		return nil, nil, errs.ErrInternal.WithMessage("存储服务未初始化")
	}
	if requestedID > 0 {
		sc, err := s.repos.Storage.GetByID(ctx, requestedID)
		if err != nil {
			return nil, nil, errs.ErrInternal.Wrap(err)
		}
		if sc == nil || sc.Status != model.StatusEnabled {
			return nil, nil, errs.ErrInvalidParam.WithDetail("存储策略不可用")
		}
		if sc.UserID != nil {
			if *sc.UserID != user.ID {
				return nil, nil, errs.ErrForbidden.WithMessage("无权使用该存储策略")
			}
			if !s.storages.AllowUserStrategy() {
				return nil, nil, errs.ErrForbidden.WithMessage("自定义存储未开放")
			}
		}
		st, err := s.storages.Build(ctx, sc)
		if err != nil {
			return nil, nil, errs.ErrInternal.Wrap(err)
		}
		return st, sc, nil
	}
	st, sc, err := s.storages.Resolve(ctx, user, 0)
	if err != nil {
		return nil, nil, errs.ErrInternal.Wrap(err)
	}
	return st, sc, nil
}

// inspectFile 校验图片内容并返回格式、扩展名与尺寸。
func (s *UploadService) inspectFile(tmpPath, filename string, limits *uploadLimits) (format, ext string, width, height int, err error) {
	ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	if ext == "jpeg" {
		ext = "jpg"
	}
	if ext != "" && !limits.allowsExt(ext) {
		return "", "", 0, 0, errs.ErrInvalidParam.WithDetail("不支持的文件类型: " + ext)
	}
	f, err := os.Open(tmpPath)
	if err != nil {
		return "", "", 0, 0, errs.ErrInternal.Wrap(err)
	}
	defer f.Close()
	head := make([]byte, 12)
	n, _ := io.ReadFull(f, head)
	format = imageproc.DetectFormat(head[:n])
	if format == "" {
		if ext == "ico" {
			return "ico", "ico", 0, 0, nil
		}
		return "", "", 0, 0, errs.ErrInvalidParam.WithDetail("无法识别的图片格式")
	}
	if format == imageproc.FormatSVG {
		if !limits.allowSVG {
			return "", "", 0, 0, errs.ErrInvalidParam.WithDetail("SVG 上传已禁用")
		}
		return format, "svg", 0, 0, nil
	}
	if !imageproc.CanDecode(format) {
		if ext == "" {
			ext = imageproc.ExtOf(format)
		}
		return format, ext, 0, 0, nil
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", "", 0, 0, errs.ErrInternal.Wrap(err)
	}
	info, perr := imageproc.Probe(f)
	if perr != nil {
		return "", "", 0, 0, errs.ErrInvalidParam.WithDetail("图片解析失败")
	}
	format = info.Format
	ext = imageproc.ExtOf(format)
	if !limits.allowsExt(ext) {
		return "", "", 0, 0, errs.ErrInvalidParam.WithDetail("不支持的文件类型: " + ext)
	}
	return format, ext, info.Width, info.Height, nil
}

// saveTemp 把数据流写入临时文件并计算 SHA-256。
func (s *UploadService) saveTemp(r io.Reader, limit int64) (string, int64, string, error) {
	f, err := os.CreateTemp("", "pichub-up-*.tmp")
	if err != nil {
		return "", 0, "", errs.ErrInternal.Wrap(err)
	}
	name := f.Name()
	h := sha256.New()
	var reader io.Reader = r
	if limit > 0 {
		reader = io.LimitReader(r, limit+1)
	}
	n, cerr := io.Copy(io.MultiWriter(f, h), reader)
	if cerr != nil {
		f.Close()
		os.Remove(name)
		return "", 0, "", errs.ErrInternal.Wrap(cerr)
	}
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", 0, "", errs.ErrInternal.Wrap(err)
	}
	if limit > 0 && n > limit {
		os.Remove(name)
		return "", 0, "", errs.ErrFileTooLarge.WithDetail(fmt.Sprintf("单文件不能超过 %d MB", limit>>20))
	}
	if n == 0 {
		os.Remove(name)
		return "", 0, "", errs.ErrInvalidParam.WithDetail("上传内容为空")
	}
	return name, n, hex.EncodeToString(h.Sum(nil)), nil
}

func (s *UploadService) checkUploadPermission(ctx context.Context, user *model.User) error {
	if user == nil {
		return errs.ErrUnauthorized
	}
	switch user.Status {
	case model.UserStatusNormal:
	case model.UserStatusDisabled:
		return errs.ErrForbidden.WithMessage("账号已被禁用")
	case model.UserStatusPending:
		return errs.ErrEmailRequired.WithMessage("请先完成邮箱验证")
	}
	if !user.EmailVerified {
		if strings.EqualFold(s.settings.Get(ctx, KeyRegisterRequireEmailVerify), "soft") {
			return errs.ErrEmailRequired.WithMessage("请先完成邮箱验证后再上传")
		}
	}
	return nil
}

func (s *UploadService) newLimits(ctx context.Context, user *model.User) *uploadLimits {
	l := &uploadLimits{
		maxFileSize:    s.cfg.Upload.MaxFileSize,
		allowSVG:       s.cfg.Upload.AllowSVG,
		dedupEnabled:   s.cfg.Upload.DedupEnabled,
		dedupScope:     strings.ToLower(strings.TrimSpace(s.cfg.Upload.DedupScope)),
		keepOriginName: s.cfg.Upload.KeepOriginName,
	}
	if l.maxFileSize <= 0 {
		l.maxFileSize = defaultMaxFileSize
	}
	if l.dedupScope != "global" {
		l.dedupScope = "user"
	}
	if user.MaxFileSize > 0 {
		l.maxFileSize = user.MaxFileSize
	}
	l.dailyLimit = user.DailyLimit
	if l.dailyLimit <= 0 && user.GroupID > 0 && s.repos.Group != nil {
		if g, err := s.repos.Group.GetByID(ctx, user.GroupID); err == nil && g != nil {
			l.dailyLimit = g.DailyLimit
		}
	}
	exts := s.cfg.Upload.AllowedExtensions
	if raw := strings.TrimSpace(s.settings.Get(ctx, KeyUploadAllowedExtensions)); raw != "" {
		var parsed []string
		if strings.HasPrefix(raw, "[") {
			_ = json.Unmarshal([]byte(raw), &parsed)
		}
		if len(parsed) == 0 {
			parsed = strings.Split(raw, ",")
		}
		if len(parsed) > 0 {
			exts = parsed
		}
	}
	if len(exts) == 0 {
		exts = []string{"jpg", "jpeg", "png", "gif", "webp", "bmp", "svg", "avif", "ico"}
	}
	l.exts = make(map[string]bool, len(exts))
	for _, e := range exts {
		e = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(e), "."))
		if e != "" {
			l.exts[e] = true
		}
	}
	vis := user.DefaultVisibility
	if vis == "" {
		vis = s.settings.GetString(ctx, KeyUploadDefaultVisibility, s.cfg.Upload.DefaultVisibility)
	}
	if vis == "" {
		vis = "public"
	}
	l.defaultPublic = !strings.EqualFold(vis, "private")
	return l
}

// buildKey 按策略模板生成对象 Key。
func (s *UploadService) buildKey(user *model.User, sc *model.StorageConfig, hash, ext string) string {
	now := model.Now()
	prefix := ""
	tpl := ""
	if sc != nil {
		prefix = strings.Trim(sc.PathPrefix, "/")
		var conf manager.StrategyConfig
		if len(sc.Config) > 0 {
			_ = json.Unmarshal(sc.Config, &conf)
		}
		tpl = strings.TrimSpace(conf.PathTemplate)
	}
	if prefix == "" {
		prefix = strings.Trim(s.cfg.Storage.Local.PathPrefix, "/")
	}
	if tpl == "" {
		tpl = s.cfg.Storage.Local.PathTemplate
	}
	if tpl == "" {
		tpl = "{prefix}/{yyyy}/{MM}/{dd}/{hash2}/{hash}.{ext}"
	}
	repl := strings.NewReplacer(
		"{prefix}", prefix,
		"{yyyy}", now.Format("2006"),
		"{MM}", now.Format("01"),
		"{dd}", now.Format("02"),
		"{hh}", now.Format("15"),
		"{mm}", now.Format("04"),
		"{ss}", now.Format("05"),
		"{hash2}", hash[:2],
		"{hash}", hash,
		"{ext}", ext,
		"{user_id}", strconv.FormatUint(user.ID, 10),
	)
	key := repl.Replace(tpl)
	key = strings.TrimLeft(filepath.ToSlash(key), "/")
	for strings.Contains(key, "//") {
		key = strings.ReplaceAll(key, "//", "/")
	}
	return key
}

// assetURL 计算图片的访问地址。
func (s *UploadService) assetURL(st storage.Storage, key, hash, ext string) string {
	if st != nil && st.Driver() != storage.DriverLocal {
		if u := st.PublicURL(key); u != "" {
			return u
		}
	}
	return s.cfg.SiteURL() + "/i/" + hash + "." + ext
}

// resp 构造上传响应。
func (s *UploadService) resp(img *model.Image, dedup bool) *dto.UploadResp {
	u := imageURL(s.cfg, img)
	name := img.DisplayName
	if name == "" {
		name = img.Hash
	}
	alt := strings.ReplaceAll(name, "]", "")
	return &dto.UploadResp{
		ID:       img.ID,
		URL:      u,
		ThumbURL: thumbURL(s.cfg, img.Hash, img.Ext),
		Markdown: "![" + alt + "](" + u + ")",
		HTML:     fmt.Sprintf(`<img src="%s" alt="%s" />`, html.EscapeString(u), html.EscapeString(alt)),
		BBCode:   "[img]" + u + "[/img]",
		Size:     img.Size,
		Width:    img.Width,
		Height:   img.Height,
		Hash:     img.Hash,
		Dedup:    dedup,
	}
}

func (s *UploadService) checkRemoteHost(ctx context.Context, host string) error {
	if host == "" {
		return errs.ErrInvalidParam.WithDetail("无效的远程地址")
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return errs.ErrInvalidParam.WithDetail("无法解析远程地址")
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
			ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
			return errs.ErrForbidden.WithMessage("不允许访问内网地址")
		}
	}
	return nil
}

func splitTags(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '\t' || r == '\n'
	})
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func validSHA256(h string) bool {
	if len(h) != 64 {
		return false
	}
	_, err := hex.DecodeString(h)
	return err == nil
}
