package router

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/handler"
)

const devPage = `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>PicHub</title>
<style>body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Microsoft YaHei",sans-serif;background:#F7F8FA;color:#1F2329;display:flex;align-items:center;justify-content:center;height:100vh;margin:0}
main{background:#fff;border-radius:12px;padding:40px 48px;box-shadow:0 4px 12px rgba(0,0,0,.06);max-width:520px}
h1{font-size:20px;margin:0 0 12px}p{color:#4E5969;line-height:1.7;margin:6px 0}code{background:#F2F3F5;padding:2px 6px;border-radius:4px}</style>
</head><body><main><h1>PicHub 后端已启动</h1>
<p>当前二进制未嵌入前端资源。</p>
<p>开发模式请运行 <code>npm run dev</code>，生产模式请先构建前端后使用 <code>go build -tags=embed</code>。</p>
</main></body></html>`

func registerStatic(r *gin.Engine, o Options) {
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || path == "/api" {
			handler.Fail(c, errs.ErrNotFound.WithMessage("接口不存在"))
			return
		}
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.JSON(http.StatusMethodNotAllowed, gin.H{"code": errs.CodeInvalidParam, "message": "方法不允许"})
			return
		}
		if o.Web == nil {
			if path == "/" {
				c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(devPage))
				return
			}
			c.String(http.StatusNotFound, "frontend not embedded")
			return
		}
		if serveAsset(c, o.Web, path) {
			return
		}
		serveIndex(c, o.Web)
	})
}

func serveAsset(c *gin.Context, fsys fs.FS, urlPath string) bool {
	name := strings.TrimPrefix(urlPath, "/")
	if name == "" || strings.Contains(name, "..") {
		return false
	}
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		return false
	}
	if strings.HasPrefix(name, "assets/") {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Header("Cache-Control", "no-cache")
	}
	if ct := mime.TypeByExtension(filepath.Ext(name)); ct != "" {
		c.Header("Content-Type", ct)
	}
	if rs, ok := f.(io.ReadSeeker); ok {
		http.ServeContent(c.Writer, c.Request, filepath.Base(name), st.ModTime(), rs)
		return true
	}
	data, rerr := io.ReadAll(f)
	if rerr != nil {
		return false
	}
	c.Data(http.StatusOK, contentType(name), data)
	return true
}

func serveIndex(c *gin.Context, fsys fs.FS) {
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		c.String(http.StatusNotFound, "index.html not found")
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

func contentType(name string) string {
	if ct := mime.TypeByExtension(filepath.Ext(name)); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
