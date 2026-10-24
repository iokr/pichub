// Package cache 封装 Redis 能力：验证码、限流、去重、引用缓冲与令牌黑名单。
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/mojocn/base64Captcha"
	"github.com/redis/go-redis/v9"

	"github.com/pichub/pichub/internal/config"
)

// CaptchaStore 是基于 Redis 的一次性验证码存储。
type CaptchaStore struct {
	rdb    *redis.Client
	prefix string
	ttl    time.Duration
}

// NewCaptchaStore 创建 Redis 验证码存储。
func NewCaptchaStore(rdb *redis.Client, prefix string, ttl time.Duration) *CaptchaStore {
	return &CaptchaStore{rdb: rdb, prefix: prefix + "captcha:", ttl: ttl}
}

// Set 保存验证码答案。
func (s *CaptchaStore) Set(id string, value string) error {
	return s.rdb.Set(context.Background(), s.prefix+id, value, s.ttl).Err()
}

// Get 读取验证码答案，clear 为 true 时删除。
func (s *CaptchaStore) Get(id string, clear bool) string {
	ctx := context.Background()
	key := s.prefix + id
	val, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	if clear {
		s.rdb.Del(ctx, key)
	}
	return val
}

// Verify 校验验证码，一次性消费。
func (s *CaptchaStore) Verify(id, answer string, clear bool) bool {
	v := s.Get(id, clear)
	return v != "" && equalFold(v, answer)
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// Captcha 图片验证码生成与校验。
type Captcha struct {
	captcha *base64Captcha.Captcha
	expire  int
}

// NewCaptcha 根据配置创建验证码组件。
func NewCaptcha(rdb *redis.Client, cfg config.CaptchaConfig, keyPrefix string) *Captcha {
	store := NewCaptchaStore(rdb, keyPrefix, time.Duration(cfg.ExpireSeconds)*time.Second)

	var driver base64Captcha.Driver
	switch cfg.Type {
	case "math":
		driver = base64Captcha.NewDriverMath(cfg.Height, cfg.Width, cfg.NoiseCount,
			base64Captcha.OptionShowSlimeLine|base64Captcha.OptionShowSineLine, nil, nil, nil)
	default:
		source := "abcdefghjkmnpqrstuvwxyz23456789"
		driver = base64Captcha.NewDriverString(cfg.Height, cfg.Width, cfg.NoiseCount,
			base64Captcha.OptionShowSlimeLine, cfg.Length, source, nil, nil, nil)
	}
	return &Captcha{captcha: base64Captcha.NewCaptcha(driver, store), expire: cfg.ExpireSeconds}
}

// Generate 生成验证码，返回 ID 与 Base64 图片。
func (c *Captcha) Generate() (id, b64 string, err error) {
	id, b64, _, err = c.captcha.Generate()
	if err != nil {
		return "", "", fmt.Errorf("generate captcha: %w", err)
	}
	return id, b64, nil
}

// Verify 校验验证码（大小写不敏感，一次性）。
func (c *Captcha) Verify(id, answer string) bool {
	if id == "" || answer == "" {
		return false
	}
	return c.captcha.Verify(id, answer, true)
}

// ExpireSeconds 返回验证码有效期秒数。
func (c *Captcha) ExpireSeconds() int { return c.expire }
