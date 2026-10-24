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

// AdminService 提供管理后台能力。
type AdminService struct {
	cfg      *config.Config
	log      *logger.Logger
	repos    Repos
	settings *SettingsService
	stats    *StatsService
	queue    QueueStats
}

// NewAdminService 创建管理服务。
func NewAdminService(d Deps, settings *SettingsService, stats *StatsService) *AdminService {
	return &AdminService{cfg: d.Config, log: d.Log, repos: d.Repos, settings: settings, stats: stats, queue: d.QueueStats}
}

// Dashboard 返回仪表盘数据。
func (s *AdminService) Dashboard(ctx context.Context) (*dto.DashboardResp, error) {
	resp := &dto.DashboardResp{}
	var err error
	if resp.UserCount, err = s.repos.User.Count(ctx); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if resp.ImageCount, err = s.repos.Image.CountAll(ctx); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if resp.StorageBytes, err = s.repos.Image.SumSizeAll(ctx); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	midnight := time.Now().UTC().Truncate(24 * time.Hour)
	if resp.TodayUploads, err = s.repos.Image.CountCreatedSince(ctx, midnight); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if resp.TrashCount, err = s.repos.Image.CountByStatus(ctx, model.ImageStatusTrash); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if resp.AlbumCount, err = s.repos.Album.CountAll(ctx); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if resp.ShareCount, err = s.repos.Share.CountAll(ctx); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if resp.RefererHits, err = s.repos.Referer.TotalHits(ctx); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	groups, err := s.repos.Group.List(ctx)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	resp.GroupCount = int64(len(groups))
	if _, total, err := s.repos.Invite.List(ctx, 1, 1); err == nil {
		resp.InviteCount = total
	}
	if s.queue != nil {
		if ready, delayed, dead, qerr := s.queue.Stats(ctx); qerr == nil {
			resp.QueueReady = ready
			resp.QueueDelayed = delayed
			resp.QueueDead = dead
		}
	}
	return resp, nil
}

// Users 分页查询用户。
func (s *AdminService) Users(ctx context.Context, q model.UserQuery) ([]*dto.AdminUserResp, int64, error) {
	list, total, err := s.repos.User.List(ctx, q)
	if err != nil {
		return nil, 0, errs.ErrInternal.Wrap(err)
	}
	return dto.NewAdminUserRespList(list), total, nil
}

// UpdateUserStatus 启用或禁用用户。
func (s *AdminService) UpdateUserStatus(ctx context.Context, adminID, userID uint64, status int8, reason, ip, ua string) error {
	if status != model.UserStatusNormal && status != model.UserStatusDisabled && status != model.UserStatusPending {
		return errs.ErrInvalidParam.WithDetail("状态值无效")
	}
	u, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if u == nil {
		return errs.ErrNotFound.WithMessage("用户不存在")
	}
	if u.ID == adminID && status != model.UserStatusNormal {
		return errs.ErrForbidden.WithMessage("不能禁用当前登录账号")
	}
	if u.Role == model.RoleAdmin && status != model.UserStatusNormal {
		admins, err := s.repos.User.CountAdmins(ctx)
		if err != nil {
			return errs.ErrInternal.Wrap(err)
		}
		if admins <= 1 {
			return errs.ErrForbidden.WithMessage("不能禁用唯一的管理员")
		}
	}
	if err := s.repos.User.UpdateFields(ctx, userID, map[string]any{"status": status}); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	logOp(ctx, s.log, s.repos.Log, adminID, "admin.user_status", "user", userID, ip, ua, map[string]any{
		"status": status, "reason": reason,
	})
	return nil
}

// UpdateUserGroup 调整用户组与配额。
func (s *AdminService) UpdateUserGroup(ctx context.Context, adminID, userID uint64, req *dto.UpdateUserGroupReq, ip, ua string) error {
	group, err := s.repos.Group.GetByID(ctx, req.GroupID)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if group == nil || group.Status != model.StatusEnabled {
		return errs.ErrInvalidParam.WithDetail("用户组不存在或已禁用")
	}
	fields := map[string]any{
		"group_id":      group.ID,
		"quota_bytes":   group.QuotaBytes,
		"max_file_size": group.MaxFileSize,
		"daily_limit":   group.DailyLimit,
	}
	if req.QuotaBytes != nil {
		fields["quota_bytes"] = *req.QuotaBytes
	}
	if req.MaxFileSize != nil {
		fields["max_file_size"] = *req.MaxFileSize
	}
	if req.DailyLimit != nil {
		fields["daily_limit"] = *req.DailyLimit
	}
	if err := s.repos.User.UpdateFields(ctx, userID, fields); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	logOp(ctx, s.log, s.repos.Log, adminID, "admin.user_group", "user", userID, ip, ua, map[string]any{"group_id": group.ID})
	return nil
}

