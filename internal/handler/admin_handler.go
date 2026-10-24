package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/model"
)

// AdminDashboard 返回管理后台仪表盘数据。
func (h *Handler) AdminDashboard(c *gin.Context) {
	resp, err := h.svc.Admin.Dashboard(c.Request.Context())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// AdminUsers 返回用户列表。
func (h *Handler) AdminUsers(c *gin.Context) {
	q := model.UserQuery{
		Keyword: c.Query("keyword"),
		Status:  queryInt8(c, "status"),
		Role:    queryInt8(c, "role"),
		GroupID: queryUint(c, "group_id"),
		Page:    queryInt(c, "page", 1),
		Size:    queryInt(c, "size", 20),
	}
	list, total, err := h.svc.Admin.Users(c.Request.Context(), q)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, dto.PageResp{List: list, Total: total, Size: q.Size})
}

// AdminUpdateUserStatus 启用或禁用用户。
func (h *Handler) AdminUpdateUserStatus(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateUserStatusReq
	if !BindJSON(c, &req) {
		return
	}
	ip, ua := h.meta(c)
	if err := h.svc.Admin.UpdateUserStatus(c.Request.Context(), h.userID(c), id, req.Status, req.Reason, ip, ua); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// AdminUpdateUserGroup 调整用户组与配额。
func (h *Handler) AdminUpdateUserGroup(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateUserGroupReq
	if !BindJSON(c, &req) {
		return
	}
	ip, ua := h.meta(c)
	if err := h.svc.Admin.UpdateUserGroup(c.Request.Context(), h.userID(c), id, &req, ip, ua); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// AdminResetUserPassword 重置用户密码。
func (h *Handler) AdminResetUserPassword(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.ResetUserPasswordReq
	if !BindJSON(c, &req) {
		return
	}
	ip, ua := h.meta(c)
	if err := h.svc.Admin.ResetUserPassword(c.Request.Context(), h.userID(c), id, &req, ip, ua); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// AdminGroups 返回用户组列表。
func (h *Handler) AdminGroups(c *gin.Context) {
	list, err := h.svc.Admin.Groups(c.Request.Context())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// AdminCreateGroup 创建用户组。
func (h *Handler) AdminCreateGroup(c *gin.Context) {
	var req dto.GroupCreateReq
	if !BindJSON(c, &req) {
		return
	}
	ip, ua := h.meta(c)
	resp, err := h.svc.Admin.CreateGroup(c.Request.Context(), h.userID(c), &req, ip, ua)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// AdminUpdateGroup 更新用户组。
func (h *Handler) AdminUpdateGroup(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.GroupUpdateReq
	if !BindJSON(c, &req) {
		return
	}
	ip, ua := h.meta(c)
	resp, err := h.svc.Admin.UpdateGroup(c.Request.Context(), h.userID(c), id, &req, ip, ua)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// AdminDeleteGroup 删除用户组。
func (h *Handler) AdminDeleteGroup(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	ip, ua := h.meta(c)
	if err := h.svc.Admin.DeleteGroup(c.Request.Context(), h.userID(c), id, ip, ua); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// AdminSettings 返回系统设置列表。
func (h *Handler) AdminSettings(c *gin.Context) {
	list, err := h.svc.Settings.List(c.Request.Context())
	if err != nil {
		Fail(c, err)
		return
	}
	out := make([]*dto.SettingResp, 0, len(list))
	for _, item := range list {
		out = append(out, dto.NewSettingResp(item))
	}
	OK(c, gin.H{"list": out})
}

// AdminUpdateSettings 批量更新系统设置。
func (h *Handler) AdminUpdateSettings(c *gin.Context) {
	var req dto.UpdateSettingsReq
	if !BindJSON(c, &req) {
		return
	}
	ip, ua := h.meta(c)
	if err := h.svc.Settings.Update(c.Request.Context(), req.Settings, h.userID(c), ip, ua); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// AdminLogs 返回操作日志。
func (h *Handler) AdminLogs(c *gin.Context) {
	q := model.LogQuery{
		UserID: queryUint(c, "user_id"),
		Action: c.Query("action"),
		Page:   queryInt(c, "page", 1),
		Size:   queryInt(c, "size", 20),
	}
	list, total, err := h.svc.Admin.Logs(c.Request.Context(), q)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, dto.PageResp{List: list, Total: total, Size: q.Size})
}

// AdminEmailLogs 返回邮件日志。
func (h *Handler) AdminEmailLogs(c *gin.Context) {
	q := model.EmailLogQuery{
		UserID: queryUint(c, "user_id"),
		Status: queryInt8(c, "status"),
		Page:   queryInt(c, "page", 1),
		Size:   queryInt(c, "size", 20),
	}
	list, total, err := h.svc.Admin.EmailLogs(c.Request.Context(), q)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, dto.PageResp{List: list, Total: total, Size: q.Size})
}

// AdminInvites 返回邀请码列表。
func (h *Handler) AdminInvites(c *gin.Context) {
	page := queryInt(c, "page", 1)
	size := queryInt(c, "size", 20)
	list, total, err := h.svc.Admin.Invites(c.Request.Context(), page, size)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, dto.PageResp{List: list, Total: total, Size: size})
}

// AdminCreateInvites 批量创建邀请码。
func (h *Handler) AdminCreateInvites(c *gin.Context) {
	var req dto.InviteCreateReq
	if !BindJSON(c, &req) {
		return
	}
	ip, ua := h.meta(c)
	list, err := h.svc.Admin.CreateInvites(c.Request.Context(), h.userID(c), &req, ip, ua)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// AdminDeleteInvite 删除邀请码。
func (h *Handler) AdminDeleteInvite(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	ip, ua := h.meta(c)
	if err := h.svc.Admin.DeleteInvite(c.Request.Context(), h.userID(c), id, ip, ua); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}
