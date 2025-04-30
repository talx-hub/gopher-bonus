package semaphore

import (
	"fmt"
	"sync"
	"time"
)

type Semaphore struct {
	semaCh chan struct{}
	m      *sync.RWMutex
}

func New(maxRequestCount int) *Semaphore {
	return &Semaphore{
		semaCh: make(chan struct{}, maxRequestCount),
		m:      &sync.RWMutex{},
	}
}

func (s *Semaphore) Acquire() {
	s.m.RLock()
	defer s.m.RUnlock()

	s.semaCh <- struct{}{}
	// TODO: log.Trace!
	fmt.Println("acquire", "worker count:", len(s.semaCh))
}

func (s *Semaphore) Release() {
	s.m.RLock()
	defer s.m.RUnlock()

	// TODO: log.Trace!
	fmt.Println("release", "worker count:", len(s.semaCh))
	<-s.semaCh
}

func (s *Semaphore) ChangeMaxRequestsCount(requestCount uint64, startEpoch time.Time, rpmNew uint64) {
	s.m.Lock()
	defer s.m.Unlock()

	// TODO: log.Trace!
	fmt.Println("ChangeMaxRequestsCount")
	interval := time.Since(startEpoch)
	rpmCurr := float64(requestCount) / interval.Minutes()
	newMaxRequestCount := uint64(rpmCurr) / rpmNew // округляем в меньшую сторону -> не переборщим с количеством запросов
	s.semaCh = make(chan struct{}, newMaxRequestCount)
}
