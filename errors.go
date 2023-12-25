package locker

import "errors"

var (
	// ErrLocked is returned when lock obtain fails because the lock is already held
	ErrLocked = errors.New("locker: locked")

	// ErrLockNotHeld is returned when lock release fails because the lock is already held by someone else or does not exist
	ErrLockNotHeld = errors.New("locker: lock not held")
)
