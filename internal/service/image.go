package service

import (
	"context"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pichub/pichub/internal/auth"
	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/imageproc"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
	"github.com/pichub/pichub/internal/storage"
)

// Delivery 是图片直出所需的数据。
type Delivery struct {
	Image       *model.Image
	LocalPath   string
	Reader      io.ReadCloser
	Info        *storage.FileInfo
	ContentType string
}

// ImageService 处理图片管理、直出与引用查询。
type ImageService struct {
	cfg       *config.Config
	log       *logger.Logger
	repos     Repos
	settings  *SettingsService
	albums    *AlbumService
	tags      *TagService
	referers  *RefererService
	storages  StorageManager
	processor *imageproc.Processor
}

// NewImageService 创建图片服务。
func NewImageService(d Deps, settings *SettingsService, albums *AlbumService, tags *TagService, referers *RefererService) *ImageService {
	return &ImageService{
		cfg: d.Config, log: d.Log, repos: d.Repos, settings: settings,
		albums: albums, tags: tags, referers: referers,
		storages: d.Storages, processor: d.Processor,
	}
}

// List 查询用户图片列表。
func (s *ImageService) List(ctx context.Context, userID uint64, req dto.ImageListReq, withTags bool) ([]*dto.ImageResp, int64, error) {
	q := model.ImageQuery{
		UserID:    userID,
		Keyword:   strings.TrimSpace(req.Keyword),
		AlbumID:   req.AlbumID,
		TagID:     req.TagID,
		Ext:       strings.TrimPrefix(strings.ToLower(strings.TrimSpace(req.Ext)), "."),
		Referred:  req.Referred,
		IsPublic:  req.IsPublic,
		Cursor:    req.Cursor,
		Page:      req.Page,
		Size:      req.Size,
		TrashOnly: req.Trash,
	}
	list, total, err := s.repos.Image.List(ctx, q)
	if err != nil {
		return nil, 0, errs.ErrInternal.Wrap(err)
	}
	out := make([]*dto.ImageResp, 0, len(list))
	for _, img := range list {
		resp := s.toResp(img)
		if withTags {
			tags, err := s.tags.ListByImage(ctx, img.ID)
			if err != nil {
				return nil, 0, err
			}
			resp.Tags = tags
		}
		out = append(out, resp)
	}
	return out, total, nil
}

