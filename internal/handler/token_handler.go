package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
)

// TokenList 返回 API Token 列表。
func (h *Handler) TokenList(c *gin.Context) {
	list, err := h.svc.Token.ListTokens(c.Request.Context(), h.userID(c))
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// TokenCreate 创建 API Token。
func (h *Handler) TokenCreate(c *gin.Context) {
	var req dto.TokenCreateReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Token.CreateToken(c.Request.Context(), h.userID(c), &req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// TokenDelete 删除 API Token。
func (h *Handler) TokenDelete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Token.DeleteToken(c.Request.Context(), h.userID(c), id); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// TokenSetStatus 启用或禁用 API Token。
func (h *Handler) TokenSetStatus(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.TokenStatusReq
	if !BindJSON(c, &req) {
		return
	}
	if req.Status != 0 && req.Status != 1 {
		Fail(c, errs.ErrInvalidParam.WithDetail("状态值无效"))
		return
	}
	if err := h.svc.Token.SetTokenStatus(c.Request.Context(), h.userID(c), id, req.Status); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}
