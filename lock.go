package locker

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Lock struct {
	redis         *redis.Client
	key           string
	value         string
	refreshTimes  int
	cancelRefresh func()
	opts          *Options
}

// Refresh lock time
func (inst *Lock) Refresh(ctx context.Context) error {
	ttlVal := strconv.FormatInt(int64(inst.opts.TTL/time.Millisecond), 10)
	status, err := inst.redis.Eval(ctx, luaRefresh, []string{inst.key}, inst.value, ttlVal).Result()
	if err != nil {
		return err
	} else if status == int64(1) {
		return nil
	}
	return ErrLocked
}

func (inst *Lock) Release() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	inst.cancelRefresh()
	res, err := inst.redis.Eval(ctx, luaRelease, []string{inst.key}, inst.value).Result()
	if errors.Is(err, redis.Nil) {
		return ErrLockNotHeld
	} else if err != nil {
		return err
	}

	if i, ok := res.(int64); !ok || i != 1 {
		return ErrLockNotHeld
	}
	return nil
}

func (inst *Lock) Key() string {
	return inst.key
}

func (inst *Lock) SilentRelease() {
	err := inst.Release()
	if err != nil {
		inst.opts.Logger.Warn("locker: release lock failed", "key", inst.key, "error", err.Error())
	}
}
