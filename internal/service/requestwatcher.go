package service

import (
	"time"
)

type requestWatcher struct {
	requestsCh   <-chan struct{}
	stop         chan struct{}
	requestCount uint64
	startTime    time.Time
	stopTime     time.Time
}

func newRequestWatcher(requestsCh <-chan struct{}) *requestWatcher {
	return &requestWatcher{
		requestsCh: requestsCh,
	}
}

func (w *requestWatcher) Start() {
	w.startTime = time.Now()
	w.requestCount = 0

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

func (w *requestWatcher) Stop() {
	w.stop <- struct{}{}
}

func (w *requestWatcher) GetRPM() uint64 {
	interval := w.stopTime.Sub(w.startTime)
	rpmCurr := float64(w.requestCount) / interval.Minutes()
	return uint64(rpmCurr) // округляем в меньшую сторону -> не переборщим с количеством запросов
}