// Get 返回图片详情。
func (s *ImageService) Get(ctx context.Context, userID, id uint64) (*dto.ImageResp, error) {
	img, err := s.own(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	resp := s.toResp(img)
	tags, err := s.tags.ListByImage(ctx, img.ID)
	if err != nil {
		return nil, err
	}
	resp.Tags = tags
	return resp, nil
}

// Update 修改图片信息。
func (s *ImageService) Update(ctx context.Context, userID, id uint64, req *dto.ImageUpdateReq) (*dto.ImageResp, error) {
	img, err := s.own(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{}
	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if name == "" || len([]rune(name)) > 255 {
			return nil, errs.ErrInvalidParam.WithDetail("名称长度需为 1-255 个字符")
		}
		fields["display_name"] = name
	}
	if req.IsPublic != nil {
		fields["is_public"] = *req.IsPublic
	}
	var newAlbumID *uint64
	if req.ClearAlbum {
		fields["album_id"] = nil
	} else if req.AlbumID != nil {
		if *req.AlbumID > 0 {
			album, err := s.repos.Album.GetByID(ctx, *req.AlbumID)
			if err != nil {
				return nil, errs.ErrInternal.Wrap(err)
			}
			if album == nil || album.UserID != userID {
				return nil, errs.ErrInvalidParam.WithDetail("相册不存在")
			}
			fields["album_id"] = album.ID
			newAlbumID = &album.ID
		} else {
			fields["album_id"] = nil
		}
	}
	err = s.repos.Tx.Do(ctx, func(ctx context.Context) error {
		if len(fields) > 0 {
			if err := s.repos.Image.UpdateFields(ctx, img.ID, fields); err != nil {
				return errs.ErrInternal.Wrap(err)
			}
		}
		if _, ok := fields["album_id"]; ok {
			oldID := img.AlbumID
			if oldID != nil && (newAlbumID == nil || *newAlbumID != *oldID) {
				if err := s.repos.Album.IncrImageCount(ctx, *oldID, -1); err != nil {
					return errs.ErrInternal.Wrap(err)
				}
			}
			if newAlbumID != nil && (oldID == nil || *newAlbumID != *oldID) {
				if err := s.repos.Album.IncrImageCount(ctx, *newAlbumID, 1); err != nil {
					return errs.ErrInternal.Wrap(err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if req.Tags != nil {
		if _, err := s.tags.SyncImageTags(ctx, userID, img.ID, *req.Tags); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, userID, id)
}

// Delete 把图片移入回收站。
func (s *ImageService) Delete(ctx context.Context, userID, id uint64) error {
	img, err := s.own(ctx, userID, id)
	if err != nil {
		return err
	}
	if img.Status == model.ImageStatusTrash {
		return nil
	}
	now := model.Now()
	err = s.repos.Tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repos.Image.SoftDelete(ctx, img.ID, now.UnixNano(), now); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		if img.AlbumID != nil {
			if err := s.repos.Album.IncrImageCount(ctx, *img.AlbumID, -1); err != nil {
				return errs.ErrInternal.Wrap(err)
			}
		}
		return nil
	})
	return err
}

// Batch 执行批量操作，返回影响数量。
func (s *ImageService) Batch(ctx context.Context, userID uint64, req *dto.ImageBatchReq) (int, error) {
	if len(req.IDs) == 0 {
		return 0, errs.ErrInvalidParam.WithDetail("请选择图片")
	}
	if len(req.IDs) > 500 {
		return 0, errs.ErrInvalidParam.WithDetail("单次最多操作 500 张图片")
	}
	switch req.Action {
	case dto.BatchDelete, dto.BatchRestore, dto.BatchMove, dto.BatchSetPublic, dto.BatchSetPrivate:
	default:
		return 0, errs.ErrInvalidParam.WithDetail("不支持的操作类型")
	}
	var targetAlbum *model.Album
	if req.Action == dto.BatchMove {
		if req.AlbumID == nil || *req.AlbumID == 0 {
			return 0, errs.ErrInvalidParam.WithDetail("请选择目标相册")
		}
		album, err := s.repos.Album.GetByID(ctx, *req.AlbumID)
		if err != nil {
			return 0, errs.ErrInternal.Wrap(err)
		}
		if album == nil || album.UserID != userID {
			return 0, errs.ErrInvalidParam.WithDetail("目标相册不存在")
		}
		targetAlbum = album
	}
	affected := 0
	for _, id := range req.IDs {
		img, err := s.repos.Image.GetByID(ctx, id)
		if err != nil {
			return affected, errs.ErrInternal.Wrap(err)
		}
		if img == nil || img.UserID != userID {
			continue
		}
		switch req.Action {
		case dto.BatchDelete:
			if img.Status == model.ImageStatusNormal {
				if err := s.Delete(ctx, userID, id); err != nil {
					return affected, err
				}
				affected++
			}
		case dto.BatchRestore:
			if img.Status == model.ImageStatusTrash {
				if err := s.Restore(ctx, userID, id); err != nil {
					return affected, err
				}
				affected++
			}
		case dto.BatchMove:
			if img.Status != model.ImageStatusNormal {
				continue
			}
			if err := s.moveImage(ctx, userID, img, targetAlbum); err != nil {
				return affected, err
			}
			affected++
		case dto.BatchSetPublic, dto.BatchSetPrivate:
			fields := map[string]any{"is_public": req.Action == dto.BatchSetPublic}
			if err := s.repos.Image.UpdateFields(ctx, img.ID, fields); err != nil {
				return affected, errs.ErrInternal.Wrap(err)
			}
			affected++
		}
	}
	return affected, nil
}

func (s *ImageService) moveImage(ctx context.Context, userID uint64, img *model.Image, album *model.Album) error {
	return s.repos.Tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repos.Image.UpdateFields(ctx, img.ID, map[string]any{"album_id": album.ID}); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		if img.AlbumID != nil {
			if *img.AlbumID == album.ID {
				return nil
			}
			if err := s.repos.Album.IncrImageCount(ctx, *img.AlbumID, -1); err != nil {
				return errs.ErrInternal.Wrap(err)
			}
		}
		if err := s.repos.Album.IncrImageCount(ctx, album.ID, 1); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		return nil
	})
}

// Trash 返回回收站图片列表。
func (s *ImageService) Trash(ctx context.Context, userID uint64, req dto.ImageListReq) ([]*dto.ImageResp, int64, error) {
	req.Trash = true
	return s.List(ctx, userID, req, false)
}

// Restore 从回收站还原图片。
func (s *ImageService) Restore(ctx context.Context, userID, id uint64) error {
	img, err := s.repos.Image.GetByID(ctx, id)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if img == nil || img.UserID != userID {
		return errs.ErrNotFound.WithMessage("图片不存在")
	}
	if img.Status != model.ImageStatusTrash {
		return nil
	}
	return s.repos.Tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repos.Image.Restore(ctx, img.ID); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		if img.AlbumID != nil {
			if err := s.repos.Album.IncrImageCount(ctx, *img.AlbumID, 1); err != nil {
				return errs.ErrInternal.Wrap(err)
			}
		}
		return nil
	})
}

// ClearTrash 清空用户回收站。
func (s *ImageService) ClearTrash(ctx context.Context, userID uint64) (int, error) {
	cleared := 0
	for {
		req := dto.ImageListReq{Trash: true, Page: 1, Size: 100}
		list, total, err := s.List(ctx, userID, req, false)
		if err != nil {
			return cleared, err
		}
		if len(list) == 0 {
			return cleared, nil
		}
		for _, item := range list {
			if err := s.HardDelete(ctx, userID, item.ID); err != nil {
				return cleared, err
			}
			cleared++
		}
		if total <= int64(len(list)) {
			return cleared, nil
		}
	}
}

// CleanupTrash 清理超过保留期的回收站图片（全局定时任务）。
func (s *ImageService) CleanupTrash(ctx context.Context, before time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	list, err := s.repos.Image.ListTrashExpired(ctx, before, limit)
	if err != nil {
		return 0, errs.ErrInternal.Wrap(err)
	}
	n := 0
	for _, img := range list {
		if err := s.HardDelete(ctx, img.UserID, img.ID); err != nil {
			if s.log != nil {
				s.log.Warn("cleanup trash failed", logger.Err(err), logger.Uint64("image_id", img.ID))
			}
			continue
		}
		n++
	}
	return n, nil
}

// HardDelete 物理删除图片记录、引用、分享与存储对象。
func (s *ImageService) HardDelete(ctx context.Context, userID, id uint64) error {
	img, err := s.repos.Image.GetByID(ctx, id)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if img == nil || img.UserID != userID {
		return nil
	}
	if err := s.repos.Tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repos.Referer.DeleteByImage(ctx, img.ID); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		if err := s.repos.Share.DeleteByTarget(ctx, model.ShareTargetImage, img.ID); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		if err := s.repos.Tag.DetachAll(ctx, img.ID); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		if err := s.repos.Image.HardDelete(ctx, img.ID); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		if err := s.repos.User.SubUsedBytes(ctx, img.UserID, img.Size); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		return nil
	}); err != nil {
		return err
	}
	// 删除数据库记录后，若存储 Key 不再被引用，则删除存储对象。
	if others, err := s.repos.Image.GetByStorageKey(ctx, img.StorageKey); err == nil && others == nil {
		if err := s.deleteObject(ctx, img); err != nil && s.log != nil {
			s.log.Warn("delete storage object failed", logger.Err(err), logger.Uint64("image_id", img.ID))
		}
	}
	return nil
}

