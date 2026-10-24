package service

import (
	"context"
	"strings"
	"time"

	"github.com/pichub/pichub/internal/auth"
	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
)

// ShareService 处理分享链接。
type ShareService struct {
	cfg      *config.Config
	log      *logger.Logger
	repos    Repos
	settings *SettingsService
}

// NewShareService 创建分享服务。
func NewShareService(d Deps, settings *SettingsService) *ShareService {
	return &ShareService{cfg: d.Config, log: d.Log, repos: d.Repos, settings: settings}
}

// List 返回用户的分享列表。
func (s *ShareService) List(ctx context.Context, userID uint64) ([]*dto.ShareResp, error) {
	list, err := s.repos.Share.ListByUser(ctx, userID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	out := make([]*dto.ShareResp, 0, len(list))
	for _, sh := range list {
		out = append(out, s.toResp(sh))
	}
	return out, nil
}

// Create 创建分享链接。
func (s *ShareService) Create(ctx context.Context, userID uint64, req *dto.ShareCreateReq) (*dto.ShareResp, error) {
	targetType := strings.ToLower(strings.TrimSpace(req.TargetType))
	switch targetType {
	case model.ShareTargetImage:
		img, err := s.repos.Image.GetByID(ctx, req.TargetID)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if img == nil || img.UserID != userID || img.Status != model.ImageStatusNormal {
			return nil, errs.ErrNotFound.WithMessage("图片不存在")
		}
	case model.ShareTargetAlbum:
		album, err := s.repos.Album.GetByID(ctx, req.TargetID)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if album == nil || album.UserID != userID {
			return nil, errs.ErrNotFound.WithMessage("相册不存在")
		}
	default:
		return nil, errs.ErrInvalidParam.WithDetail("分享类型只能是 image 或 album")
	}
	if err := s.checkShareQuota(ctx, userID); err != nil {
		return nil, err
	}
	sh := &model.Share{
		UserID:     userID,
		TargetType: targetType,
		TargetID:   req.TargetID,
		Status:     model.StatusEnabled,
		MaxViews:   req.MaxViews,
	}
	if sh.MaxViews < 0 {
		sh.MaxViews = 0
	}
	if req.ExpiresInHours > 0 {
		if req.ExpiresInHours > 24*365 {
			req.ExpiresInHours = 24 * 365
		}
		sh.ExpiresAt = ptrTime(model.Now().Add(timeHours(req.ExpiresInHours)))
	}
	if pwd := strings.TrimSpace(req.Password); pwd != "" {
		if len(pwd) < 4 || len(pwd) > 32 {
			return nil, errs.ErrInvalidParam.WithDetail("访问密码长度需为 4-32 位")
		}
		hash, err := auth.HashPassword(pwd, s.cfg.Security.BcryptCost)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		sh.PasswordHash = hash
	}
	slug, err := s.newSlug(ctx)
	if err != nil {
		return nil, err
	}
	sh.Slug = slug
	if err := s.repos.Share.Create(ctx, sh); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return s.toResp(sh), nil
}

// Delete 删除分享。
func (s *ShareService) Delete(ctx context.Context, userID, id uint64) error {
	sh, err := s.repos.Share.GetByID(ctx, id)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if sh == nil || sh.UserID != userID {
		return errs.ErrNotFound.WithMessage("分享不存在")
	}
	if err := s.repos.Share.Delete(ctx, id); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	return nil
}

// Access 访问分享内容。
func (s *ShareService) Access(ctx context.Context, slug, password string) (*dto.ShareAccessResp, error) {
	sh, err := s.repos.Share.GetBySlug(ctx, strings.TrimSpace(slug))
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if sh == nil || sh.Status != model.StatusEnabled {
		return nil, errs.ErrNotFound.WithMessage("分享不存在或已失效")
	}
	if sh.ExpiresAt != nil && sh.ExpiresAt.Before(model.Now()) {
		return nil, errs.ErrNotFound.WithMessage("分享已过期")
	}
	if sh.MaxViews > 0 && sh.Views >= sh.MaxViews {
		return nil, errs.ErrNotFound.WithMessage("分享访问次数已达上限")
	}
	if sh.HasPassword() && !auth.VerifyPassword(sh.PasswordHash, password) {
		return nil, errs.ErrForbidden.WithMessage("访问密码错误")
	}
	resp := &dto.ShareAccessResp{Share: s.toResp(sh)}
	switch sh.TargetType {
	case model.ShareTargetImage:
		img, err := s.repos.Image.GetByID(ctx, sh.TargetID)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if img == nil || img.Status != model.ImageStatusNormal {
			return nil, errs.ErrNotFound.WithMessage("分享内容不存在")
		}
		resp.Images = []*dto.ImageResp{s.imageResp(img)}
	case model.ShareTargetAlbum:
		album, err := s.repos.Album.GetByID(ctx, sh.TargetID)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if album == nil {
			return nil, errs.ErrNotFound.WithMessage("分享内容不存在")
		}
		resp.Album = dto.NewAlbumResp(album)
		list, _, err := s.repos.Image.List(ctx, model.ImageQuery{
			UserID:  album.UserID,
			AlbumID: &album.ID,
			Page:    1,
			Size:    200,
		})
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		resp.Images = make([]*dto.ImageResp, 0, len(list))
		for _, img := range list {
			resp.Images = append(resp.Images, s.imageResp(img))
		}
	default:
		return nil, errs.ErrNotFound.WithMessage("分享内容不存在")
	}
	if err := s.repos.Share.IncrViews(ctx, sh.ID); err != nil && s.log != nil {
		s.log.Warn("incr share views failed", logger.Err(err))
	}
	return resp, nil
}

func (s *ShareService) toResp(sh *model.Share) *dto.ShareResp {
	resp := dto.NewShareResp(sh)
	if resp != nil {
		resp.URL = s.cfg.SiteURL() + "/s/" + sh.Slug
	}
	return resp
}

func (s *ShareService) imageResp(img *model.Image) *dto.ImageResp {
	resp := dto.NewImageResp(img)
	resp.URL = imageURL(s.cfg, img)
	resp.ThumbURL = thumbURL(s.cfg, img.Hash, img.Ext)
	return resp
}

func (s *ShareService) checkShareQuota(ctx context.Context, userID uint64) error {
	if s.repos.Group == nil {
		return nil
	}
	u, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if u == nil || u.GroupID == 0 {
		return nil
	}
	g, err := s.repos.Group.GetByID(ctx, u.GroupID)
	if err != nil || g == nil || g.MaxShares <= 0 {
		return nil
	}
	n, err := s.repos.Share.CountByUser(ctx, userID)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if n >= int64(g.MaxShares) {
		return errs.ErrForbidden.WithMessage("分享数量已达上限")
	}
	return nil
}

func (s *ShareService) newSlug(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		slug, err := auth.NewOpaqueToken(5)
		if err != nil {
			return "", errs.ErrInternal.Wrap(err)
		}
		exists, err := s.repos.Share.GetBySlug(ctx, slug)
		if err != nil {
			return "", errs.ErrInternal.Wrap(err)
		}
		if exists == nil {
			return slug, nil
		}
	}
	return "", errs.ErrInternal.WithMessage("生成分享链接失败")
}

func timeHours(h int) (d time.Duration) { return time.Duration(h) * time.Hour }