// ResetUserPassword 管理员重置用户密码。
func (s *AdminService) ResetUserPassword(ctx context.Context, adminID, userID uint64, req *dto.ResetUserPasswordReq, ip, ua string) error {
	if err := auth.ValidatePasswordStrength(&s.cfg.Security, req.NewPassword); err != nil {
		return errs.ErrPasswordWeak.WithMessage(err.Error())
	}
	u, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if u == nil {
		return errs.ErrNotFound.WithMessage("用户不存在")
	}
	hash, err := auth.HashPassword(req.NewPassword, s.cfg.Security.BcryptCost)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if err := s.repos.User.UpdateFields(ctx, userID, map[string]any{"password_hash": hash}); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	logOp(ctx, s.log, s.repos.Log, adminID, "admin.reset_password", "user", userID, ip, ua, nil)
	return nil
}

// Groups 返回全部用户组。
func (s *AdminService) Groups(ctx context.Context) ([]*dto.GroupResp, error) {
	list, err := s.repos.Group.List(ctx)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	out := make([]*dto.GroupResp, 0, len(list))
	for _, g := range list {
		resp := dto.NewGroupResp(g)
		if n, err := s.repos.Group.CountUsers(ctx, g.ID); err == nil {
			resp.UserCount = n
		}
		out = append(out, resp)
	}
	return out, nil
}

// CreateGroup 创建用户组。
func (s *AdminService) CreateGroup(ctx context.Context, adminID uint64, req *dto.GroupCreateReq, ip, ua string) (*dto.GroupResp, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 32 {
		return nil, errs.ErrInvalidParam.WithDetail("用户组名称长度需为 1-32 个字符")
	}
	if req.IsDefault {
		if err := s.clearDefaultGroup(ctx); err != nil {
			return nil, err
		}
	}
	status := int8(model.StatusEnabled)
	if req.Status != nil {
		status = *req.Status
	}
	g := &model.UserGroup{
		Name:               name,
		Description:        trimTo(strings.TrimSpace(req.Description), 255),
		QuotaBytes:         req.QuotaBytes,
		MaxFileSize:        req.MaxFileSize,
		DailyLimit:         req.DailyLimit,
		MaxAlbums:          req.MaxAlbums,
		MaxTokens:          req.MaxTokens,
		MaxShares:          req.MaxShares,
		AllowRegister:      req.AllowRegister != nil && *req.AllowRegister,
		AllowCustomStorage: req.AllowCustomStorage != nil && *req.AllowCustomStorage,
		IsDefault:          req.IsDefault,
		Status:             status,
	}
	if err := s.repos.Group.Create(ctx, g); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	logOp(ctx, s.log, s.repos.Log, adminID, "admin.group_create", "group", g.ID, ip, ua, map[string]any{"name": name})
	return dto.NewGroupResp(g), nil
}

