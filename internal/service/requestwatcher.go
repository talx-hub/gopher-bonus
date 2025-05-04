package service

import (
	"math"
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
	interval := w.stopTime.Sub(w.startTime).Minutes()
	const tolerance = 0.001
	if math.Abs(interval-0.0) < tolerance {
		// TODO: log.Error()
		return uint64(float64(w.requestCount) / tolerance)
	}
	rpmCurr := float64(w.requestCount) / interval
	return uint64(rpmCurr) // округляем в меньшую сторону -> не переборщим с количеством запросов
}
