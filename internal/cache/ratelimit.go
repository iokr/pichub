package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// rateLimitScript 原子执行 INCR + 首次设置过期时间。
var rateLimitScript = redis.NewScript(`
local c = redis.call('INCR', KEYS[1])
if c == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return c
`)

// RateLimiter 基于 Redis 的固定窗口限流器。
type RateLimiter struct {
	rdb    *redis.Client
	prefix string
}

// NewRateLimiter 创建限流器。
func NewRateLimiter(rdb *redis.Client, keyPrefix string) *RateLimiter {
	return &RateLimiter{rdb: rdb, prefix: keyPrefix + "rl:"}
}

// Allow 判断 key 是否未超过窗口内限额，并返回当前计数。
func (l *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int64, error) {
	if limit <= 0 {
		return true, 0, nil
	}
	cnt, err := rateLimitScript.Run(ctx, l.rdb, []string{l.prefix + key}, window.Milliseconds()).Int64()
	if err != nil {
		return false, 0, err
	}
	return cnt <= int64(limit), cnt, nil
}

// Count 返回当前窗口计数，不做自增。
func (l *RateLimiter) Count(ctx context.Context, key string) (int64, error) {
	v, err := l.rdb.Get(ctx, l.prefix+key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	return strconv.ParseInt(v, 10, 64)
}

// Reset 清除计数。
func (l *RateLimiter) Reset(ctx context.Context, key string) error {
	return l.rdb.Del(ctx, l.prefix+key).Err()
}

// Incr 累加并返回计数，用于登录失败次数统计。
func (l *RateLimiter) Incr(ctx context.Context, key string, window time.Duration) (int64, error) {
	return rateLimitScript.Run(ctx, l.rdb, []string{l.prefix + key}, window.Milliseconds()).Int64()
}

// TokenBlacklist 基于 Redis 的 JWT 黑名单。
type TokenBlacklist struct {
	rdb    *redis.Client
	prefix string
}

// NewTokenBlacklist 创建黑名单。
func NewTokenBlacklist(rdb *redis.Client, keyPrefix string) *TokenBlacklist {
	return &TokenBlacklist{rdb: rdb, prefix: keyPrefix + "jwt:black:"}
}

// Revoke 在令牌过期前将其加入黑名单。
func (b *TokenBlacklist) Revoke(ctx context.Context, jti string, expireAt time.Time) error {
	ttl := time.Until(expireAt)
	if ttl <= 0 {
		return nil
	}
	return b.rdb.Set(ctx, b.prefix+jti, "1", ttl).Err()
}

// IsRevoked 判断令牌是否已注销。
func (b *TokenBlacklist) IsRevoked(ctx context.Context, jti string) (bool, error) {
	n, err := b.rdb.Exists(ctx, b.prefix+jti).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Deduper 基于 Redis SETNX 的一次性标记（防重放/防重复提交）。
type Deduper struct {
	rdb    *redis.Client
	prefix string
}

// NewDeduper 创建去重组件。
func NewDeduper(rdb *redis.Client, keyPrefix string) *Deduper {
	return &Deduper{rdb: rdb, prefix: keyPrefix + "dedup:"}
}

// MarkOnce 首次标记返回 true，重复返回 false。
func (d *Deduper) MarkOnce(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return d.rdb.SetNX(ctx, d.prefix+key, "1", ttl).Result()
}
