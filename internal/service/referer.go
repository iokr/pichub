package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pichub/pichub/internal/cache"
	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/model"
)

type userRefererFlag struct {
	enabled bool
	at      time.Time
}

// RefererService 负责引用统计的采集、落库与查询。
type RefererService struct {
	cfg      *config.Config
	log      *logger.Logger
	repos    Repos
	settings *SettingsService
	buffer   *cache.RefererBuffer

	mu    sync.RWMutex
	flags map[uint64]userRefererFlag
}

// NewRefererService 创建引用统计服务。
func NewRefererService(d Deps, settings *SettingsService) *RefererService {
	return &RefererService{
		cfg: d.Config, log: d.Log, repos: d.Repos, settings: settings, buffer: d.Buffer,
		flags: make(map[uint64]userRefererFlag),
	}
}

// Record 记录一次图片访问引用，写入异步缓冲。
func (s *RefererService) Record(ctx context.Context, img *model.Image, referer, userAgent string) {
	if img == nil || !s.enabled(ctx) || !s.userEnabled(ctx, img.UserID) {
		return
	}
	ref := strings.TrimSpace(referer)
	if ref == "" {
		return
	}
	u, err := url.Parse(ref)
	if err != nil || u.Host == "" {
		return
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || s.isSelf(host) {
		return
	}
	if s.cfg.Referer.ExcludeBots && isBotUA(userAgent, s.cfg.Referer.BotUserAgents) {
		return
	}
	sum := sha256.Sum256([]byte(ref))
	if s.buffer != nil {
		s.buffer.Push(cache.RefererEvent{
			ImageID:     img.ID,
			RefererHash: hex.EncodeToString(sum[:]),
			RefererURL:  trimTo(ref, 1024),
			RefererHost: trimTo(host, 255),
		})
	}
}

// Flush 将缓冲中的引用计数落库。
func (s *RefererService) Flush(ctx context.Context) (int, error) {
	if s.buffer == nil {
		return 0, nil
	}
	items, err := s.buffer.Snapshot(ctx, "")
	if err != nil {
		return 0, errs.ErrInternal.Wrap(err)
	}
	if len(items) == 0 {
		return 0, nil
	}
	now := model.Now()
	rows := make([]*model.ImageReferer, 0, len(items))
	for _, it := range items {
		if it.ImageID == 0 || it.RefererHash == "" {
			continue
		}
		rows = append(rows, &model.ImageReferer{
			ImageID:     it.ImageID,
			RefererHash: it.RefererHash,
			RefererURL:  trimTo(it.RefererURL, 1024),
			RefererHost: trimTo(it.RefererHost, 255),
			HitCount:    it.Hits,
			FirstSeenAt: now,
			LastSeenAt:  now,
		})
	}
	if len(rows) == 0 {
		return 0, nil
	}
	if err := s.repos.Referer.UpsertBatch(ctx, rows); err != nil {
		return 0, errs.ErrInternal.Wrap(err)
	}
	if err := s.repos.Referer.UpsertDailyBatch(ctx, model.Today(), rows); err != nil && s.log != nil {
		s.log.Warn("upsert daily referer stats failed", logger.Err(err))
	}
	return len(rows), nil
}

// CleanupDaily 清理过期的按天聚合数据。
func (s *RefererService) CleanupDaily(ctx context.Context) error {
	days := s.cfg.Referer.DailyRetainDays
	if days <= 0 {
		days = 90
	}
	cutoff := model.Date(time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02"))
	if err := s.repos.Referer.DeleteDailyBefore(ctx, cutoff); err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	return nil
}

// TopHosts 返回用户 TOP 引用域名。
func (s *RefererService) TopHosts(ctx context.Context, userID uint64, limit, days int) ([]*model.HostStat, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if days <= 0 || days > 365 {
		days = 30
	}
	list, err := s.repos.Referer.TopHosts(ctx, userID, time.Now().UTC().AddDate(0, 0, -days), limit)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return list, nil
}

// Trend 返回用户引用趋势。
func (s *RefererService) Trend(ctx context.Context, userID uint64, days int) ([]*model.DayStat, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	now := time.Now().UTC()
	list, err := s.repos.Referer.Trend(ctx, userID, now.AddDate(0, 0, -(days-1)), now)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return list, nil
}

// TotalHits 返回全站引用总次数。
func (s *RefererService) TotalHits(ctx context.Context) (int64, error) {
	n, err := s.repos.Referer.TotalHits(ctx)
	if err != nil {
		return 0, errs.ErrInternal.Wrap(err)
	}
	return n, nil
}

func (s *RefererService) enabled(ctx context.Context) bool {
	return s.cfg.Referer.Enabled && s.settings.GetBool(ctx, KeyRefererEnabled, true)
}

// userEnabled 带 5 分钟内存缓存的用户开关判断，避免每次图片请求查库。
func (s *RefererService) userEnabled(ctx context.Context, userID uint64) bool {
	if !s.cfg.Referer.UserCanDisable {
		return true
	}
	s.mu.RLock()
	flag, ok := s.flags[userID]
	s.mu.RUnlock()
	if ok && time.Since(flag.at) < 5*time.Minute {
		return flag.enabled
	}
	enabled := true
	if u, err := s.repos.User.GetByID(ctx, userID); err == nil && u != nil {
		enabled = u.RefererStatEnabled
	}
	s.mu.Lock()
	s.flags[userID] = userRefererFlag{enabled: enabled, at: time.Now()}
	s.mu.Unlock()
	return enabled
}

func (s *RefererService) isSelf(host string) bool {
	site, err := url.Parse(s.cfg.SiteURL())
	if err != nil || site.Hostname() == "" {
		return false
	}
	self := strings.ToLower(site.Hostname())
	return host == self || strings.HasSuffix(host, "."+self)
}

func isBotUA(ua string, extra []string) bool {
	if ua == "" {
		return false
	}
	lower := strings.ToLower(ua)
	known := []string{"bot", "spider", "crawler", "slurp", "curl/", "wget/", "python-requests", "headlesschrome", "phantomjs"}
	for _, k := range known {
		if strings.Contains(lower, k) {
			return true
		}
	}
	for _, k := range extra {
		if k != "" && strings.Contains(lower, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

// InvalidateUser 清除用户引用开关缓存。
func (s *RefererService) InvalidateUser(userID uint64) {
	s.mu.Lock()
	delete(s.flags, userID)
	s.mu.Unlock()
}

func (s *RefererService) logWarn(msg string, err error) {
	if s.log != nil {
		s.log.Warn(msg, logger.Err(err))
	}
}
