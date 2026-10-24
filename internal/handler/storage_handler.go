package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
)

// StorageList 返回当前用户的存储策略。
func (h *Handler) StorageList(c *gin.Context) {
	list, err := h.svc.Storage.List(c.Request.Context(), h.user(c), false)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// StorageCreate 创建存储策略。
func (h *Handler) StorageCreate(c *gin.Context) {
	var req dto.StorageCreateReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Storage.Create(c.Request.Context(), h.user(c), &req, false)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// StorageUpdate 更新存储策略。
func (h *Handler) StorageUpdate(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.StorageUpdateReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Storage.Update(c.Request.Context(), h.user(c), id, &req, false)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// StorageDelete 删除存储策略。
func (h *Handler) StorageDelete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Storage.Delete(c.Request.Context(), h.user(c), id, false); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// StorageTest 测试存储策略连通性。
func (h *Handler) StorageTest(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	resp, err := h.svc.Storage.Test(c.Request.Context(), h.user(c), id, false)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// AdminStorageList 返回全部存储策略。
func (h *Handler) AdminStorageList(c *gin.Context) {
	list, err := h.svc.Storage.List(c.Request.Context(), h.user(c), true)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// AdminStorageCreate 创建系统存储策略。
func (h *Handler) AdminStorageCreate(c *gin.Context) {
	var req dto.StorageCreateReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Storage.Create(c.Request.Context(), h.user(c), &req, true)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// AdminStorageUpdate 更新系统存储策略。
func (h *Handler) AdminStorageUpdate(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.StorageUpdateReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Storage.Update(c.Request.Context(), h.user(c), id, &req, true)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// AdminStorageDelete 删除系统存储策略。
func (h *Handler) AdminStorageDelete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Storage.Delete(c.Request.Context(), h.user(c), id, true); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// AdminStorageTest 测试系统存储策略连通性。
func (h *Handler) AdminStorageTest(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	resp, err := h.svc.Storage.Test(c.Request.Context(), h.user(c), id, true)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}
