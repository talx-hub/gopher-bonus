package serviceerrs

import (
	"errors"
	"time"
)

var ErrSemaphoreTimeoutExceeded = errors.New(
	"semaphore acquire timeout exceeded")

type TooManyRequestsError struct {
	RetryAfter time.Duration
	RPM        uint64
}

func (e *TooManyRequestsError) Error() string {
	return "too many requests"
}
