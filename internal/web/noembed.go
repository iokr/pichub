//go:build !embed || noembed

// Package web 提供内嵌前端静态资源。
package web

import "io/fs"

// FS 在未启用 embed 标签时返回空，前端由 Vite 开发服务器提供。
func FS() (fs.FS, bool) { return nil, false }
