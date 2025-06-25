package workerpool

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/service/dto"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
	"github.com/talx-hub/gopher-bonus/internal/utils/logger"
)

type AccrualClient interface {
	GetOrderInfo(ctx context.Context, orderID string) (dto.AccrualInfo, error)
}

type AccrualSemaphore interface {
	AcquireWithTimeout(timeout time.Duration) error
	ChangeMaxRequests(newMaxRequests uint64)
	Release()
}

type WorkerPool struct {
	Client         AccrualClient
	Sema           AccrualSemaphore
	WaitGroup      *sync.WaitGroup
	Jobs           <-chan string
	RateDataCh     chan<- serviceerrs.TooManyRequestsError
	RequestCounter chan<- struct{}
	Results        chan<- dto.AccrualInfo
	OnWorkerStart  func()
}

func New(
	client AccrualClient,
	sema AccrualSemaphore,
	wg *sync.WaitGroup,
	jobs <-chan string,
	rateDataCh chan<- serviceerrs.TooManyRequestsError,
	requestCounter chan<- struct{},
	results chan<- dto.AccrualInfo,
) *WorkerPool {
	return &WorkerPool{
		Client:         client,
		Sema:           sema,
		WaitGroup:      wg,
		Jobs:           jobs,
		RateDataCh:     rateDataCh,
		RequestCounter: requestCounter,
		Results:        results,
	}
}

func (pool *WorkerPool) Start(ctx context.Context, workerCount int) context.CancelFunc {
	workerCtx, workerCancel := context.WithCancel(ctx)
	for range workerCount {
		pool.WaitGroup.Add(1)
		go pool.worker(workerCtx, workerCancel)
	}
	log := logger.FromContext(workerCtx).With("module", "worker_pool")
	log.LogAttrs(ctx, slog.LevelInfo,
		"all workers started", slog.Int("count", workerCount))

	return workerCancel
}

func (pool *WorkerPool) ChangeMaxRequests(newMaxRequests uint64) {
	pool.Sema.ChangeMaxRequests(newMaxRequests)
}

func (pool *WorkerPool) worker(ctx context.Context, cancelAll context.CancelFunc) {
	if pool.OnWorkerStart != nil {
		pool.OnWorkerStart()
	}
	defer pool.WaitGroup.Done()

	log := logger.FromContext(ctx).With("module", "worker pool")
	defer log.LogAttrs(ctx, slog.LevelInfo, "worker stopped")

	for {
		select {
		case <-ctx.Done():
			return
		case orderID, ok := <-pool.Jobs:
			if !ok {
				return
			}

			if err := pool.Sema.AcquireWithTimeout(model.DefaultTimeout); err != nil {
				log.With("unit", "semaphore").LogAttrs(
					ctx,
					slog.LevelWarn,
					err.Error(),
				)
				pool.Results <- pool.dummy(orderID, dto.StatusAgentFailed)
				continue
			}
			log.With("unit", "semaphore").
				LogAttrs(ctx, slog.LevelDebug, "acquire")
			pool.RequestCounter <- struct{}{}

			data, err := pool.Client.GetOrderInfo(
				ctx,
				orderID)
			pool.Sema.Release()
			log.With("unit", "semaphore").
				LogAttrs(ctx, slog.LevelDebug, "release")

			if err != nil {
				if errors.Is(err, serviceerrs.ErrNoContent) {
					pool.Results <- pool.dummy(orderID, dto.StatusCalculatorNoContent)
					continue
				}

				pool.Results <- pool.dummy(orderID, dto.StatusCalculatorFailed)
				if ctx.Err() != nil {
					return
				}
				log.LogAttrs(ctx, slog.LevelError,
					"failed to get order info", slog.Any(model.KeyLoggerError, err))
				var tmrErr *serviceerrs.TooManyRequestsError
				if errors.As(err, &tmrErr) {
					cancelAll()
					pool.RateDataCh <- *tmrErr
					return
				}
				continue
			}
			pool.Results <- data
		}
	}
}

func (pool *WorkerPool) dummy(orderID string, status dto.AccrualStatus) dto.AccrualInfo {
	return dto.AccrualInfo{
		Order:  orderID,
		Status: string(status),
	}
}
