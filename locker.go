package locker

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	luaRefresh = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("pexpire", KEYS[1], ARGV[2]) else return 0 end`
	luaRelease = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
)

type Locker struct {
	redis *redis.Client
	src   rand.Source
	mutex *sync.Mutex
}

func NewLocker(redis *redis.Client) *Locker {
	return &Locker{
		redis: redis,
		src:   rand.NewSource(time.Now().UnixNano()),
		mutex: new(sync.Mutex),
	}
}

func (inst *Locker) Obtain(ctx context.Context, key string, opts ...Option) (*Lock, error) {
	o := newDefaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	value := inst.randomToken()
	retry := o.RetryStrategy

	if o.Prefix != "" {
		key = o.Prefix + ":" + key
	}

	var timer *time.Timer
	for retryTimes, maxRetryTimes := 0, o.MaxRetryTimes; retryTimes <= maxRetryTimes; retryTimes++ {
		if retryTimes > 0 {
			o.Logger.Info("locker: retry lock", "key", key, "times", retryTimes, "max_times", maxRetryTimes)
		}
		ok, err := inst.redis.SetNX(ctx, key, value, o.TTL).Result()
		if err != nil {
			return nil, err
		} else if ok {
			refreshCtx, cancel := context.WithCancel(ctx)
			var lock = &Lock{
				redis:         inst.redis,
				key:           key,
				value:         value,
				cancelRefresh: cancel,
				opts:          o,
			}
			if o.AutoRefresh && o.TTL > time.Second {
				go inst.autoRefresh(refreshCtx, lock)
			}

			return lock, nil
		}

		backoff := retry.NextBackoff()
		if backoff < 0 {
			return nil, ErrLocked
		}

		if timer == nil {
			timer = time.NewTimer(backoff)
		} else {
			timer.Reset(backoff)
		}

		<-timer.C
	}
	timer.Stop()

	return nil, ErrLocked
}

func (inst *Locker) Do(ctx context.Context, key string, fn func(ctx context.Context) error, opts ...Option) error {
	lock, err := inst.Obtain(ctx, key, opts...)
	if err != nil {
		return err
	}
	defer lock.SilentRelease()
	return fn(ctx)
}

func (inst *Locker) autoRefresh(ctx context.Context, lock *Lock) {
	ttl := lock.opts.TTL
	// Refresh interval, the interval must be before the lock expires
	interval := ttl - time.Second
	if interval < time.Second {
		lock.opts.Logger.Warn("locker: If the lock duration is too short, the lock renewal may fail", "key", lock.key, "ttl", ttl.Milliseconds())
	}

	// Does not automatically refresh the lock when the refresh interval is too short
	if interval <= (200 * time.Millisecond) {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if lock.refreshTimes >= lock.opts.MaxAutoRefreshTimes {
				return
			}
			lock.refreshTimes++

			err := lock.Refresh(ctx)
			if err != nil {
				if errors.Is(err, ErrLocked) {
					lock.opts.Logger.Warn("locker: refresh failed", "refresh_times", lock.refreshTimes, "error", err.Error())
				}

				return
			}
		}
	}
}

func (inst *Locker) randomToken() string {
	inst.mutex.Lock()
	defer inst.mutex.Unlock()
	return randomString(inst.src, 16)
}
