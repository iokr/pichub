package service

import (
	"context"

	"github.com/pichub/pichub/internal/errs"
	"github.com/pichub/pichub/internal/model"
)

// StatsService 提供引用统计查询。
type StatsService struct {
	repos    Repos
	referers *RefererService
}

// NewStatsService 创建统计服务。
func NewStatsService(d Deps, referers *RefererService) *StatsService {
	return &StatsService{repos: d.Repos, referers: referers}
}

// TopReferers 返回用户 TOP 引用域名。
func (s *StatsService) TopReferers(ctx context.Context, userID uint64, limit, days int) ([]*model.HostStat, error) {
	if s.referers != nil {
		return s.referers.TopHosts(ctx, userID, limit, days)
	}
	list, err := s.repos.Referer.TopHosts(ctx, userID, model.Now().AddDate(0, 0, -days), limit)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return list, nil
}

// RefererTrend 返回用户引用趋势。
func (s *StatsService) RefererTrend(ctx context.Context, userID uint64, days int) ([]*model.DayStat, error) {
	if s.referers != nil {
		return s.referers.Trend(ctx, userID, days)
	}
	now := model.Now()
	list, err := s.repos.Referer.Trend(ctx, userID, now.AddDate(0, 0, -(days-1)), now)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return list, nil
}
