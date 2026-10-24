package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
)

// Profile 返回当前用户资料。
func (h *Handler) Profile(c *gin.Context) {
	resp, err := h.svc.User.Profile(c.Request.Context(), h.userID(c))
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// UpdateProfile 修改当前用户资料。
func (h *Handler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateProfileReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.User.UpdateProfile(c.Request.Context(), h.userID(c), &req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// ChangePassword 修改密码。
func (h *Handler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordReq
	if !BindJSON(c, &req) {
		return
	}
	ip, ua := h.meta(c)
	if err := h.svc.User.ChangePassword(c.Request.Context(), h.userID(c), &req, ip, ua); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// UserStats 返回当前用户用量统计。
func (h *Handler) UserStats(c *gin.Context) {
	resp, err := h.svc.User.Stats(c.Request.Context(), h.userID(c))
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}
