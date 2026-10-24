package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
)

// S3CredentialList 返回 S3 凭据列表。
func (h *Handler) S3CredentialList(c *gin.Context) {
	list, err := h.svc.Token.ListS3Credentials(c.Request.Context(), h.userID(c))
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// S3CredentialCreate 生成 S3 凭据。
func (h *Handler) S3CredentialCreate(c *gin.Context) {
	var req dto.S3CredCreateReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Token.CreateS3Credential(c.Request.Context(), h.userID(c), &req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// S3CredentialDelete 删除 S3 凭据。
func (h *Handler) S3CredentialDelete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Token.DeleteS3Credential(c.Request.Context(), h.userID(c), id); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}
