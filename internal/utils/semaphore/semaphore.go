package semaphore

import "sync"

type Semaphore struct {
	semaCh chan struct{}
	m      *sync.RWMutex
}

func New(maxRequestCount int) *Semaphore {
	return &Semaphore{
		semaCh: make(chan struct{}, maxRequestCount),
	}
}

func (s *Semaphore) Acquire() {
	s.m.RLock()
	defer s.m.RUnlock()

	s.semaCh <- struct{}{}
}

func (s *Semaphore) Release() {
	s.m.RLock()
	defer s.m.RUnlock()

	<-s.semaCh
}

func (s *Semaphore) ChangeMaxRequestsCount(rmpCurr, rmpNew uint64) {
	s.m.Lock()
	defer s.m.Unlock()

	newMaxRequestCount := rmpCurr / rmpNew
	s.semaCh = make(chan struct{}, newMaxRequestCount)
}
