package service

import (
	"context"
	"strings"

	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
)

// AlbumService 处理相册业务。
type AlbumService struct {
	log      *logger.Logger
	repos    Repos
	settings *SettingsService
}

// NewAlbumService 创建相册服务。
func NewAlbumService(d Deps, settings *SettingsService) *AlbumService {
	return &AlbumService{log: d.Log, repos: d.Repos, settings: settings}
}

// List 返回用户全部相册。
func (s *AlbumService) List(ctx context.Context, userID uint64) ([]*dto.AlbumResp, error) {
	list, err := s.repos.Album.ListByUser(ctx, userID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	out := make([]*dto.AlbumResp, 0, len(list))
	for _, a := range list {
		out = append(out, s.toResp(ctx, a))
	}
	return out, nil
}

// Get 返回相册详情。
func (s *AlbumService) Get(ctx context.Context, userID, id uint64) (*dto.AlbumResp, error) {
	a, err := s.own(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return s.toResp(ctx, a), nil
}

// Create 创建相册。
func (s *AlbumService) Create(ctx context.Context, userID uint64, req *dto.AlbumCreateReq) (*dto.AlbumResp, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 64 {
		return nil, errs.ErrInvalidParam.WithDetail("相册名称长度需为 1-64 个字符")
	}
	if s.repos.Group != nil {
		user, err := s.repos.User.GetByID(ctx, userID)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if user != nil && user.GroupID > 0 {
			if g, err := s.repos.Group.GetByID(ctx, user.GroupID); err == nil && g != nil && g.MaxAlbums > 0 {
				n, err := s.repos.Album.CountByUser(ctx, userID)
				if err != nil {
					return nil, errs.ErrInternal.Wrap(err)
				}
				if n >= int64(g.MaxAlbums) {
					return nil, errs.ErrForbidden.WithMessage("相册数量已达上限")
				}
			}
		}
	}
	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}
	a := &model.Album{
		UserID:      userID,
		Name:        name,
		Description: trimTo(strings.TrimSpace(req.Description), 255),
		IsPublic:    isPublic,
	}
	if err := s.repos.Album.Create(ctx, a); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return dto.NewAlbumResp(a), nil
}

// Update 更新相册。
func (s *AlbumService) Update(ctx context.Context, userID, id uint64, req *dto.AlbumUpdateReq) (*dto.AlbumResp, error) {
	a, err := s.own(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len([]rune(name)) > 64 {
			return nil, errs.ErrInvalidParam.WithDetail("相册名称长度需为 1-64 个字符")
		}
		fields["name"] = name
	}
	if req.Description != nil {
		fields["description"] = trimTo(strings.TrimSpace(*req.Description), 255)
	}
	if req.IsPublic != nil {
		fields["is_public"] = *req.IsPublic
	}
	if req.SortOrder != nil {
		fields["sort_order"] = *req.SortOrder
	}
	if req.CoverImageID != nil {
		if *req.CoverImageID == 0 {
			fields["cover_image_id"] = nil
		} else {
			img, err := s.repos.Image.GetByID(ctx, *req.CoverImageID)
			if err != nil {
				return nil, errs.ErrInternal.Wrap(err)
			}
			if img == nil || img.UserID != userID {
				return nil, errs.ErrInvalidParam.WithDetail("封面图片不存在")
			}
			fields["cover_image_id"] = img.ID
		}
	}
	if len(fields) > 0 {
		if err := s.repos.Album.UpdateFields(ctx, a.ID, fields); err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
	}
	return s.Get(ctx, userID, id)
}

// Delete 删除相册，图片本身保留但解除归属。
func (s *AlbumService) Delete(ctx context.Context, userID, id uint64) error {
	a, err := s.own(ctx, userID, id)
	if err != nil {
		return err
	}
	return s.repos.Tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repos.Album.DetachImages(ctx, a.ID); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		if err := s.repos.Share.DeleteByTarget(ctx, model.ShareTargetAlbum, a.ID); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		if err := s.repos.Album.Delete(ctx, a.ID); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		return nil
	})
}

// RecalcImageCount 重算相册的图片冗余计数。
func (s *AlbumService) RecalcImageCount(ctx context.Context, albumID uint64) {
	if albumID == 0 {
		return
	}
	if err := s.repos.Album.RecalcImageCount(ctx, albumID); err != nil && s.log != nil {
		s.log.Warn("recalc album image count failed", logger.Err(err), logger.Uint64("album_id", albumID))
	}
}

func (s *AlbumService) own(ctx context.Context, userID, id uint64) (*model.Album, error) {
	a, err := s.repos.Album.GetByID(ctx, id)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if a == nil || a.UserID != userID {
		return nil, errs.ErrNotFound.WithMessage("相册不存在")
	}
	return a, nil
}

func (s *AlbumService) toResp(ctx context.Context, a *model.Album) *dto.AlbumResp {
	resp := dto.NewAlbumResp(a)
	if resp == nil {
		return nil
	}
	if a.CoverImageID != nil {
		if img, err := s.repos.Image.GetByID(ctx, *a.CoverImageID); err == nil && img != nil {
			resp.CoverURL = img.URL
		}
	}
	return resp
}
