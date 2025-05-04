package requestwatcher

import (
	"math"
	"sync"
	"time"
)

type RequestWatcher struct {
	requestsCh   <-chan struct{}
	stop         chan struct{}
	stopOnce     sync.Once
	requestCount uint64
	startTime    time.Time
	stopTime     time.Time
}

func New(requestsCh <-chan struct{}) *RequestWatcher {
	return &RequestWatcher{
		requestsCh: requestsCh,
	}
}

func (w *RequestWatcher) Start() {
	w.startTime = time.Now()
	w.requestCount = 0
	w.stopOnce = sync.Once{}
	w.stop = make(chan struct{})

	go func() {
		for {
			select {
			case <-w.stop:
				w.stopTime = time.Now()
				return
			case _, ok := <-w.requestsCh:
				if !ok {
					return
				}
				w.requestCount++
			}
		}
	}()
}

func (w *RequestWatcher) Stop() {
	w.stopOnce.Do(func() {
		close(w.stop)
	})
}

func (w *RequestWatcher) GetRPM() uint64 {
	interval := w.stopTime.Sub(w.startTime).Minutes()
	const tolerance = 0.001
	if math.Abs(interval-0.0) < tolerance {
		// TODO: log.Error()
		return uint64(float64(w.requestCount) / tolerance)
	}
	rpmCurr := float64(w.requestCount) / interval
	return uint64(rpmCurr) // округляем в меньшую сторону -> не переборщим с количеством запросов
}
