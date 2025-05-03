package semaphore

import (
	"fmt"
)

type Semaphore struct {
	semaCh chan struct{}
}

func New(maxRequestCount uint64) *Semaphore {
	return &Semaphore{
		semaCh: make(chan struct{}, maxRequestCount),
	}
}

func (s *Semaphore) Acquire() {
	// TODO: log.Trace!
	fmt.Println("acquire", "worker count:", len(s.semaCh))
	s.semaCh <- struct{}{}
}

func (s *Semaphore) Release() {
	// TODO: log.Trace!
	fmt.Println("release", "worker count:", len(s.semaCh))
	<-s.semaCh
}
