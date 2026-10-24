package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pichub/pichub/internal/auth"
	"github.com/pichub/pichub/internal/cache"
	"github.com/pichub/pichub/internal/dto"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
	"github.com/pichub/pichub/internal/service"
)

// Authenticator 校验 JWT 与 API Token。
type Authenticator struct {
	log       *logger.Logger
	jwt       *auth.Manager
	blacklist *cache.TokenBlacklist
	tokens    *service.TokenService
	users     *service.UserService
}

// NewAuthenticator 创建鉴权中间件。
func NewAuthenticator(log *logger.Logger, jwtMgr *auth.Manager, blacklist *cache.TokenBlacklist, tokens *service.TokenService, users *service.UserService) *Authenticator {
	return &Authenticator{log: log, jwt: jwtMgr, blacklist: blacklist, tokens: tokens, users: users}
}

// Required 要求请求必须携带合法凭据。
func (a *Authenticator) Required() gin.HandlerFunc {
	return a.handle(false)
}

// Optional 尝试解析凭据，但允许匿名访问。
func (a *Authenticator) Optional() gin.HandlerFunc {
	return a.handle(true)
}

func (a *Authenticator) handle(optional bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c)
		if raw == "" {
			if optional {
				c.Next()
				return
			}
			unauthorized(c, errs.ErrUnauthorized)
			return
		}

		ctx := c.Request.Context()
		if strings.HasPrefix(raw, "ph_") {
			user, token, err := a.tokens.ValidateAPIToken(ctx, raw, c.ClientIP())
			if err != nil {
				if optional {
					c.Next()
					return
				}
				unauthorized(c, err)
				return
			}
			scopes := splitScopes(token.Scopes)
			setUser(c, user, AuthTypeAPIToken, nil, scopes)
			c.Next()
			return
		}

		claims, err := a.jwt.ParseAccess(raw)
		if err != nil {
			if optional {
				c.Next()
				return
			}
			if errors.Is(err, auth.ErrTokenExpired) {
				unauthorized(c, errs.ErrTokenExpired)
				return
			}
			unauthorized(c, errs.ErrUnauthorized.WithDetail("访问令牌无效"))
			return
		}
		if a.blacklist != nil {
			if revoked, rerr := a.blacklist.IsRevoked(ctx, claims.ID); rerr == nil && revoked {
				unauthorized(c, errs.ErrUnauthorized.WithDetail("访问令牌已失效"))
				return
			}
		}
		user, err := a.users.Get(ctx, claims.UserID)
		if err != nil {
			if optional {
				c.Next()
				return
			}
			unauthorized(c, errs.ErrUnauthorized.WithDetail("用户不存在"))
			return
		}
		if user.Status == model.UserStatusDisabled {
			unauthorized(c, errs.ErrForbidden.WithMessage("账号已被禁用"))
			return
		}
		setUser(c, user, AuthTypeJWT, claims, nil)
		c.Next()
	}
}

func setUser(c *gin.Context, user *model.User, authType string, claims *auth.Claims, scopes []string) {
	c.Set(CtxUser, user)
	c.Set(CtxAuthType, authType)
	if claims != nil {
		c.Set(CtxClaims, claims)
	}
	if scopes != nil {
		c.Set(CtxScopes, scopes)
	}
}

func bearerToken(c *gin.Context) string {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		return ""
	}
	if len(header) > 7 && strings.EqualFold(header[:7], "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return ""
}

func unauthorized(c *gin.Context, err error) {
	appErr := errs.From(err)
	if appErr == nil {
		appErr = errs.ErrUnauthorized
	}
	c.AbortWithStatusJSON(appErr.HTTPStatus(), dto.Response{Code: appErr.Code(), Message: appErr.Message()})
}

func splitScopes(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
