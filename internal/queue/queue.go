// Package queue 提供基于 Redis 的异步任务队列与工作协程池。
package queue

import (
	"context"
	"encoding/json"
	"sync"
)

// 任务类型。
const (
	TaskSendEmail = "email.send"
)

// Task 是队列中的一个任务。
type Task struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Attempt   int             `json:"attempt"`
	CreatedAt int64           `json:"created_at"`
}

// Handler 处理一类任务，返回错误时按策略重试。
type Handler func(ctx context.Context, task *Task) error

// Registry 是任务处理器注册表。
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewRegistry 创建空的处理器注册表。
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

// Register 注册任务处理器。
func (r *Registry) Register(taskType string, h Handler) {
	r.mu.Lock()
	r.handlers[taskType] = h
	r.mu.Unlock()
}

// Handler 查找任务处理器。
func (r *Registry) Handler(taskType string) (Handler, bool) {
	r.mu.RLock()
	h, ok := r.handlers[taskType]
	r.mu.RUnlock()
	return h, ok
}

// SendEmailPayload 是发送邮件任务的负载。
type SendEmailPayload struct {
	UserID        uint64 `json:"user_id"`
	Kind          string `json:"kind"`
	To            string `json:"to"`
	Username      string `json:"username"`
	URL           string `json:"url"`
	Email         string `json:"email,omitempty"`
	ExpireMinutes int    `json:"expire_minutes"`
}
