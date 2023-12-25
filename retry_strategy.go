package locker

import (
	"sync/atomic"
	"time"
)

// RetryStrategy allows customizing the lock retry strategy
type RetryStrategy interface {
	// NextBackoff returns the next retry time
	NextBackoff() time.Duration
}

type linearBackoff time.Duration

// LinearBackoff fixed interval retry strategy
func LinearBackoff(backoff time.Duration) RetryStrategy {
	return linearBackoff(backoff)
}

func (r linearBackoff) NextBackoff() time.Duration {
	return time.Duration(r)
}

// NoRetry means do not retry
func NoRetry() RetryStrategy {
	return linearBackoff(-1)
}

type exponentialBackoff struct {
	cnt uint64
	min time.Duration
	max time.Duration
}

// ExponentialBackoff exponential interval retry strategy, retry time is 2**n milliseconds (n refers to the number of times).
// You can set minimum and maximum values, the maximum value can only be below 139 minutes (i.e., 2 to the 23rd power)
func ExponentialBackoff(min, max time.Duration) RetryStrategy {
	return &exponentialBackoff{min: min, max: max}
}

func (r *exponentialBackoff) NextBackoff() time.Duration {
	cnt := atomic.AddUint64(&r.cnt, 1)
	ms := 2 << 22
	if cnt < 22 {
		ms = 2 << cnt
	}
	if d := time.Duration(ms) * time.Millisecond; d < r.min {
		return r.min
	} else if r.max != 0 && d > r.max {
		return r.max
	} else {
		return d
	}
}
