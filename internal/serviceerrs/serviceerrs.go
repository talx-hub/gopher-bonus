package serviceerrs

import "time"

type TooManyRequestsError struct {
	TimeToSleep time.Duration
	RPM         uint64
}

func (e *TooManyRequestsError) Error() string {
	return "too many requests"
}
