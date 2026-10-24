package cron

import (
	"context"
	"time"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/imageproc"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/repository"
	"github.com/pichub/pichub/internal/service"
)

// Deps 是定时任务所需的依赖。
type Deps struct {
	Images    *service.ImageService
	Referer   *service.RefererService
	Upload    *service.UploadService
	Processor *imageproc.Processor
	Emails    *repository.EmailRepo
	Logs      *repository.LogRepo
}

// New 按配置注册全部定时任务。
func New(cfg *config.Config, deps Deps, log *logger.Logger) *Scheduler {
	s := newScheduler(log)
	if !cfg.Cron.Enabled {
		if log != nil {
			log.Info("cron disabled by config")
		}
		return s
	}

	add := func(enabled bool, name, expr string, fn func(ctx context.Context) error) {
		if !enabled || fn == nil {
			return
		}
		if err := s.Add(name, expr, fn); err != nil {
			if log != nil {
				log.Warn("skip invalid cron job",
					logger.String("job", name),
					logger.String("schedule", expr),
					logger.Err(err))
			}
		}
	}

	// 引用计数每分钟刷盘一次，避免高频图片请求直接写库。
	add(cfg.Referer.Enabled, "referer_flush", "* * * * *", func(ctx context.Context) error {
		if deps.Referer == nil {
			return nil
		}
		n, err := deps.Referer.Flush(ctx)
		if err != nil {
			return err
		}
		if n > 0 && log != nil {
			log.Debug("referer stats flushed", logger.Int("rows", n))
		}
		return nil
	})

	// 每日清理过期引用聚合数据。
	add(cfg.Cron.RefererAggregate.Enabled, "referer_aggregate", cfg.Cron.RefererAggregate.Schedule, func(ctx context.Context) error {
		if deps.Referer == nil {
			return nil
		}
		return deps.Referer.CleanupDaily(ctx)
	})

	// 清理超过保留期的回收站图片。
	add(cfg.Cron.TrashCleanup.Enabled, "trash_cleanup", cfg.Cron.TrashCleanup.Schedule, func(ctx context.Context) error {
		if deps.Images == nil {
			return nil
		}
		days := cfg.Cron.TrashCleanup.RetainDays
		if days <= 0 {
			days = 30
		}
		cutoff := time.Now().UTC().AddDate(0, 0, -days)
		total := 0
		for {
			n, err := deps.Images.CleanupTrash(ctx, cutoff, 200)
			if err != nil {
				return err
			}
			total += n
			if n < 200 {
				break
			}
		}
		if log != nil {
			log.Info("trash cleanup finished", logger.Int("removed", total), logger.Int("retain_days", days))
		}
		return nil
	})

	// 清理图片处理缓存与过期的分片上传会话。
	add(cfg.Cron.CacheCleanup.Enabled, "cache_cleanup", cfg.Cron.CacheCleanup.Schedule, func(ctx context.Context) error {
		if deps.Processor != nil {
			if c := deps.Processor.Cache(); c != nil {
				removed, freed, err := c.Cleanup()
				if err != nil {
					return err
				}
				if log != nil {
					log.Info("image cache cleaned", logger.Int("removed", removed), logger.Int64("freed_bytes", freed))
				}
			}
		}
		if deps.Upload != nil {
			if n := deps.Upload.CleanupChunks(time.Now().Add(-24 * time.Hour)); n > 0 && log != nil {
				log.Info("stale upload sessions cleaned", logger.Int("removed", n))
			}
		}
		return nil
	})

	// 清理过期邮箱验证/重置令牌与邮件日志。
	add(cfg.Cron.EmailLogCleanup.Enabled, "email_log_cleanup", cfg.Cron.EmailLogCleanup.Schedule, func(ctx context.Context) error {
		if deps.Emails == nil {
			return nil
		}
		now := time.Now().UTC()
		if err := deps.Emails.DeleteExpiredTokens(ctx, now); err != nil {
			return err
		}
		days := cfg.Cron.EmailLogCleanup.RetainDays
		if days <= 0 {
			days = 30
		}
		return deps.Emails.DeleteLogsBefore(ctx, now.AddDate(0, 0, -days))
	})

	// 清理过期操作日志。
	add(cfg.Cron.OperationLogCleanup.Enabled, "operation_log_cleanup", cfg.Cron.OperationLogCleanup.Schedule, func(ctx context.Context) error {
		if deps.Logs == nil {
			return nil
		}
		days := cfg.Cron.OperationLogCleanup.RetainDays
		if days <= 0 {
			days = 90
		}
		return deps.Logs.DeleteBefore(ctx, time.Now().UTC().AddDate(0, 0, -days))
	})

	return s
}
