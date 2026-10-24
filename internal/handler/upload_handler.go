package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/model"
	"github.com/pichub/pichub/internal/service"
)

// Upload 上传单张图片（multipart/form-data）。
func (h *Handler) Upload(c *gin.Context) {
	user := h.user(c)
	limit := h.maxUploadBytes(user)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit+1<<20)

	fh, err := c.FormFile("file")
	if err != nil {
		Fail(c, uploadBindError(err))
		return
	}
	if fh.Size <= 0 {
		Fail(c, errs.ErrInvalidParam.WithDetail("文件为空"))
		return
	}
	if fh.Size > limit {
		Fail(c, errs.ErrFileTooLarge.WithMessage("文件超过大小限制"))
		return
	}
	f, err := fh.Open()
	if err != nil {
		Fail(c, errs.ErrInternal.Wrap(err))
		return
	}
	defer f.Close()

	opts := h.uploadOptions(c, fh.Filename)
	resp, err := h.svc.Upload.Upload(c.Request.Context(), user, f, fh.Size, opts)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// UploadCheck 秒传检测。
func (h *Handler) UploadCheck(c *gin.Context) {
	var req dto.UploadCheckReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Upload.Check(c.Request.Context(), h.user(c), &req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// UploadURL 远程拉取图片。
func (h *Handler) UploadURL(c *gin.Context) {
	var req dto.UploadURLReq
	if !BindJSON(c, &req) {
		return
	}
	ip, ua := h.meta(c)
	resp, err := h.svc.Upload.UploadURL(c.Request.Context(), h.user(c), &req, ip, ua)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// ChunkInit 初始化分片上传。
func (h *Handler) ChunkInit(c *gin.Context) {
	var req dto.ChunkInitReq
	if !BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Upload.ChunkInit(c.Request.Context(), h.user(c), &req)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

// UploadPart 上传单个分片。
func (h *Handler) UploadPart(c *gin.Context) {
	uploadID := c.Param("id")
	if uploadID == "" {
		Fail(c, errs.ErrInvalidParam.WithDetail("缺少上传 ID"))
		return
	}
	index, err := chunkIndex(c)
	if err != nil {
		Fail(c, err)
		return
	}
	fh, ferr := c.FormFile("file")
	if ferr != nil {
		Fail(c, uploadBindError(ferr))
		return
	}
	f, oerr := fh.Open()
	if oerr != nil {
		Fail(c, errs.ErrInternal.Wrap(oerr))
		return
	}
	defer f.Close()
	if uerr := h.svc.Upload.UploadPart(c.Request.Context(), h.user(c), uploadID, index, f, fh.Size); uerr != nil {
		Fail(c, uerr)
		return
	}
	OK(c, gin.H{"index": index, "size": fh.Size})
}

// ChunkComplete 合并分片并入库。
func (h *Handler) ChunkComplete(c *gin.Context) {
	uploadID := c.Param("id")
	if uploadID == "" {
		Fail(c, errs.ErrInvalidParam.WithDetail("缺少上传 ID"))
		return
	}
	var req dto.ChunkCompleteReq
	_ = c.ShouldBindJSON(&req)
	ip, ua := h.meta(c)
	resp, err := h.svc.Upload.ChunkComplete(c.Request.Context(), h.user(c), uploadID, ip, ua)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, resp)
}

func (h *Handler) uploadOptions(c *gin.Context, filename string) service.UploadOptions {
	opts := service.UploadOptions{
		Filename:  filename,
		Tags:      c.PostForm("tags"),
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
	if raw := c.PostForm("album_id"); raw != "" {
		if id, err := strconv.ParseUint(raw, 10, 64); err == nil && id > 0 {
			opts.AlbumID = &id
		}
	}
	if raw := c.PostForm("is_public"); raw != "" {
		if b, err := strconv.ParseBool(raw); err == nil {
			opts.IsPublic = &b
		}
	}
	if raw := c.PostForm("storage_id"); raw != "" {
		if id, err := strconv.ParseUint(raw, 10, 64); err == nil {
			opts.StorageID = id
		}
	}
	return opts
}

func (h *Handler) maxUploadBytes(user *model.User) int64 {
	if user != nil && user.MaxFileSize > 0 {
		return user.MaxFileSize
	}
	if h.cfg.Upload.MaxFileSize > 0 {
		return h.cfg.Upload.MaxFileSize
	}
	return 10 << 20
}

func chunkIndex(c *gin.Context) (int, error) {
	for _, key := range []string{"index", "part", "part_number", "partNumber"} {
		raw := c.PostForm(key)
		if raw == "" {
			raw = c.Query(key)
		}
		if raw == "" {
			continue
		}
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			return 0, errs.ErrInvalidParam.WithDetail("分片序号不合法")
		}
		return v, nil
	}
	return 0, errs.ErrInvalidParam.WithDetail("缺少分片序号")
}

func uploadBindError(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "too large") || strings.Contains(msg, "MaxBytesError") {
		return errs.ErrFileTooLarge.WithMessage("文件超过大小限制")
	}
	if err == http.ErrMissingFile || strings.Contains(msg, "no such file") {
		return errs.ErrInvalidParam.WithDetail("缺少文件字段 file")
	}
	if err == io.EOF {
		return errs.ErrInvalidParam.WithDetail("请求体为空")
	}
	return errs.ErrInvalidParam.WithDetail(msg)
}
