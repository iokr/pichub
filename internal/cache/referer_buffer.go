package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pichub/pichub/internal/logger"
)

// RefererEvent 是一次图片访问产生的引用事件。
type RefererEvent struct {
	ImageID     uint64
	RefererHash string
	RefererURL  string
	RefererHost string
}

// RefererItem 是从 Redis 缓冲读取出的聚合条目。
type RefererItem struct {
	ImageID     uint64
	RefererHash string
	RefererURL  string
	RefererHost string
	Hits        int64
}

// RefererBuffer 使用带缓冲 channel + 单 goroutine 异步写入 Redis，
// 保证图片直出永不阻塞；channel 满时丢弃并计数。
type RefererBuffer struct {
	rdb     *redis.Client
	log     *logger.Logger
	prefix  string
	ch      chan RefererEvent
	done    chan struct{}
	wg      sync.WaitGroup
	dropped atomic.Int64
	started atomic.Bool
}

// NewRefererBuffer 创建引用统计缓冲。
func NewRefererBuffer(rdb *redis.Client, keyPrefix string, size int, log *logger.Logger) *RefererBuffer {
	if size <= 0 {
		size = 10000
	}
	return &RefererBuffer{
		rdb:    rdb,
		log:    log,
		prefix: keyPrefix + "ref:",
		ch:     make(chan RefererEvent, size),
		done:   make(chan struct{}),
	}
}

// Start 启动后台消费协程。
func (b *RefererBuffer) Start(ctx context.Context) {
	if !b.started.CompareAndSwap(false, true) {
		return
	}
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		for {
			select {
			case <-b.done:
				b.drain()
				return
			case ev := <-b.ch:
				b.write(ctx, ev)
			}
		}
	}()
}

func (b *RefererBuffer) drain() {
	for {
		select {
		case ev := <-b.ch:
			b.write(context.Background(), ev)
		default:
			return
		}
	}
}

// Push 非阻塞写入事件，缓冲满时丢弃。
func (b *RefererBuffer) Push(ev RefererEvent) {
	select {
	case b.ch <- ev:
	default:
		b.dropped.Add(1)
	}
}

// Stop 停止消费并冲刷剩余事件。
func (b *RefererBuffer) Stop() {
	select {
	case <-b.done:
	default:
		close(b.done)
	}
	b.wg.Wait()
}

// Dropped 返回被丢弃的事件数。
func (b *RefererBuffer) Dropped() int64 { return b.dropped.Load() }

func (b *RefererBuffer) write(ctx context.Context, ev RefererEvent) {
	date := time.Now().UTC().Format("20060102")
	field := fmt.Sprintf("%d|%s", ev.ImageID, ev.RefererHash)
	pipe := b.rdb.Pipeline()
	pipe.HIncrBy(ctx, b.prefix+"buf:"+date, field, 1)
	pipe.Set(ctx, b.prefix+"meta:"+ev.RefererHash,
		ev.RefererURL+"\x00"+ev.RefererHost, 48*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil && b.log != nil {
		b.log.Warn("referer buffer write failed", logger.Err(err))
	}
}

// snapshotScript 原子读取并删除当日缓冲，避免并发丢计数。
var snapshotScript = redis.NewScript(`
local vals = redis.call('HGETALL', KEYS[1])
redis.call('DEL', KEYS[1])
return vals
`)

// Snapshot 读取并清空指定日期的缓冲（默认今天）。
func (b *RefererBuffer) Snapshot(ctx context.Context, date string) ([]RefererItem, error) {
	if date == "" {
		date = time.Now().UTC().Format("20060102")
	}
	res, err := snapshotScript.Run(ctx, b.rdb, []string{b.prefix + "buf:" + date}).Result()
	if err != nil {
		return nil, err
	}
	values, ok := res.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected snapshot result %T", res)
	}
	items := make([]RefererItem, 0, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		field, _ := values[i].(string)
		hits, _ := strconv.ParseInt(fmt.Sprint(values[i+1]), 10, 64)
		parts := strings.SplitN(field, "|", 2)
		if len(parts) != 2 {
			continue
		}
		id, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			continue
		}
		item := RefererItem{ImageID: id, RefererHash: parts[1], Hits: hits}
		if meta, err := b.rdb.Get(ctx, b.prefix+"meta:"+parts[1]).Result(); err == nil {
			segs := strings.SplitN(meta, "\x00", 2)
			item.RefererURL = segs[0]
			if len(segs) > 1 {
				item.RefererHost = segs[1]
			}
		}
		items = append(items, item)
	}
	return items, nil
}
