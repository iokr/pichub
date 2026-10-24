package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
)

// ShareList 返回我的分享列表。
func (h *Handler) ShareList(c *gin.Context) {
	list, err := h.svc.Share.List(c.Request.Context(), h.userID(c))
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// ShareCreate 创建分享。
func (h *Handler) ShareCreate(c *gin.Context) {
	var req dto.ShareCreateReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Share.Create(c.Request.Context(), h.userID(c), &req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// ShareDelete 删除分享。
func (h *Handler) ShareDelete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Share.Delete(c.Request.Context(), h.userID(c), id); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// ShareAccess 访问分享内容（公开）。
func (h *Handler) ShareAccess(c *gin.Context) {
	slug := c.Param("slug")
	password := c.Query("password")
	if password == "" {
		password = c.PostForm("password")
	}
	resp, err := h.svc.Share.Access(c.Request.Context(), slug, password)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}
