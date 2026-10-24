package router

import (
	"testing"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/handler"
	"github.com/pichub/pichub/internal/middleware"
)

// TestRoutesRegister 确保全部路由可以成功注册（Gin 在冲突时会 panic）。
func TestRoutesRegister(t *testing.T) {
	cfg := config.Default()
	h := handler.New(cfg, nil, nil, nil)
	auth := middleware.NewAuthenticator(nil, nil, nil, nil, nil)
	lim := middleware.NewLimiter(nil, nil)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("route registration panicked: %v", r)
		}
	}()
	engine := New(Options{Config: cfg, Handler: h, Auth: auth, Limiter: lim})
	if engine == nil {
		t.Fatal("router.New returned nil engine")
	}
}
