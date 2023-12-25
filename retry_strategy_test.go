package locker

import (
	"testing"
	"time"
)

func TestRetryStrategy(t *testing.T) {
	t.Run("Fixed Interval Retry Strategy", func(t *testing.T) {
		retry := LinearBackoff(time.Second)
		for _, want := range []time.Duration{
			time.Second,
			time.Second,
			time.Second,
		} {
			got := retry.NextBackoff()
			if got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		}
	})

	t.Run("Exponential Interval Retry Strategy", func(t *testing.T) {
		retry := ExponentialBackoff(10*time.Millisecond, 300*time.Millisecond)
		for _, want := range []time.Duration{
			10 * time.Millisecond,
			10 * time.Millisecond,
			16 * time.Millisecond,
			32 * time.Millisecond,
			64 * time.Millisecond,
			128 * time.Millisecond,
			256 * time.Millisecond,
			300 * time.Millisecond,
			300 * time.Millisecond,
		} {
			got := retry.NextBackoff()
			if got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		}
	})
}
