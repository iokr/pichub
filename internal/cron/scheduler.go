// Package cron 提供轻量级 5 字段 cron 表达式调度器。
package cron

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pichub/pichub/internal/logger"
)

// Job 是一个定时任务。
type Job struct {
	Name     string
	Schedule string
	Run      func(ctx context.Context) error
}

// matcher 表示一个 cron 字段的取值集合。
type matcher struct {
	bits uint64
	star bool
}

func (m matcher) has(v int) bool {
	if v < 0 || v >= 64 {
		return false
	}
	return m.bits&(1<<uint(v)) != 0
}

func (m *matcher) set(v int) { m.bits |= 1 << uint(v) }

// schedule 是解析后的表达式：分 时 日 月 周。
type schedule struct {
	minute matcher
	hour   matcher
	dom    matcher
	month  matcher
	dow    matcher
}

// parseField 解析单个 cron 字段，支持 *、a、a-b、*/n、a-b/n 与逗号列表。
func parseField(expr string, min, max int) (matcher, error) {
	var m matcher
	if expr == "*" {
		m.star = true
		for i := min; i <= max; i++ {
			m.set(i)
		}
		return m, nil
	}
	for _, part := range strings.Split(expr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return m, fmt.Errorf("字段 %q 含空片段", expr)
		}
		step := 1
		rng := part
		if idx := strings.Index(part, "/"); idx >= 0 {
			rng = strings.TrimSpace(part[:idx])
			n, err := strconv.Atoi(strings.TrimSpace(part[idx+1:]))
			if err != nil || n <= 0 {
				return m, fmt.Errorf("步长无效: %q", part)
			}
			step = n
		}
		var lo, hi int
		switch {
		case rng == "*":
			lo, hi = min, max
		case strings.Contains(rng, "-"):
			segs := strings.SplitN(rng, "-", 2)
			a, err1 := strconv.Atoi(strings.TrimSpace(segs[0]))
			b, err2 := strconv.Atoi(strings.TrimSpace(segs[1]))
			if err1 != nil || err2 != nil {
				return m, fmt.Errorf("范围无效: %q", part)
			}
			lo, hi = a, b
		default:
			v, err := strconv.Atoi(rng)
			if err != nil {
				return m, fmt.Errorf("取值无效: %q", part)
			}
			lo = v
			hi = v
			if step > 1 {
				hi = max
			}
		}
		if lo < min || hi > max || lo > hi {
			return m, fmt.Errorf("取值超出范围 %d-%d: %q", min, max, part)
		}
		for i := lo; i <= hi; i += step {
			m.set(i)
		}
	}
	return m, nil
}

// parseSchedule 解析 5 字段 cron 表达式。
func parseSchedule(expr string) (*schedule, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("定时表达式需要 5 个字段，实际 %d 个: %q", len(fields), expr)
	}
	minute, err := parseField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("分钟字段: %w", err)
	}
	hour, err := parseField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("小时字段: %w", err)
	}
	dom, err := parseField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("日期字段: %w", err)
	}
	month, err := parseField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("月份字段: %w", err)
	}
	dow, err := parseField(fields[4], 0, 7)
	if err != nil {
		return nil, fmt.Errorf("星期字段: %w", err)
	}
	// 0 与 7 均表示周日。
	if dow.has(7) {
		dow.set(0)
	}
	return &schedule{minute: minute, hour: hour, dom: dom, month: month, dow: dow}, nil
}

// match 判断给定时刻（精确到分钟）是否命中表达式。
func (s *schedule) match(t time.Time) bool {
	if !s.minute.has(t.Minute()) || !s.hour.has(t.Hour()) || !s.month.has(int(t.Month())) {
		return false
	}
	domMatch := s.dom.has(t.Day())
	dowMatch := s.dow.has(int(t.Weekday()))
	switch {
	case s.dom.star && s.dow.star:
		return true
	case s.dom.star:
		return dowMatch
	case s.dow.star:
		return domMatch
	default:
		// 与 Vixie cron 一致：日与星期都受限时取并集。
		return domMatch || dowMatch
	}
}

type entry struct {
	job   Job
	sched *schedule
	last  time.Time
}

// Scheduler 按 cron 表达式在后台执行任务。
type Scheduler struct {
	log *logger.Logger

	mu      sync.Mutex
	entries []*entry
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

func newScheduler(log *logger.Logger) *Scheduler { return &Scheduler{log: log} }

// Add 注册任务，表达式非法时返回错误。
func (s *Scheduler) Add(name, expr string, fn func(ctx context.Context) error) error {
	if fn == nil {
		return fmt.Errorf("job %s: 回调为空", name)
	}
	sch, err := parseSchedule(expr)
	if err != nil {
		return fmt.Errorf("job %s: %w", name, err)
	}
	s.mu.Lock()
	s.entries = append(s.entries, &entry{
		job:   Job{Name: name, Schedule: expr, Run: fn},
		sched: sch,
	})
	s.mu.Unlock()
	return nil
}

// Start 启动调度循环。
func (s *Scheduler) Start(parent context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		// 启动时先检查一次，避免服务重启错过整点任务。
		s.tick(ctx, time.Now())
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.tick(ctx, time.Now())
			}
		}
	}()
	if s.log != nil {
		s.mu.Lock()
		n := len(s.entries)
		s.mu.Unlock()
		s.log.Info("cron scheduler started", logger.Int("jobs", n))
	}
}

func (s *Scheduler) tick(ctx context.Context, now time.Time) {
	minute := now.Truncate(time.Minute)
	s.mu.Lock()
	due := make([]Job, 0, len(s.entries))
	for _, e := range s.entries {
		if e.last.Equal(minute) || !e.sched.match(minute) {
			continue
		}
		e.last = minute
		due = append(due, e.job)
	}
	s.mu.Unlock()

	for _, job := range due {
		job := job
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			start := time.Now()
			err := job.Run(ctx)
			if err != nil {
				if s.log != nil && ctx.Err() == nil {
					s.log.Warn("cron job failed",
						logger.String("job", job.Name),
						logger.Duration("cost", time.Since(start)),
						logger.Err(err))
				}
				return
			}
			if s.log != nil {
				s.log.Info("cron job done",
					logger.String("job", job.Name),
					logger.Duration("cost", time.Since(start)))
			}
		}()
	}
}

// Stop 停止调度并等待运行中的任务结束。
func (s *Scheduler) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.started = false
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.wg.Wait()
}
