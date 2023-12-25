package locker

import (
	"sync"
	"time"
)

var _optMutex = new(sync.RWMutex)

type Options struct {
	TTL                 time.Duration
	AutoRefresh         bool
	MaxAutoRefreshTimes int
	RetryStrategy       RetryStrategy
	MaxRetryTimes       int
	Prefix              string
	Logger              Logger
}

var defaultOptions = &Options{
	TTL:                 5 * time.Second,
	AutoRefresh:         true,
	MaxAutoRefreshTimes: 5,
	RetryStrategy:       NoRetry(),
	MaxRetryTimes:       3,
	Prefix:              "",
	Logger:              &defaultLogger{},
}

func SetDefaultOptions(options *Options) {
	opts := *options

	_optMutex.Lock()
	defer _optMutex.Unlock()

	defaultOptions = &opts
}

func GetDefaultOptions() (options *Options) {
	_optMutex.RLock()
	defer _optMutex.RUnlock()

	opts := *defaultOptions
	return &opts
}

type Option func(options *Options)

// WithTTL sets the lock expiration time, default is 5 seconds
func WithTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.TTL = ttl
	}
}

// WithAutoRefresh sets whether to automatically refresh the lock time, default is true
func WithAutoRefresh(autoRefresh bool) Option {
	return func(o *Options) {
		o.AutoRefresh = autoRefresh
	}
}

// WithMaxAutoRefreshTimes sets the maximum number of automatic refreshes, default is 5
func WithMaxAutoRefreshTimes(times int) Option {
	return func(o *Options) {
		o.MaxAutoRefreshTimes = times
	}
}

// WithRetryStrategy sets the retry strategy, default is no retry
func WithRetryStrategy(retryStrategy RetryStrategy) Option {
	return func(o *Options) {
		o.RetryStrategy = retryStrategy
	}
}

// WithMaxRetryTimes sets the maximum number of retrials, default is 3
func WithMaxRetryTimes(times int) Option {
	return func(o *Options) {
		o.MaxRetryTimes = times
	}
}

// WithMaxRetryTimes sets the maximum number of retrials, default is 3
func WithPrefix(prefix string) Option {
	return func(o *Options) {
		o.Prefix = prefix
	}
}

func newDefaultOptions() *Options {
	return GetDefaultOptions()
}