func (s *ImageService) deleteObject(ctx context.Context, img *model.Image) error {
	user, err := s.repos.User.GetByID(ctx, img.UserID)
	if err != nil {
		return err
	}
	st, _, err := s.storages.Resolve(ctx, user, 0)
	if err != nil {
		return err
	}
	return st.Delete(ctx, img.StorageKey)
}

// Referers 查询图片引用来源。
func (s *ImageService) Referers(ctx context.Context, userID, imageID uint64, page, size int, order string) ([]*dto.RefererResp, int64, error) {
	if _, err := s.own(ctx, userID, imageID); err != nil {
		return nil, 0, err
	}
	list, total, err := s.repos.Referer.ListByImage(ctx, model.RefererQuery{
		ImageID: imageID, Page: page, Size: size, OrderBy: order,
	})
	if err != nil {
		return nil, 0, errs.ErrInternal.Wrap(err)
	}
	out := make([]*dto.RefererResp, 0, len(list))
	for _, r := range list {
		out = append(out, dto.NewRefererResp(r))
	}
	return out, total, nil
}

// RefererSummary 返回单图引用汇总。
func (s *ImageService) RefererSummary(ctx context.Context, userID, imageID uint64) (*dto.RefererSummaryResp, error) {
	if _, err := s.own(ctx, userID, imageID); err != nil {
		return nil, err
	}
	total, hosts, err := s.repos.Referer.Summary(ctx, imageID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return &dto.RefererSummaryResp{TotalHits: total, HostCount: hosts}, nil
}

// DeleteReferer 删除单条引用记录。
func (s *ImageService) DeleteReferer(ctx context.Context, userID, imageID, refererID uint64) error {
	if _, err := s.own(ctx, userID, imageID); err != nil {
		return err
	}
	if err := s.repos.Referer.Delete(ctx, refererID, imageID); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	return nil
}

// DeliverOriginal 打开原图，校验可见性与签名。
func (s *ImageService) DeliverOriginal(ctx context.Context, hash string, ownerID uint64, exp int64, sign string) (*Delivery, error) {
	img, st, err := s.accessible(ctx, hash, ownerID, exp, sign)
	if err != nil {
		return nil, err
	}
	key := img.StorageKey
	if key == "" {
		key = img.Hash + "." + img.Ext
	}
	if provider, ok := st.(storage.LocalPathProvider); ok {
		if path, ok := provider.LocalPath(key); ok {
			if fi, serr := os.Stat(path); serr == nil {
				return &Delivery{
					Image: img, LocalPath: path, ContentType: img.Mime,
					Info: &storage.FileInfo{Size: fi.Size(), ModTime: fi.ModTime()},
				}, nil
			}
		}
	}
	rc, info, err := st.Get(ctx, key)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, errs.ErrNotFound.WithMessage("图片文件不存在")
		}
		return nil, errs.ErrInternal.Wrap(err)
	}
	return &Delivery{Image: img, Reader: rc, Info: info, ContentType: img.Mime}, nil
}

