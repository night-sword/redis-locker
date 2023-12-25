package locker

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

var addr = os.Getenv("ADDR")
var password = os.Getenv("PASSWORD")
var db, _ = strconv.Atoi(os.Getenv("DB"))

func TestLocker(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	opts := GetDefaultOptions()
	opts.Prefix = "test"
	SetDefaultOptions(opts)
	lockKey := "lock"
	lockKeyWithPrefix := opts.Prefix + ":" + lockKey
	ctx := context.Background()

	t.Run("An error should be returned when the lock is occupied", func(t *testing.T) {
		locker := NewLocker(client)
		defer client.Del(ctx, lockKeyWithPrefix)
		_, _ = client.Set(ctx, lockKeyWithPrefix, "1", time.Minute).Result()
		lock, err := locker.Obtain(ctx, lockKey)
		if !errors.Is(err, ErrLocked) {
			t.Errorf("expected error to be %v but got %v", ErrLocked, err)
		}
		if lock != nil {
			t.Errorf("expected lock to be nil but got %v", lock)
		}
	})

	t.Run("Can't release someone else's lock", func(t *testing.T) {
		locker := NewLocker(client)
		defer client.Del(ctx, lockKeyWithPrefix)
		lock, err := locker.Obtain(ctx, lockKey)
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}
		_, _ = client.Set(ctx, lockKeyWithPrefix, "1", time.Minute).Result()
		err = lock.Release()
		if !errors.Is(err, ErrLockNotHeld) {
			t.Errorf("expected error to be %v but got %v", ErrLockNotHeld, err)
		}
		lock.SilentRelease()
	})

	t.Run("Cannot release the lock repeatedly", func(t *testing.T) {
		locker := NewLocker(client)
		defer client.Del(ctx, lockKeyWithPrefix)
		lock, err := locker.Obtain(ctx, lockKey)
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}
		err = lock.Release()
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}
		err = lock.Release()
		if !errors.Is(err, ErrLockNotHeld) {
			t.Errorf("expected error to be %v but got %v", ErrLockNotHeld, err)
		}
	})

	t.Run("Can open and close auto-renewal", func(t *testing.T) {
		locker := NewLocker(client)
		defer client.Del(ctx, lockKeyWithPrefix)
		lock, err := locker.Obtain(ctx, lockKey, WithTTL(2*time.Second))
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}
		time.Sleep(2 * time.Second)
		err = lock.Release()
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}
		lock, err = locker.Obtain(ctx, lockKey, WithAutoRefresh(false), WithTTL(2*time.Second))
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}
		time.Sleep(3 * time.Second)
		err = lock.Release()
		if !errors.Is(err, ErrLockNotHeld) {
			t.Errorf("expected error to be %v but got %v", ErrLockNotHeld, err)
		}
	})

	t.Run("Retry is successful", func(t *testing.T) {
		locker := NewLocker(client)
		defer client.Del(ctx, lockKeyWithPrefix)
		lock, err := locker.Obtain(ctx, lockKey, WithTTL(20*time.Millisecond))
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}
		defer lock.SilentRelease()
		_, err = locker.Obtain(ctx, lockKey, WithRetryStrategy(linearBackoff(20*time.Millisecond)), WithMaxRetryTimes(3))
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}
	})

	t.Run("Retry failed", func(t *testing.T) {
		locker := NewLocker(client)
		defer client.Del(ctx, lockKeyWithPrefix)
		lock, err := locker.Obtain(ctx, lockKey, WithTTL(time.Second))
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}
		defer lock.SilentRelease()
		_, err = locker.Obtain(ctx, lockKey, WithRetryStrategy(linearBackoff(5*time.Millisecond)), WithMaxRetryTimes(2))
		if !errors.Is(err, ErrLocked) {
			t.Errorf("expected error to be %v but got %v", ErrLocked, err)
		}
	})

	t.Run("The lock should be released after execution", func(t *testing.T) {
		locker := NewLocker(client)
		defer client.Del(ctx, lockKeyWithPrefix)
		for i := 0; i < 5; i++ {
			err := locker.Do(ctx, lockKey, func(ctx context.Context) error {
				return nil
			})
			if err != nil {
				t.Errorf("expected no error but got %v", err)
			}
		}
	})

	t.Run("Locks should be released after a panic", func(t *testing.T) {
		locker := NewLocker(client)
		defer client.Del(ctx, lockKeyWithPrefix)
		defer func() {
			if r := recover(); r != nil {
				err := locker.Do(ctx, lockKey, func(ctx context.Context) error {
					return nil
				})
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		}()
		locker.Do(ctx, lockKey, func(ctx context.Context) error {
			panic("err")
		})
	})

	t.Run("An error should be returned if the lock cannot be obtained", func(t *testing.T) {
		locker := NewLocker(client)
		defer client.Del(ctx, lockKeyWithPrefix)
		lock, err := locker.Obtain(ctx, lockKey)
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}
		err = locker.Do(ctx, lockKey, func(ctx context.Context) error {
			panic("")
		})
		if !errors.Is(err, ErrLocked) {
			t.Errorf("expected error to be %v but got %v", ErrLocked, err)
		}
		lock.SilentRelease()
	})
}
