package serviceerrs

import (
	"errors"
	"strconv"
	"time"
)

var ErrSemaphoreTimeoutExceeded = errors.New(
	"semaphore acquire timeout exceeded")

var ErrSemaphoreAcquireTemporaryUnavailable = errors.New(
	"semaphore acquire temporary unavailable")

var ErrNoContent = errors.New("no content")

var ErrNotFound = errors.New("object not found")

var ErrTokenExpired = errors.New("token expired")

var ErrUnexpected = errors.New("unexpected server error")

var ErrInsufficientFunds = errors.New("insufficient funds")

type TooManyRequestsError struct {
	RetryAfter time.Duration
	RPM        uint64
}

func (e *TooManyRequestsError) Error() string {
	return "too many requests. Retry after " + e.RetryAfter.String() + ". " +
		"requested RPM: " + strconv.FormatUint(e.RPM, 10)
}