// UpdateGroup 更新用户组。
func (s *AdminService) UpdateGroup(ctx context.Context, adminID, id uint64, req *dto.GroupUpdateReq, ip, ua string) (*dto.GroupResp, error) {
	g, err := s.repos.Group.GetByID(ctx, id)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if g == nil {
		return nil, errs.ErrNotFound.WithMessage("用户组不存在")
	}
	fields := map[string]any{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len([]rune(name)) > 32 {
			return nil, errs.ErrInvalidParam.WithDetail("用户组名称长度需为 1-32 个字符")
		}
		fields["name"] = name
	}
	if req.Description != nil {
		fields["description"] = trimTo(strings.TrimSpace(*req.Description), 255)
	}
	if req.QuotaBytes != nil {
		fields["quota_bytes"] = *req.QuotaBytes
	}
	if req.MaxFileSize != nil {
		fields["max_file_size"] = *req.MaxFileSize
	}
	if req.DailyLimit != nil {
		fields["daily_limit"] = *req.DailyLimit
	}
	if req.MaxAlbums != nil {
		fields["max_albums"] = *req.MaxAlbums
	}
	if req.MaxTokens != nil {
		fields["max_tokens"] = *req.MaxTokens
	}
	if req.MaxShares != nil {
		fields["max_shares"] = *req.MaxShares
	}
	if req.AllowRegister != nil {
		fields["allow_register"] = *req.AllowRegister
	}
	if req.AllowCustomStorage != nil {
		fields["allow_custom_storage"] = *req.AllowCustomStorage
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	if req.IsDefault != nil {
		if *req.IsDefault && !g.IsDefault {
			if err := s.clearDefaultGroup(ctx); err != nil {
				return nil, err
			}
		}
		fields["is_default"] = *req.IsDefault
	}
	if len(fields) == 0 {
		return dto.NewGroupResp(g), nil
	}
	if err := s.repos.Group.UpdateFields(ctx, id, fields); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	logOp(ctx, s.log, s.repos.Log, adminID, "admin.group_update", "group", id, ip, ua, nil)
	updated, err := s.repos.Group.GetByID(ctx, id)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	resp := dto.NewGroupResp(updated)
	if n, err := s.repos.Group.CountUsers(ctx, id); err == nil {
		resp.UserCount = n
	}
	return resp, nil
}

// DeleteGroup 删除用户组。
func (s *AdminService) DeleteGroup(ctx context.Context, adminID, id uint64, ip, ua string) error {
	g, err := s.repos.Group.GetByID(ctx, id)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if g == nil {
		return errs.ErrNotFound.WithMessage("用户组不存在")
	}
	if g.IsDefault {
		return errs.ErrForbidden.WithMessage("默认用户组不能删除")
	}
	n, err := s.repos.Group.CountUsers(ctx, id)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if n > 0 {
		return errs.ErrForbidden.WithMessage("用户组下仍有用户，无法删除")
	}
	if err := s.repos.Group.Delete(ctx, id); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	logOp(ctx, s.log, s.repos.Log, adminID, "admin.group_delete", "group", id, ip, ua, nil)
	return nil
}

// Logs 分页查询操作日志。
func (s *AdminService) Logs(ctx context.Context, q model.LogQuery) ([]*dto.LogResp, int64, error) {
	list, total, err := s.repos.Log.List(ctx, q)
	if err != nil {
		return nil, 0, errs.ErrInternal.Wrap(err)
	}
	return dto.NewLogRespList(list), total, nil
}

// EmailLogs 分页查询邮件发送日志。
func (s *AdminService) EmailLogs(ctx context.Context, q model.EmailLogQuery) ([]*dto.EmailLogResp, int64, error) {
	list, total, err := s.repos.Email.ListLogs(ctx, q)
	if err != nil {
		return nil, 0, errs.ErrInternal.Wrap(err)
	}
	return dto.NewEmailLogRespList(list), total, nil
}

// Invites 分页查询邀请码。
func (s *AdminService) Invites(ctx context.Context, page, size int) ([]*dto.InviteResp, int64, error) {
	list, total, err := s.repos.Invite.List(ctx, page, size)
	if err != nil {
		return nil, 0, errs.ErrInternal.Wrap(err)
	}
	out := make([]*dto.InviteResp, 0, len(list))
	for _, item := range list {
		out = append(out, dto.NewInviteResp(item))
	}
	return out, total, nil
}

// CreateInvites 批量创建邀请码。
func (s *AdminService) CreateInvites(ctx context.Context, adminID uint64, req *dto.InviteCreateReq, ip, ua string) ([]*dto.InviteResp, error) {
	count := req.Count
	if count <= 0 {
		count = 1
	}
	if count > 100 {
		count = 100
	}
	groupID := req.GroupID
	if groupID == 0 {
		groupID = s.defaultGroupID(ctx)
	}
	if groupID > 0 {
		g, err := s.repos.Group.GetByID(ctx, groupID)
		if err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		if g == nil {
			return nil, errs.ErrInvalidParam.WithDetail("用户组不存在")
		}
	}
	out := make([]*dto.InviteResp, 0, count)
	for i := 0; i < count; i++ {
		code, err := s.newInviteCode(ctx)
		if err != nil {
			return out, err
		}
		inv := &model.Invite{
			Code:      code,
			CreatorID: adminID,
			MaxUses:   req.MaxUses,
			GroupID:   groupID,
			Status:    model.StatusEnabled,
		}
		if inv.MaxUses <= 0 {
			inv.MaxUses = 1
		}
		if req.ExpiresInHours > 0 {
			inv.ExpiresAt = ptrTime(model.Now().Add(time.Duration(req.ExpiresInHours) * time.Hour))
		}
		if err := s.repos.Invite.Create(ctx, inv); err != nil {
			return out, errs.ErrInternal.Wrap(err)
		}
		out = append(out, dto.NewInviteResp(inv))
	}
	logOp(ctx, s.log, s.repos.Log, adminID, "admin.invite_create", "invite", 0, ip, ua, map[string]any{"count": len(out)})
	return out, nil
}

// DeleteInvite 删除邀请码。
func (s *AdminService) DeleteInvite(ctx context.Context, adminID, id uint64, ip, ua string) error {
	inv, err := s.repos.Invite.GetByID(ctx, id)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if inv == nil {
		return errs.ErrNotFound.WithMessage("邀请码不存在")
	}
	if err := s.repos.Invite.Delete(ctx, id); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	logOp(ctx, s.log, s.repos.Log, adminID, "admin.invite_delete", "invite", id, ip, ua, nil)
	return nil
}

func (s *AdminService) clearDefaultGroup(ctx context.Context) error {
	list, err := s.repos.Group.List(ctx)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	for _, g := range list {
		if g.IsDefault {
			if err := s.repos.Group.UpdateFields(ctx, g.ID, map[string]any{"is_default": false}); err != nil {
				return errs.ErrInternal.Wrap(err)
			}
		}
	}
	return nil
}

func (s *AdminService) defaultGroupID(ctx context.Context) uint64 {
	if g, err := s.repos.Group.GetDefault(ctx); err == nil && g != nil {
		return g.ID
	}
	return 0
}

func (s *AdminService) newInviteCode(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		code, err := auth.NewOpaqueToken(6)
		if err != nil {
			return "", errs.ErrInternal.Wrap(err)
		}
		exists, err := s.repos.Invite.GetByCode(ctx, code)
		if err != nil {
			return "", errs.ErrInternal.Wrap(err)
		}
		if exists == nil {
			return code, nil
		}
	}
	return "", errs.ErrInternal.WithMessage("生成邀请码失败")
}
