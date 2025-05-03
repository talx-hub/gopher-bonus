package service

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/talx-hub/gopher-bonus/internal/model"
)

type AccrualClient interface {
	GetOrderInfo(orderID string) (model.DTOAccrualInfo, error)
}

type Agent struct {
	client      AccrualClient
	ordersCh    chan uint64
	responsesCh chan<- model.DTOAccrualInfo
}

func New(
	ordersCh chan uint64,
	responsesCh chan<- model.DTOAccrualInfo,
	accrualAddress string,
) *Agent {
	return &Agent{
		client:      newCustomClient(accrualAddress),
		ordersCh:    ordersCh,
		responsesCh: responsesCh,
	}
}

type requestRateData struct {
	timeToSleep time.Duration
	rpm         uint64
}

func (a *Agent) Run(ctx context.Context, maxRequestCount uint64) {
	requestsCh := make(chan struct{}, runtime.NumCPU()*model.DefaultWorkerCountMultiplier)
	watcher := newRequestWatcher(requestsCh)
	watcher.Start()

	wg := &sync.WaitGroup{}
	rateDataCh := make(chan requestRateData)
	pool := NewWorkerPool(
		a.client,
		wg,
		a.ordersCh,
		rateDataCh,
		requestsCh,
		a.responsesCh,
	)
	poolCancel := pool.Start(ctx, maxRequestCount)

	rateData := requestRateData{}
	var timer *time.Timer
	for {
		select {
		case <-ctx.Done():
			poolCancel()
			wg.Wait()
			close(rateDataCh)
			close(a.responsesCh)
			close(rateDataCh)
			return
		case rateData = <-rateDataCh:
			watcher.Stop()
			timer = time.NewTimer(rateData.timeToSleep)
		case <-timer.C:
			currRPM := watcher.GetRPM()
			newMaxRequestCount := maxRequestCount
			if currRPM != 0 {
				newMaxRequestCount = rateData.rpm / currRPM
			}

			watcher.Start()
			poolCancel = pool.Start(ctx, newMaxRequestCount)
		}
	}
}
