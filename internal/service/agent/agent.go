package agent

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/service/agent/internal/httpclient"
	"github.com/talx-hub/gopher-bonus/internal/service/agent/internal/requestwatcher"
	"github.com/talx-hub/gopher-bonus/internal/service/agent/internal/workerpool"
	"github.com/talx-hub/gopher-bonus/internal/service/dto"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
	"github.com/talx-hub/gopher-bonus/internal/utils/logger"
	"github.com/talx-hub/gopher-bonus/internal/utils/semaphore"
)

type Agent struct {
	ordersCh       chan string
	responsesCh    chan<- dto.AccrualInfo
	accrualAddress string
	workerCount    int
}

func New(
	ordersCh chan string,
	responsesCh chan<- dto.AccrualInfo,
	accrualAddress string,
) *Agent {
	return &Agent{
		accrualAddress: accrualAddress,
		ordersCh:       ordersCh,
		responsesCh:    responsesCh,
		workerCount:    runtime.NumCPU() * model.DefaultWorkerCountMultiplier,
	}
}

func (a *Agent) Run(ctx context.Context, maxRequestCount uint64) {
	log := logger.FromContext(ctx).With("service", "agent")
	log.LogAttrs(ctx, slog.LevelInfo, "running")

	requestsCh := make(chan struct{}, runtime.NumCPU()*model.DefaultWorkerCountMultiplier)
	rpmWatcher := requestwatcher.New(requestsCh, log)
	rpmWatcher.Start()

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
	log.LogAttrs(ctx, slog.LevelInfo, "starting worker pool")
	poolCancel := pool.Start(ctx, a.workerCount)

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
			log.LogAttrs(ctx, slog.LevelInfo, "stopped")
			return
		case rateData = <-rateDataCh:
			wg.Wait()
			rpmWatcher.Stop()
			timer = time.NewTimer(rateData.RetryAfter)
			log.LogAttrs(ctx,
				slog.LevelInfo,
				"paused requesting",
				slog.Duration("retry_after", rateData.RetryAfter))
		case <-timer.C:
			currRPM := rpmWatcher.GetRPM()
			newMaxRequestCount := maxRequestCount
			if currRPM != 0 {
				newMaxRequestCount = rateData.RPM / currRPM
			}

			rpmWatcher.Start()
			pool.ChangeMaxRequests(newMaxRequestCount)
			poolCancel = pool.Start(ctx, a.workerCount)
			log.LogAttrs(ctx,
				slog.LevelInfo,
				"restarted requesting",
				slog.Int("old_rpm", int(maxRequestCount)),
				slog.Int("new_rpm", int(newMaxRequestCount)))
		}
	}
}
