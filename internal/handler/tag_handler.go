package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
)

// TagList 返回当前用户标签列表。
func (h *Handler) TagList(c *gin.Context) {
	list, err := h.svc.Tag.List(c.Request.Context(), h.userID(c))
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// TagCreate 创建标签。
func (h *Handler) TagCreate(c *gin.Context) {
	var req dto.TagCreateReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Tag.Create(c.Request.Context(), h.userID(c), &req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// TagUpdate 更新标签。
func (h *Handler) TagUpdate(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.TagUpdateReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Tag.Update(c.Request.Context(), h.userID(c), id, &req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// TagDelete 删除标签。
func (h *Handler) TagDelete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Tag.Delete(c.Request.Context(), h.userID(c), id); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}
