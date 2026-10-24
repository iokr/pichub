package queue

import (
	"context"
	"sync"
	"time"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/logger"
)

// Worker 是任务消费协程池。
type Worker struct {
	q        *RedisQueue
	registry *Registry
	cfg      config.QueueConfig
	log      *logger.Logger

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// NewWorker 创建工作协程池。
func NewWorker(q *RedisQueue, registry *Registry, cfg config.QueueConfig, log *logger.Logger) *Worker {
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 2
	}
	if cfg.MaxRetry <= 0 {
		cfg.MaxRetry = 3
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = 30 * time.Second
	}
	if cfg.TaskTimeout <= 0 {
		cfg.TaskTimeout = 5 * time.Minute
	}
	return &Worker{q: q, registry: registry, cfg: cfg, log: log}
}

// Start 启动任务消费与延迟任务提升协程。
func (w *Worker) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	w.cancel = cancel

	for i := 0; i < w.cfg.WorkerCount; i++ {
		w.wg.Add(1)
		go func(id int) {
			defer w.wg.Done()
			w.loop(ctx, id)
		}(i)
	}
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.promoteLoop(ctx)
	}()
	if w.log != nil {
		w.log.Info("queue workers started", logger.Int("workers", w.cfg.WorkerCount))
	}
}

// Stop 停止所有协程并等待退出。
func (w *Worker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
}

func (w *Worker) loop(ctx context.Context, id int) {
	for {
		if ctx.Err() != nil {
			return
		}
		task, err := w.q.Reserve(ctx, 5*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if w.log != nil {
				w.log.Warn("reserve task failed", logger.Err(err), logger.Int("worker", id))
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}
		if task == nil {
			continue
		}
		w.handle(ctx, task)
	}
}

func (w *Worker) handle(ctx context.Context, task *Task) {
	h, ok := w.registry.Handler(task.Type)
	if !ok {
		if w.log != nil {
			w.log.Warn("no handler for task", logger.String("type", task.Type))
		}
		_ = w.q.Dead(ctx, task, "no handler registered")
		return
	}
	tctx, cancel := context.WithTimeout(ctx, w.cfg.TaskTimeout)
	err := h(tctx, task)
	cancel()
	if err == nil {
		return
	}
	if w.log != nil {
		w.log.Warn("task failed",
			logger.String("type", task.Type),
			logger.String("id", task.ID),
			logger.Int("attempt", task.Attempt+1),
			logger.Err(err))
	}
	if task.Attempt+1 >= w.cfg.MaxRetry {
		_ = w.q.Dead(ctx, task, err.Error())
		return
	}
	task.Attempt++
	if rerr := w.q.Retry(ctx, task, w.cfg.RetryDelay); rerr != nil && w.log != nil {
		w.log.Error("requeue task failed", logger.Err(rerr), logger.String("id", task.ID))
	}
}

func (w *Worker) promoteLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.q.PromoteDelayed(ctx); err != nil && ctx.Err() == nil && w.log != nil {
				w.log.Warn("promote delayed tasks failed", logger.Err(err))
			}
		}
	}
}
