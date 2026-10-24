package handler

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/imageproc"
	"github.com/pichub/pichub/internal/logger"
)

// ImageOriginal 输出原图，支持动态处理参数。
func (h *Handler) ImageOriginal(c *gin.Context) {
	if !h.checkAntiLeech(c) {
		return
	}
	hash := imageHash(c.Param("hash"))
	if hash == "" {
		Fail(c, errs.ErrNotFound.WithMessage("图片不存在"))
		return
	}
	owner, exp, sign := accessParams(c)
	q := c.Request.URL.Query()
	if hasDynamicParams(q) {
		opt, need, err := h.svc.Image.ParseDynamicOptions(q)
		if err != nil {
			Fail(c, errs.ErrInvalidParam.WithDetail(err.Error()))
			return
		}
		if need {
			res, rerr := h.svc.Image.RenderDynamic(c.Request.Context(), hash, owner, exp, sign, opt)
			if rerr != nil {
				Fail(c, rerr)
				return
			}
			h.writeProcessed(c, hash+"|"+opt.Key(), res)
			return
		}
	}
	h.serveOriginal(c, hash, owner, exp, sign)
}

// ImageThumbnail 输出预设缩略图。
func (h *Handler) ImageThumbnail(c *gin.Context) {
	if !h.checkAntiLeech(c) {
		return
	}
	hash := imageHash(c.Param("hash"))
	if hash == "" {
		Fail(c, errs.ErrNotFound.WithMessage("图片不存在"))
		return
	}
	size := c.Param("size")
	if ext := filepath.Ext(size); ext != "" {
		size = strings.TrimSuffix(size, ext)
	}
	owner, exp, sign := accessParams(c)
	res, err := h.svc.Image.Thumbnail(c.Request.Context(), hash, size, owner, exp, sign)
	if err != nil {
		Fail(c, err)
		return
	}
	h.writeProcessed(c, hash+"|"+size, res)
}

func (h *Handler) serveOriginal(c *gin.Context, hash string, owner uint64, exp int64, sign string) {
	d, err := h.svc.Image.DeliverOriginal(c.Request.Context(), hash, owner, exp, sign)
	if err != nil {
		Fail(c, err)
		return
	}
	h.svc.Referer.Record(c.Request.Context(), d.Image, c.GetHeader("Referer"), c.Request.UserAgent())

	etag := `"` + d.Image.Hash + `"`
	if match := c.GetHeader("If-None-Match"); match != "" && strings.Contains(match, d.Image.Hash) {
		c.Status(http.StatusNotModified)
		return
	}
	c.Header("ETag", etag)
	if d.Image.IsPublic {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Header("Cache-Control", "private, max-age=3600")
	}
	c.Header("X-Content-Type-Options", "nosniff")
	if d.ContentType != "" {
		c.Header("Content-Type", d.ContentType)
	}

	if d.LocalPath != "" {
		f, oerr := os.Open(d.LocalPath)
		if oerr != nil {
			Fail(c, errs.ErrNotFound.WithMessage("图片文件不存在"))
			return
		}
		defer f.Close()
		modTime := d.Info.ModTime
		if modTime.IsZero() {
			if fi, serr := f.Stat(); serr == nil {
				modTime = fi.ModTime()
			}
		}
		http.ServeContent(c.Writer, c.Request, filepath.Base(d.LocalPath), modTime, f)
		return
	}

	if d.Reader == nil {
		Fail(c, errs.ErrNotFound.WithMessage("图片文件不存在"))
		return
	}
	defer d.Reader.Close()
	if d.Info != nil && d.Info.Size > 0 {
		c.Header("Content-Length", strconv.FormatInt(d.Info.Size, 10))
	}
	c.Status(http.StatusOK)
	if _, cerr := io.Copy(c.Writer, d.Reader); cerr != nil && h.log != nil {
		h.log.Warn("stream image failed", logger.Err(cerr), logger.String("hash", hash))
	}
}

func (h *Handler) writeProcessed(c *gin.Context, key string, res *imageproc.Result) {
	if res == nil {
		Fail(c, errs.ErrInternal)
		return
	}
	if match := c.GetHeader("If-None-Match"); match != "" && strings.Contains(match, key) {
		c.Status(http.StatusNotModified)
		return
	}
	c.Header("ETag", `"`+key+`"`)
	c.Header("Cache-Control", "public, max-age=604800")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Type", res.Mime)
	c.Header("Content-Length", strconv.Itoa(len(res.Data)))
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write(res.Data)
}

// checkAntiLeech 校验防盗链，拒绝时已写出响应。
func (h *Handler) checkAntiLeech(c *gin.Context) bool {
	ac := h.cfg.Image.AntiLeech
	if !ac.Enabled {
		return true
	}
	ref := strings.TrimSpace(c.GetHeader("Referer"))
	if ref == "" {
		if ac.AllowEmptyReferer {
			return true
		}
		return h.denyReferer(c)
	}
	u, err := url.Parse(ref)
	if err != nil || u.Hostname() == "" {
		return h.denyReferer(c)
	}
	host := strings.ToLower(u.Hostname())
	if self, perr := url.Parse(h.cfg.SiteURL()); perr == nil {
		name := strings.ToLower(self.Hostname())
		if name != "" && (host == name || strings.HasSuffix(host, "."+name)) {
			return true
		}
	}
	for _, allowed := range ac.RefererWhitelist {
		allowed = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(allowed, ".")))
		if allowed == "" {
			continue
		}
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	return h.denyReferer(c)
}

func (h *Handler) denyReferer(c *gin.Context) bool {
	switch strings.ToLower(h.cfg.Image.AntiLeech.BlockAction) {
	case "forbidden":
		Fail(c, errs.ErrForbidden.WithMessage("该来源不允许访问"))
	default:
		c.Redirect(http.StatusFound, h.cfg.SiteURL()+"/")
		c.Abort()
	}
	return false
}

func imageHash(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if ext := filepath.Ext(raw); ext != "" && imageproc.IsImageExtension(ext) {
		raw = strings.TrimSuffix(raw, ext)
	}
	return raw
}

func accessParams(c *gin.Context) (owner uint64, exp int64, sign string) {
	if v, err := strconv.ParseUint(c.Query("uid"), 10, 64); err == nil {
		owner = v
	}
	if v, err := strconv.ParseInt(c.Query("exp"), 10, 64); err == nil {
		exp = v
	}
	return owner, exp, c.Query("sign")
}

func hasDynamicParams(q url.Values) bool {
	for _, key := range []string{"w", "h", "fit", "q", "format", "blur", "gray"} {
		if q.Get(key) != "" {
			return true
		}
	}
	return false
}
