package semaphore

import (
	"sync"
	"time"

	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
)

type Semaphore struct {
	semaCh       chan struct{}
	cond         *sync.Cond
	m            sync.Mutex
	activeCount  uint64
	blockAcquire bool
}

func New(maxRequestCount uint64) *Semaphore {
	s := &Semaphore{
		semaCh: make(chan struct{}, maxRequestCount),
	}
	s.cond = sync.NewCond(&s.m)
	return s
}

func (s *Semaphore) AcquireWithTimeout(timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	s.m.Lock()
	if s.blockAcquire {
		// TODO: log
		s.m.Unlock()
		return serviceerrs.ErrSemaphoreAcquireTemporaryUnavailable
	}
	s.m.Unlock()

	select {
	case <-timer.C:
		// TODO: log
		return serviceerrs.ErrSemaphoreTimeoutExceeded
	case s.semaCh <- struct{}{}:
		// TODO: log.Trace!
		return nil
	}
}

func (s *Semaphore) Release() {
	// TODO: log.Trace!
	s.m.Lock()
	defer s.m.Unlock()

	<-s.semaCh
	s.activeCount--
	if s.activeCount == 0 {
		s.cond.Broadcast()
	}
}

func (s *Semaphore) ChangeMaxRequests(newMaxRequests uint64) {
	s.m.Lock()
	defer s.m.Unlock()

	s.blockAcquire = true
	for s.activeCount > 0 {
		s.cond.Wait()
	}
	s.semaCh = make(chan struct{}, newMaxRequests)
	s.blockAcquire = false
}
