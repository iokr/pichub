// Package middleware 提供鉴权、限流、日志等横切关注点。
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/auth"
	"github.com/pichub/pichub/internal/model"
)

// 上下文键名。
const (
	CtxRequestID = "ph_request_id"
	CtxUser      = "ph_user"
	CtxClaims    = "ph_claims"
	CtxAuthType  = "ph_auth_type"
	CtxScopes    = "ph_scopes"
)

// 鉴权方式。
const (
	AuthTypeJWT      = "jwt"
	AuthTypeAPIToken = "api_token"
	AuthTypeS3       = "s3"
)

// User 返回上下文中的用户。
func User(c *gin.Context) *model.User {
	if v, ok := c.Get(CtxUser); ok {
		if u, ok := v.(*model.User); ok {
			return u
		}
	}
	return nil
}

// UserID 返回上下文中的用户 ID。
func UserID(c *gin.Context) uint64 {
	if u := User(c); u != nil {
		return u.ID
	}
	return 0
}

// Claims 返回上下文中的 JWT 载荷。
func Claims(c *gin.Context) *auth.Claims {
	if v, ok := c.Get(CtxClaims); ok {
		if claims, ok := v.(*auth.Claims); ok {
			return claims
		}
	}
	return nil
}

// AuthType 返回本次请求的鉴权方式。
func AuthType(c *gin.Context) string {
	if v, ok := c.Get(CtxAuthType); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Scopes 返回 API Token 的权限范围，JWT 访问时为空表示全部权限。
func Scopes(c *gin.Context) []string {
	if v, ok := c.Get(CtxScopes); ok {
		if s, ok := v.([]string); ok {
			return s
		}
	}
	return nil
}

// RequestID 返回请求 ID。
func RequestID(c *gin.Context) string {
	if v, ok := c.Get(CtxRequestID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