// S3Object 是 S3 网关读取对象的返回数据。
type S3Object struct {
	Image     *model.Image
	LocalPath string
	Reader    io.ReadCloser
	Info      *storage.FileInfo
}

// OpenS3Object 按存储 Key 打开用户对象供 S3 网关下载。
func (s *ImageService) OpenS3Object(ctx context.Context, userID uint64, key string) (*S3Object, error) {
	img, st, err := s.s3Own(ctx, userID, key)
	if err != nil {
		return nil, err
	}
	return s.openS3Object(ctx, img, st)
}

// OpenPublicS3Object 打开公开对象，供 S3 网关自定义域名匿名访问。
func (s *ImageService) OpenPublicS3Object(ctx context.Context, key string) (*S3Object, error) {
	key = strings.TrimLeft(strings.TrimSpace(key), "/")
	if key == "" {
		return nil, errs.ErrNotFound.WithMessage("对象不存在")
	}
	img, err := s.repos.Image.GetPublicByStorageKey(ctx, key)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if img == nil {
		return nil, errs.ErrNotFound.WithMessage("对象不存在")
	}
	st, _, err := s.storages.Resolve(ctx, nil, 0)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return s.openS3Object(ctx, img, st)
}

func (s *ImageService) openS3Object(ctx context.Context, img *model.Image, st storage.Storage) (*S3Object, error) {
	info := &storage.FileInfo{Size: img.Size, ContentType: img.Mime, ModTime: img.CreatedAt}
	if provider, ok := st.(storage.LocalPathProvider); ok {
		if path, ok := provider.LocalPath(img.StorageKey); ok {
			if fi, serr := os.Stat(path); serr == nil {
				return &S3Object{
					Image: img, LocalPath: path,
					Info: &storage.FileInfo{Size: fi.Size(), ContentType: img.Mime, ModTime: fi.ModTime()},
				}, nil
			}
		}
	}
	rc, finfo, gerr := st.Get(ctx, img.StorageKey)
	if gerr != nil {
		if gerr == storage.ErrNotFound {
			return nil, errs.ErrNotFound.WithMessage("对象不存在")
		}
		return nil, errs.ErrInternal.Wrap(gerr)
	}
	if finfo != nil {
		info = finfo
	}
	return &S3Object{Image: img, Reader: rc, Info: info}, nil
}

