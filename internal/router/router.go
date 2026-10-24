// Package router 负责创建 Gin Engine 并注册全部路由。
package router

import (
	"io/fs"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/handler"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/middleware"
	"github.com/pichub/pichub/internal/s3gw"
)

// Options 是路由装配所需的依赖。
type Options struct {
	Config  *config.Config
	Log     *logger.Logger
	Handler *handler.Handler
	Auth    *middleware.Authenticator
	Limiter *middleware.Limiter
	S3      *s3gw.Gateway
	Web     fs.FS
}

// New 创建 Gin Engine。
func New(o Options) *gin.Engine {
	switch strings.ToLower(o.Config.Server.Mode) {
	case "release":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	if len(o.Config.Server.TrustedProxies) > 0 {
		_ = r.SetTrustedProxies(o.Config.Server.TrustedProxies)
	} else {
		_ = r.SetTrustedProxies(nil)
	}
	r.Use(
		middleware.WithRequestID(),
		middleware.AccessLogger(o.Log),
		middleware.Recovery(o.Log),
		middleware.CORS(),
	)
	r.MaxMultipartMemory = 16 << 20

	r.GET("/healthz", o.Handler.Healthz)
	r.GET("/readyz", o.Handler.Readyz)

	registerAPI(r, o)
	registerImage(r, o)
	if o.S3 != nil {
		o.S3.Register(r)
	}
	registerStatic(r, o)
	return r
}
