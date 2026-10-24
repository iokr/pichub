// Package auth 提供 JWT、密码哈希、API Token 与 URL 签名工具。
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/model"
)

// Token 类型。
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Claims 是 JWT 载荷。
type Claims struct {
	UserID uint64 `json:"uid"`
	Role   int8   `json:"role"`
	Type   string `json:"typ"`
	jwt.RegisteredClaims
}

// TokenPair 是登录/刷新返回的令牌对。
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Manager 负责签发与校验 JWT。
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewManager 创建 JWT 管理器。
func NewManager(cfg *config.SecurityConfig) *Manager {
	return &Manager{
		secret:     []byte(cfg.JWTSecret),
		accessTTL:  cfg.JWTAccessTTL,
		refreshTTL: cfg.JWTRefreshTTL,
	}
}

// GeneratePair 为用户签发访问令牌与刷新令牌。
func (m *Manager) GeneratePair(u *model.User, jti string) (*TokenPair, error) {
	now := time.Now()
	access, err := m.sign(u, TokenTypeAccess, jti+"-a", now, m.accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := m.sign(u, TokenTypeRefresh, jti+"-r", now, m.refreshTTL)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(m.accessTTL.Seconds()),
	}, nil
}

func (m *Manager) sign(u *model.User, typ, jti string, now time.Time, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: u.ID,
		Role:   u.Role,
		Type:   typ,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   u.Username,
			Issuer:    "pichub",
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse 校验并解析令牌。
func (m *Manager) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	}, jwt.WithIssuer("pichub"), jwt.WithLeeway(30*time.Second))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}
	if !parsed.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ParseAccess 解析访问令牌。
func (m *Manager) ParseAccess(token string) (*Claims, error) {
	c, err := m.Parse(token)
	if err != nil {
		return nil, err
	}
	if c.Type != TokenTypeAccess {
		return nil, ErrInvalidToken
	}
	return c, nil
}

// ParseRefresh 解析刷新令牌。
func (m *Manager) ParseRefresh(token string) (*Claims, error) {
	c, err := m.Parse(token)
	if err != nil {
		return nil, err
	}
	if c.Type != TokenTypeRefresh {
		return nil, ErrInvalidToken
	}
	return c, nil
}

// AccessTTL 返回访问令牌有效期。
func (m *Manager) AccessTTL() time.Duration { return m.accessTTL }

// 令牌错误。
var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
)
