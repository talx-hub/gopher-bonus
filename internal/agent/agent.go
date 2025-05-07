package agent

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/talx-hub/gopher-bonus/internal/agent/internal/httpclient"
	"github.com/talx-hub/gopher-bonus/internal/agent/internal/requestwatcher"
	"github.com/talx-hub/gopher-bonus/internal/agent/internal/workerpool"
	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
	"github.com/talx-hub/gopher-bonus/internal/utils/semaphore"
)

type Agent struct {
	ordersCh       chan uint64
	responsesCh    chan<- model.DTOAccrualInfo
	accrualAddress string
}

func New(
	ordersCh chan uint64,
	responsesCh chan<- model.DTOAccrualInfo,
	accrualAddress string,
) *Agent {
	return &Agent{
		accrualAddress: accrualAddress,
		ordersCh:       ordersCh,
		responsesCh:    responsesCh,
	}
}

func (a *Agent) Run(ctx context.Context, maxRequestCount uint64) {
	requestsCh := make(chan struct{}, runtime.NumCPU()*model.DefaultWorkerCountMultiplier)
	watcher := requestwatcher.New(requestsCh)
	watcher.Start()

	wg := &sync.WaitGroup{}
	rateDataCh := make(chan serviceerrs.TooManyRequestsError)
	pool := workerpool.New(
		httpclient.New(a.accrualAddress),
		semaphore.New(maxRequestCount),
		wg,
		a.ordersCh,
		rateDataCh,
		requestsCh,
		a.responsesCh,
	)
	poolCancel := pool.Start(ctx)

	timer := time.NewTimer(model.DefaultTimeout)
	timer.Stop()
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	rateData := serviceerrs.TooManyRequestsError{}
	for {
		select {
		case <-ctx.Done():
			poolCancel()
			wg.Wait()
			close(requestsCh)
			close(rateDataCh)
			close(a.responsesCh)
			return
		case rateData = <-rateDataCh:
			watcher.Stop()
			timer = time.NewTimer(rateData.RetryAfter)
		case <-timer.C:
			currRPM := watcher.GetRPM()
			newMaxRequestCount := maxRequestCount
			if currRPM != 0 {
				newMaxRequestCount = rateData.RPM / currRPM
			}

			watcher.Start()
			pool.ChangeMaxRequests(newMaxRequestCount)
			poolCancel = pool.Start(ctx)
		}
	}
}
