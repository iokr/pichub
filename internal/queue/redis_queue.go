package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/pichub/pichub/internal/logger"
)

// Redis 键名（不含前缀）。
const (
	keyReady   = "queue:ready"
	keyDelayed = "queue:delayed"
	keyDead    = "queue:dead"
)

// promoteScript 原子地把到期的延迟任务搬回就绪队列。
var promoteScript = redis.NewScript(`
local items = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'LIMIT', 0, 200)
if #items == 0 then return 0 end
for i = 1, #items do
  redis.call('LPUSH', KEYS[2], items[i])
  redis.call('ZREM', KEYS[1], items[i])
end
return #items
`)

// RedisQueue 是基于 Redis list 与 zset 的任务队列。
type RedisQueue struct {
	rdb    *redis.Client
	prefix string
	log    *logger.Logger
}

// NewRedisQueue 创建 Redis 队列。
func NewRedisQueue(rdb *redis.Client, keyPrefix string, log *logger.Logger) *RedisQueue {
	return &RedisQueue{rdb: rdb, prefix: keyPrefix + "tasks:", log: log}
}

func (q *RedisQueue) key(name string) string { return q.prefix + name }

// Enqueue 把任务写入就绪队列。
func (q *RedisQueue) Enqueue(ctx context.Context, taskType string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal task payload: %w", err)
	}
	task := &Task{
		ID:        uuid.NewString(),
		Type:      taskType,
		Payload:   raw,
		CreatedAt: time.Now().Unix(),
	}
	return q.EnqueueRaw(ctx, task)
}

// EnqueueRaw 写入一个已构造好的任务。
func (q *RedisQueue) EnqueueRaw(ctx context.Context, task *Task) error {
	b, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	return q.rdb.LPush(ctx, q.key(keyReady), b).Err()
}

// Reserve 阻塞取出一个任务，超时返回 (nil, nil)。
func (q *RedisQueue) Reserve(ctx context.Context, timeout time.Duration) (*Task, error) {
	res, err := q.rdb.BRPop(ctx, timeout, q.key(keyReady)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	if len(res) < 2 {
		return nil, nil
	}
	var task Task
	if err := json.Unmarshal([]byte(res[1]), &task); err != nil {
		if q.log != nil {
			q.log.Warn("drop malformed task", logger.Err(err))
		}
		return nil, nil
	}
	return &task, nil
}

// Retry 把失败任务放入延迟集合，到期后重新入队。
func (q *RedisQueue) Retry(ctx context.Context, task *Task, delay time.Duration) error {
	b, err := json.Marshal(task)
	if err != nil {
		return err
	}
	score := float64(time.Now().Add(delay).UnixNano() / int64(time.Millisecond))
	return q.rdb.ZAdd(ctx, q.key(keyDelayed), redis.Z{Score: score, Member: b}).Err()
}

// Dead 把无法处理的任务写入死信队列，最多保留 1000 条。
func (q *RedisQueue) Dead(ctx context.Context, task *Task, reason string) error {
	entry := struct {
		*Task
		Reason   string `json:"reason"`
		FailedAt int64  `json:"failed_at"`
	}{Task: task, Reason: reason, FailedAt: time.Now().Unix()}
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	pipe := q.rdb.Pipeline()
	pipe.LPush(ctx, q.key(keyDead), b)
	pipe.LTrim(ctx, q.key(keyDead), 0, 999)
	_, err = pipe.Exec(ctx)
	return err
}

// PromoteDelayed 把已到期的延迟任务搬回就绪队列，返回搬运数量。
func (q *RedisQueue) PromoteDelayed(ctx context.Context) (int64, error) {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	n, err := promoteScript.Run(ctx, q.rdb, []string{q.key(keyDelayed), q.key(keyReady)}, now).Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, err
	}
	return n, nil
}

// Stats 返回队列统计信息。
func (q *RedisQueue) Stats(ctx context.Context) (ready, delayed, dead int64, err error) {
	pipe := q.rdb.Pipeline()
	readyCmd := pipe.LLen(ctx, q.key(keyReady))
	delayedCmd := pipe.ZCard(ctx, q.key(keyDelayed))
	deadCmd := pipe.LLen(ctx, q.key(keyDead))
	if _, err = pipe.Exec(ctx); err != nil {
		return 0, 0, 0, err
	}
	return readyCmd.Val(), delayedCmd.Val(), deadCmd.Val(), nil
}