// DeleteS3Object 把存储 Key 对应图片移入回收站。
func (s *ImageService) DeleteS3Object(ctx context.Context, userID uint64, key string) error {
	img, _, err := s.s3Own(ctx, userID, key)
	if err != nil {
		return err
	}
	return s.Delete(ctx, userID, img.ID)
}

// ListS3Objects 按存储 Key 前缀游标列举用户对象。
func (s *ImageService) ListS3Objects(ctx context.Context, userID uint64, prefix, after string, limit int) ([]*model.Image, bool, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	list, err := s.repos.Image.ListStorageObjects(ctx, userID, prefix, after, limit+1)
	if err != nil {
		return nil, false, errs.ErrInternal.Wrap(err)
	}
	truncated := false
	if len(list) > limit {
		list = list[:limit]
		truncated = true
	}
	return list, truncated, nil
}

func (s *ImageService) s3Own(ctx context.Context, userID uint64, key string) (*model.Image, storage.Storage, error) {
	key = strings.TrimLeft(strings.TrimSpace(key), "/")
	if key == "" {
		return nil, nil, errs.ErrNotFound.WithMessage("对象不存在")
	}
	img, err := s.repos.Image.GetByUserStorageKey(ctx, userID, key)
	if err != nil {
		return nil, nil, errs.ErrInternal.Wrap(err)
	}
	if img == nil {
		return nil, nil, errs.ErrNotFound.WithMessage("对象不存在")
	}
	var user *model.User
	if u, uerr := s.repos.User.GetByID(ctx, userID); uerr == nil {
		user = u
	}
	st, _, err := s.storages.Resolve(ctx, user, 0)
	if err != nil {
		return nil, nil, errs.ErrInternal.Wrap(err)
	}
	return img, st, nil
}

// Thumbnail 生成缩略图。
func (s *ImageService) Thumbnail(ctx context.Context, hash, size string, ownerID uint64, exp int64, sign string) (*imageproc.Result, error) {
	if s.processor == nil || !s.processor.ThumbnailEnabled() {
		return nil, errs.ErrNotFound.WithMessage("缩略图未启用")
	}
	opt, ok := s.processor.ThumbOption(size)
	if !ok {
		return nil, errs.ErrInvalidParam.WithDetail("不支持的缩略图尺寸")
	}
	return s.render(ctx, hash, ownerID, exp, sign, opt)
}

// ParseDynamicOptions 解析动态处理查询参数。
func (s *ImageService) ParseDynamicOptions(q url.Values) (imageproc.Options, bool, error) {
	if s.processor == nil {
		return imageproc.Options{}, false, nil
	}
	return s.processor.ParseOptions(q)
}

// RenderDynamic 按查询参数动态处理图片。
func (s *ImageService) RenderDynamic(ctx context.Context, hash string, ownerID uint64, exp int64, sign string, opt imageproc.Options) (*imageproc.Result, error) {
	if s.processor == nil || !s.processor.DynamicEnabled() {
		return nil, errs.ErrNotFound.WithMessage("动态处理未启用")
	}
	return s.render(ctx, hash, ownerID, exp, sign, opt)
}

