package semaphore

import (
	"errors"
	"time"
)

type Semaphore struct {
	semaCh chan struct{}
}

func New(maxRequestCount uint64) *Semaphore {
	return &Semaphore{
		semaCh: make(chan struct{}, maxRequestCount),
	}
}

func (s *Semaphore) AcquireWithTimeout(timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		return errors.New("semaphore acquire timeout exceeded")
	case s.semaCh <- struct{}{}:
		// TODO: log.Trace!
		return nil
	}
}

func (s *Semaphore) Release() {
	// TODO: log.Trace!
	<-s.semaCh
}
