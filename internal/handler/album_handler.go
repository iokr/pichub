package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
)

// AlbumList 返回当前用户相册列表。
func (h *Handler) AlbumList(c *gin.Context) {
	list, err := h.svc.Album.List(c.Request.Context(), h.userID(c))
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// AlbumCreate 创建相册。
func (h *Handler) AlbumCreate(c *gin.Context) {
	var req dto.AlbumCreateReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Album.Create(c.Request.Context(), h.userID(c), &req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// AlbumGet 返回相册详情。
func (h *Handler) AlbumGet(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	resp, err := h.svc.Album.Get(c.Request.Context(), h.userID(c), id)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// AlbumUpdate 更新相册。
func (h *Handler) AlbumUpdate(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.AlbumUpdateReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Album.Update(c.Request.Context(), h.userID(c), id, &req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// AlbumDelete 删除相册。
func (h *Handler) AlbumDelete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Album.Delete(c.Request.Context(), h.userID(c), id); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}
