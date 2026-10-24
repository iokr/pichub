package router

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/middleware"
)

func registerAPI(r *gin.Engine, o Options) {
	api := r.Group("/api/v1")
	api.GET("/site", o.Handler.SiteConfig)

	rl := o.Config.Security.RateLimit
	authLimiter := o.Limiter
	required := o.Auth.Required()

	a := api.Group("/auth")
	a.GET("/captcha", o.Handler.Captcha)
	a.POST("/register", authLimiter.ByIP("register", rl.RegisterPerHour, time.Hour), o.Handler.Register)
	a.POST("/login", authLimiter.ByIP("login", rl.LoginPerMinute, time.Minute), o.Handler.Login)
	a.POST("/refresh", o.Handler.Refresh)
	a.GET("/check-username", o.Handler.CheckUsername)
	a.GET("/check-email", o.Handler.CheckEmail)
	a.POST("/verify-email", o.Handler.VerifyEmail)
	a.GET("/verify-email", o.Handler.VerifyEmail)
	a.POST("/resend-verify", o.Handler.ResendVerify)
	a.POST("/forgot-password", o.Handler.ForgotPassword)
	a.POST("/reset-password", o.Handler.ResetPassword)

	api.GET("/s/:slug", o.Handler.ShareAccess)

	authed := api.Group("")
	authed.Use(required)
	authed.Use(authLimiter.ByUser("api", rl.APIPerMinute, time.Minute))
	registerAuthed(authed, o)

	admin := api.Group("/admin")
	admin.Use(required, middleware.RequireAdmin())
	registerAdmin(admin, o)
}

func registerAuthed(g *gin.RouterGroup, o Options) {
	h := o.Handler
	rl := o.Config.Security.RateLimit

	g.POST("/auth/logout", h.Logout)

	g.GET("/user/profile", h.Profile)
	g.PUT("/user/profile", h.UpdateProfile)
	g.PUT("/user/password", h.ChangePassword)
	g.GET("/user/stats", h.UserStats)

	upload := g.Group("/upload")
	upload.Use(middleware.RequireScope("upload"))
	upload.Use(o.Limiter.ByUser("upload", rl.UploadPerMinute, time.Minute))
	upload.POST("", h.Upload)
	upload.POST("/check", h.UploadCheck)
	upload.POST("/url", h.UploadURL)
	upload.POST("/chunk/init", h.ChunkInit)
	upload.POST("/chunk/:id", h.UploadPart)
	upload.POST("/chunk/:id/complete", h.ChunkComplete)

	images := g.Group("/images")
	images.Use(middleware.RequireScope("read"))
	images.GET("", h.ImageList)
	images.GET("/trash", h.ImageTrash)
	images.DELETE("/trash", h.ImageClearTrash)
	images.POST("/batch", h.ImageBatch)
	images.GET("/:id", h.ImageGet)
	images.PATCH("/:id", h.ImageUpdate)
	images.DELETE("/:id", h.ImageDelete)
	images.DELETE("/:id/permanent", h.ImageHardDelete)
	images.POST("/:id/restore", h.ImageRestore)
	images.GET("/:id/referers", h.ImageReferers)
	images.GET("/:id/referers/summary", h.ImageRefererSummary)
	images.DELETE("/:id/referers/:rid", h.ImageDeleteReferer)

	albums := g.Group("/albums")
	albums.GET("", h.AlbumList)
	albums.POST("", h.AlbumCreate)
	albums.GET("/:id", h.AlbumGet)
	albums.PUT("/:id", h.AlbumUpdate)
	albums.DELETE("/:id", h.AlbumDelete)

	tags := g.Group("/tags")
	tags.GET("", h.TagList)
	tags.POST("", h.TagCreate)
	tags.PUT("/:id", h.TagUpdate)
	tags.DELETE("/:id", h.TagDelete)

	tokens := g.Group("/tokens")
	tokens.GET("", h.TokenList)
	tokens.POST("", h.TokenCreate)
	tokens.DELETE("/:id", h.TokenDelete)
	tokens.PUT("/:id/status", h.TokenSetStatus)

	shares := g.Group("/shares")
	shares.GET("", h.ShareList)
	shares.POST("", h.ShareCreate)
	shares.DELETE("/:id", h.ShareDelete)

	storages := g.Group("/storages")
	storages.GET("", h.StorageList)
	storages.POST("", h.StorageCreate)
	storages.PUT("/:id", h.StorageUpdate)
	storages.DELETE("/:id", h.StorageDelete)
	storages.POST("/:id/test", h.StorageTest)

	s3 := g.Group("/s3/credentials")
	s3.GET("", h.S3CredentialList)
	s3.POST("", h.S3CredentialCreate)
	s3.DELETE("/:id", h.S3CredentialDelete)

	stats := g.Group("/stats")
	stats.GET("/referers/top", h.StatsTopReferers)
	stats.GET("/referers/trend", h.StatsRefererTrend)
}

func registerAdmin(g *gin.RouterGroup, o Options) {
	h := o.Handler

	g.GET("/dashboard", h.AdminDashboard)

	g.GET("/users", h.AdminUsers)
	g.PUT("/users/:id/status", h.AdminUpdateUserStatus)
	g.PUT("/users/:id/group", h.AdminUpdateUserGroup)
	g.PUT("/users/:id/password", h.AdminResetUserPassword)

	g.GET("/groups", h.AdminGroups)
	g.POST("/groups", h.AdminCreateGroup)
	g.PUT("/groups/:id", h.AdminUpdateGroup)
	g.DELETE("/groups/:id", h.AdminDeleteGroup)

	g.GET("/settings", h.AdminSettings)
	g.PUT("/settings", h.AdminUpdateSettings)

	g.GET("/logs", h.AdminLogs)
	g.GET("/email-logs", h.AdminEmailLogs)

	g.GET("/invites", h.AdminInvites)
	g.POST("/invites", h.AdminCreateInvites)
	g.DELETE("/invites/:id", h.AdminDeleteInvite)

	g.GET("/storages", h.AdminStorageList)
	g.POST("/storages", h.AdminStorageCreate)
	g.PUT("/storages/:id", h.AdminStorageUpdate)
	g.DELETE("/storages/:id", h.AdminStorageDelete)
	g.POST("/storages/:id/test", h.AdminStorageTest)
}
