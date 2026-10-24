package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
)

// ImageList 查询图片列表。
func (h *Handler) ImageList(c *gin.Context) {
	var req dto.ImageListReq
	if !BindQuery(c, &req) {
		return
	}
	withTags := c.Query("with_tags") == "1" || c.Query("with_tags") == "true"
	list, total, err := h.svc.Image.List(c.Request.Context(), h.userID(c), req, withTags)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, imagePage(list, total, req))
}

// ImageTrash 查询回收站列表。
func (h *Handler) ImageTrash(c *gin.Context) {
	var req dto.ImageListReq
	if !BindQuery(c, &req) {
		return
	}
	list, total, err := h.svc.Image.Trash(c.Request.Context(), h.userID(c), req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, imagePage(list, total, req))
}

// ImageGet 返回图片详情。
func (h *Handler) ImageGet(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	resp, err := h.svc.Image.Get(c.Request.Context(), h.userID(c), id)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// ImageUpdate 修改图片信息。
func (h *Handler) ImageUpdate(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.ImageUpdateReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Image.Update(c.Request.Context(), h.userID(c), id, &req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// ImageDelete 把图片移入回收站。
func (h *Handler) ImageDelete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Image.Delete(c.Request.Context(), h.userID(c), id); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// ImageHardDelete 永久删除图片。
func (h *Handler) ImageHardDelete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Image.HardDelete(c.Request.Context(), h.userID(c), id); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// ImageBatch 批量操作图片。
func (h *Handler) ImageBatch(c *gin.Context) {
	var req dto.ImageBatchReq
	if !BindJSON(c, &req) {
		return
	}
	n, err := h.svc.Image.Batch(c.Request.Context(), h.userID(c), &req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, dto.ImageBatchResp{Affected: n})
}

// ImageRestore 从回收站还原图片。
func (h *Handler) ImageRestore(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Image.Restore(c.Request.Context(), h.userID(c), id); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// ImageClearTrash 清空回收站。
func (h *Handler) ImageClearTrash(c *gin.Context) {
	n, err := h.svc.Image.ClearTrash(c.Request.Context(), h.userID(c))
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, dto.ImageBatchResp{Affected: n})
}

// ImageReferers 查询图片引用来源。
func (h *Handler) ImageReferers(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	page := queryInt(c, "page", 1)
	size := queryInt(c, "size", 20)
	order := c.DefaultQuery("order", "hits")
	list, total, err := h.svc.Image.Referers(c.Request.Context(), h.userID(c), id, page, size, order)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, dto.PageResp{List: list, Total: total, Size: size})
}

// ImageRefererSummary 返回图片引用汇总。
func (h *Handler) ImageRefererSummary(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	resp, err := h.svc.Image.RefererSummary(c.Request.Context(), h.userID(c), id)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// ImageDeleteReferer 删除单条引用记录。
func (h *Handler) ImageDeleteReferer(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	rid, ok := pathID(c, "rid")
	if !ok {
		return
	}
	if err := h.svc.Image.DeleteReferer(c.Request.Context(), h.userID(c), id, rid); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

func imagePage(list []*dto.ImageResp, total int64, req dto.ImageListReq) dto.PageResp {
	page := dto.PageResp{List: list, Total: total, Size: req.Size}
	if req.Cursor > 0 && len(list) > 0 {
		page.Cursor = list[len(list)-1].ID
	}
	return page
}
