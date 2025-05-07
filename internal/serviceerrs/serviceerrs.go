package serviceerrs

import (
	"errors"
	"time"
)

var ErrSemaphoreTimeoutExceeded = errors.New(
	"semaphore acquire timeout exceeded")

var ErrSemaphoreAcquireTemporaryUnavailable = errors.New(
	"semaphore acquire temporary unavailable")

type TooManyRequestsError struct {
	RetryAfter time.Duration
	RPM        uint64
}

func (e *TooManyRequestsError) Error() string {
	return "too many requests"
}
