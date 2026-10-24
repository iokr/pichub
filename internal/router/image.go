package router

import (
	"time"

	"github.com/gin-gonic/gin"
)

func registerImage(r *gin.Engine, o Options) {
	limit := o.Limiter.ByIP("image", o.Config.Security.RateLimit.ImagePerMinute, time.Minute)

	original := r.Group("/i")
	original.Use(limit)
	original.GET("/:hash", o.Handler.ImageOriginal)

	thumb := r.Group("/t")
	thumb.Use(limit)
	thumb.GET("/:hash/:size", o.Handler.ImageThumbnail)
}
