package requestwatcher

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"
)

type RequestWatcher struct {
	stop         chan struct{}
	requestsCh   <-chan struct{}
	log          *slog.Logger
	startTime    time.Time
	stopTime     time.Time
	requestCount uint64
	stopOnce     sync.Once
}

func New(requestsCh <-chan struct{}, log *slog.Logger) *RequestWatcher {
	if log == nil {
		log = slog.Default()
	}
	return &RequestWatcher{
		requestsCh: requestsCh,
		log:        log.With("module", "request_watcher"),
	}
}

func (w *RequestWatcher) Start() {
	w.log.LogAttrs(
		context.Background(),
		slog.LevelInfo,
		"starting request watcher timer")

	w.startTime = time.Now()
	w.requestCount = 0
	w.stopOnce = sync.Once{}
	w.stop = make(chan struct{})

	go func() {
		for {
			select {
			case <-w.stop:
				w.stopTime = time.Now()
				w.logFinish()
				return
			case _, ok := <-w.requestsCh:
				if !ok {
					w.logFinish()
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
		w.log.LogAttrs(
			context.Background(),
			slog.LevelWarn,
			"interval between errors is too small")
		return uint64(float64(w.requestCount) / tolerance)
	}
	rpmCurr := float64(w.requestCount) / interval
	return uint64(rpmCurr) // округляем в меньшую сторону -> не переборщим с количеством запросов
}

func (w *RequestWatcher) logFinish() {
	w.log.LogAttrs(
		context.Background(),
		slog.LevelInfo,
		"stoping request watcher timer")
}
