package service

import (
	"context"
	"regexp"
	"strings"

	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
)

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// TagService 处理标签业务。
type TagService struct {
	log   *logger.Logger
	repos Repos
}

// NewTagService 创建标签服务。
func NewTagService(d Deps) *TagService {
	return &TagService{log: d.Log, repos: d.Repos}
}

// List 返回用户全部标签。
func (s *TagService) List(ctx context.Context, userID uint64) ([]*dto.TagResp, error) {
	list, err := s.repos.Tag.ListByUser(ctx, userID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	out := make([]*dto.TagResp, 0, len(list))
	for _, t := range list {
		out = append(out, dto.NewTagResp(t))
	}
	return out, nil
}

// Create 创建标签。
func (s *TagService) Create(ctx context.Context, userID uint64, req *dto.TagCreateReq) (*dto.TagResp, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 32 {
		return nil, errs.ErrInvalidParam.WithDetail("标签名称长度需为 1-32 个字符")
	}
	color := strings.TrimSpace(req.Color)
	if color != "" && !colorPattern.MatchString(color) {
		return nil, errs.ErrInvalidParam.WithDetail("颜色需为 #RRGGBB 格式")
	}
	exists, err := s.repos.Tag.GetByUserName(ctx, userID, name)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if exists != nil {
		return nil, errs.ErrInvalidParam.WithDetail("标签已存在")
	}
	t := &model.Tag{UserID: userID, Name: name, Color: color}
	if err := s.repos.Tag.Create(ctx, t); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return dto.NewTagResp(t), nil
}

// Update 更新标签。
func (s *TagService) Update(ctx context.Context, userID, id uint64, req *dto.TagUpdateReq) (*dto.TagResp, error) {
	t, err := s.own(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len([]rune(name)) > 32 {
			return nil, errs.ErrInvalidParam.WithDetail("标签名称长度需为 1-32 个字符")
		}
		if name != t.Name {
			exists, err := s.repos.Tag.GetByUserName(ctx, userID, name)
			if err != nil {
				return nil, errs.ErrInternal.Wrap(err)
			}
			if exists != nil {
				return nil, errs.ErrInvalidParam.WithDetail("标签已存在")
			}
		}
		fields["name"] = name
	}
	if req.Color != nil {
		color := strings.TrimSpace(*req.Color)
		if color != "" && !colorPattern.MatchString(color) {
			return nil, errs.ErrInvalidParam.WithDetail("颜色需为 #RRGGBB 格式")
		}
		fields["color"] = color
	}
	if len(fields) > 0 {
		if err := s.repos.Tag.UpdateFields(ctx, t.ID, fields); err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
	}
	if req.Name != nil {
		t.Name = strings.TrimSpace(*req.Name)
	}
	if req.Color != nil {
		t.Color = strings.TrimSpace(*req.Color)
	}
	return dto.NewTagResp(t), nil
}

// Delete 删除标签并解除全部关联。
func (s *TagService) Delete(ctx context.Context, userID, id uint64) error {
	t, err := s.own(ctx, userID, id)
	if err != nil {
		return err
	}
	return s.repos.Tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repos.Tag.DetachByTag(ctx, t.ID); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		if err := s.repos.Tag.Delete(ctx, t.ID); err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		return nil
	})
}

// ResolveNames 把标签名列表解析为标签 ID，不存在时自动创建。
func (s *TagService) ResolveNames(ctx context.Context, userID uint64, names []string) ([]uint64, error) {
	ids := make([]uint64, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" || len([]rune(name)) > 32 {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if len(ids) >= 20 {
			break
		}
		t, err := s.repos.Tag.GetByUserName(ctx, userID, name)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if t == nil {
			t = &model.Tag{UserID: userID, Name: name}
			if err := s.repos.Tag.Create(ctx, t); err != nil {
				return nil, errs.ErrInternal.Wrap(err)
			}
		}
		ids = append(ids, t.ID)
	}
	return ids, nil
}

// FindIDsByNames 仅查询已存在的标签 ID，不创建。
func (s *TagService) FindIDsByNames(ctx context.Context, userID uint64, names []string) ([]uint64, error) {
	ids := make([]uint64, 0, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		t, err := s.repos.Tag.GetByUserName(ctx, userID, name)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if t != nil {
			ids = append(ids, t.ID)
		}
	}
	return ids, nil
}

// ListByImage 返回图片关联的标签。
func (s *TagService) ListByImage(ctx context.Context, imageID uint64) ([]*dto.TagResp, error) {
	ids, err := s.repos.Tag.ListByImage(ctx, imageID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	out := make([]*dto.TagResp, 0, len(ids))
	for _, id := range ids {
		t, err := s.repos.Tag.GetByID(ctx, id)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if t != nil {
			out = append(out, dto.NewTagResp(t))
		}
	}
	return out, nil
}

// SyncImageTags 用给定标签名替换图片的标签关联，并维护计数。
func (s *TagService) SyncImageTags(ctx context.Context, userID, imageID uint64, names []string) ([]*dto.TagResp, error) {
	oldIDs, err := s.repos.Tag.ListByImage(ctx, imageID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	newIDs, err := s.ResolveNames(ctx, userID, names)
	if err != nil {
		return nil, err
	}
	if err := s.repos.Tag.DetachAll(ctx, imageID); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if err := s.repos.Tag.Attach(ctx, imageID, newIDs); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	oldSet := make(map[uint64]struct{}, len(oldIDs))
	for _, id := range oldIDs {
		oldSet[id] = struct{}{}
	}
	newSet := make(map[uint64]struct{}, len(newIDs))
	for _, id := range newIDs {
		newSet[id] = struct{}{}
	}
	for id := range oldSet {
		if _, ok := newSet[id]; !ok {
			if err := s.repos.Tag.IncrImageCount(ctx, id, -1); err != nil && s.log != nil {
				s.log.Warn("decr tag image count failed", logger.Err(err))
			}
		}
	}
	for id := range newSet {
		if _, ok := oldSet[id]; !ok {
			if err := s.repos.Tag.IncrImageCount(ctx, id, 1); err != nil && s.log != nil {
				s.log.Warn("incr tag image count failed", logger.Err(err))
			}
		}
	}
	return s.ListByImage(ctx, imageID)
}

func (s *TagService) own(ctx context.Context, userID, id uint64) (*model.Tag, error) {
	t, err := s.repos.Tag.GetByID(ctx, id)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if t == nil || t.UserID != userID {
		return nil, errs.ErrNotFound.WithMessage("标签不存在")
	}
	return t, nil
}