func (s *ImageService) render(ctx context.Context, hash string, ownerID uint64, exp int64, sign string, opt imageproc.Options) (*imageproc.Result, error) {
	img, st, err := s.accessible(ctx, hash, ownerID, exp, sign)
	if err != nil {
		return nil, err
	}
	key := img.StorageKey
	if key == "" {
		key = img.Hash + "." + img.Ext
	}
	sourceID := img.Hash + "|" + img.Ext
	res, err := s.processor.RenderCached(ctx, sourceID, func(ctx context.Context) (io.ReadCloser, error) {
		rc, _, gerr := st.Get(ctx, key)
		return rc, gerr
	}, opt)
	if err != nil {
		if err == imageproc.ErrUnsupportedFormat {
			return nil, errs.ErrInvalidParam.WithMessage("该图片格式不支持处理")
		}
		if err == imageproc.ErrImageTooLarge {
			return nil, errs.ErrFileTooLarge.WithMessage("图片尺寸过大")
		}
		if err == storage.ErrNotFound {
			return nil, errs.ErrNotFound.WithMessage("图片文件不存在")
		}
		return nil, errs.ErrInternal.Wrap(err)
	}
	return res, nil
}

func (s *ImageService) accessible(ctx context.Context, hash string, ownerID uint64, exp int64, sign string) (*model.Image, storage.Storage, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return nil, nil, errs.ErrNotFound
	}
	img, err := s.repos.Image.GetByHash(ctx, hash)
	if err != nil {
		return nil, nil, errs.ErrInternal.Wrap(err)
	}
	if img == nil {
		img, err = s.repos.Image.GetByHashAny(ctx, hash)
		if err != nil {
			return nil, nil, errs.ErrInternal.Wrap(err)
		}
	}
	if img == nil {
		return nil, nil, errs.ErrNotFound.WithMessage("图片不存在")
	}
	if !img.IsPublic && img.UserID != ownerID {
		secret := s.cfg.Security.URLSignSecret
		if secret == "" {
			secret = s.cfg.Security.JWTSecret
		}
		if !auth.VerifyURLSign(secret, img.Hash, exp, sign) {
			return nil, nil, errs.ErrForbidden.WithMessage("图片不可访问")
		}
	}
	var user *model.User
	if ownerID > 0 {
		if u, uerr := s.repos.User.GetByID(ctx, ownerID); uerr == nil {
			user = u
		}
	}
	st, _, err := s.storages.Resolve(ctx, user, 0)
	if err != nil {
		return nil, nil, errs.ErrInternal.Wrap(err)
	}
	return img, st, nil
}

// toResp 构造图片响应并补齐缩略图地址。
func (s *ImageService) toResp(img *model.Image) *dto.ImageResp {
	resp := dto.NewImageResp(img)
	resp.URL = imageURL(s.cfg, img)
	resp.ThumbURL = thumbURL(s.cfg, img.Hash, img.Ext)
	return resp
}

func (s *ImageService) own(ctx context.Context, userID, id uint64) (*model.Image, error) {
	img, err := s.repos.Image.GetByID(ctx, id)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if img == nil || img.UserID != userID {
		return nil, errs.ErrNotFound.WithMessage("图片不存在")
	}
	return img, nil
}

// imageURL 返回图片的访问地址。本地存储的地址由站点地址派生，
// 修改站点域名后历史图片也能跟随，避免库里存的旧绝对地址失效。
func imageURL(cfg *config.Config, img *model.Image) string {
	if img == nil {
		return ""
	}
	serveBySite := img.Hash != "" && img.Ext != ""
	if cfg != nil && serveBySite && (img.StorageDriver == storage.DriverLocal || img.URL == "") {
		return cfg.SiteURL() + "/i/" + img.Hash + "." + img.Ext
	}
	return img.URL
}

// thumbURL 构造缩略图地址。
func thumbURL(cfg *config.Config, hash, ext string) string {
	if cfg == nil || !cfg.Image.Thumbnail.Enabled || hash == "" {
		return ""
	}
	name := ""
	sizes := cfg.Image.Thumbnail.Sizes
	if len(sizes) > 0 {
		name = sizes[len(sizes)/2].Name
		if name == "" {
			name = strconv.Itoa(sizes[len(sizes)/2].Width)
		}
	}
	if name == "" {
		name = "800"
	}
	format := cfg.Image.Thumbnail.Format
	if format == "" {
		format = "webp"
	}
	return cfg.SiteURL() + "/t/" + hash + "/" + name + "." + format
}
