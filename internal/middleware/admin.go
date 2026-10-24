package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/model"
)

// RequireAdmin 要求当前用户为管理员。
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := User(c)
		if u == nil {
			unauthorized(c, errs.ErrUnauthorized)
			return
		}
		if u.Role != model.RoleAdmin {
			c.AbortWithStatusJSON(errs.ErrForbidden.HTTPStatus(), dto.Response{
				Code: errs.ErrForbidden.Code(), Message: "需要管理员权限",
			})
			return
		}
		c.Next()
	}
}

// RequireScope 校验 API Token 的权限范围，JWT 访问不受限。
func RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthType(c) != AuthTypeAPIToken {
			c.Next()
			return
		}
		for _, s := range Scopes(c) {
			if s == scope || s == "*" {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(errs.ErrForbidden.HTTPStatus(), dto.Response{
			Code: errs.ErrForbidden.Code(), Message: "令牌缺少 " + scope + " 权限",
		})
	}
}
