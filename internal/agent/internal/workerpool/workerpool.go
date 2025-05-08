package workerpool

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
)

type AccrualClient interface {
	GetOrderInfo(orderID string) (model.DTOAccrualInfo, error)
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
	Jobs           <-chan uint64
	RateDataCh     chan<- serviceerrs.TooManyRequestsError
	RequestCounter chan<- struct{}
	Results        chan<- model.DTOAccrualInfo
	OnWorkerStart  func()
}

func New(
	client AccrualClient,
	sema AccrualSemaphore,
	wg *sync.WaitGroup,
	jobs <-chan uint64,
	rateDataCh chan<- serviceerrs.TooManyRequestsError,
	requestCounter chan<- struct{},
	results chan<- model.DTOAccrualInfo,
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

	for {
		select {
		case <-ctx.Done():
			return
		case orderID, ok := <-pool.Jobs:
			if !ok {
				return
			}

			if err := pool.Sema.AcquireWithTimeout(model.DefaultTimeout); err != nil {
				// TODO: log
				pool.Results <- pool.dummy(orderID, model.StatusAgentFailed)
				continue
			}
			pool.RequestCounter <- struct{}{}
			data, err := pool.Client.GetOrderInfo(strconv.FormatUint(orderID, 10))
			pool.Sema.Release()
			if err != nil {
				// TODO: log
				pool.Results <- pool.dummy(orderID, model.StatusCalculatorFailed)
				var tmrErr *serviceerrs.TooManyRequestsError
				if ctx.Err() == nil && errors.As(err, &tmrErr) {
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

func (pool *WorkerPool) dummy(orderID uint64, status model.AccrualStatus) model.DTOAccrualInfo {
	return model.DTOAccrualInfo{
		Order:  strconv.FormatUint(orderID, 10),
		Status: string(status),
	}
}
