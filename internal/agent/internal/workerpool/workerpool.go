package workerpool

import (
	"context"
	"errors"
	"runtime"
	"strconv"
	"sync"

	"github.com/talx-hub/gopher-bonus/internal/agent/internal/httpclient"
	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
	"github.com/talx-hub/gopher-bonus/internal/utils/semaphore"
)

type AccrualClient interface {
	GetOrderInfo(orderID string) (model.DTOAccrualInfo, error)
}

type AccrualSemaphore interface {
	AcquireWithTimeout(timeout time.Duration) error
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
}

func New(
	clientAddr string,
	wg *sync.WaitGroup,
	jobs <-chan uint64,
	rateDataCh chan<- serviceerrs.TooManyRequestsError,
	requestCounter chan<- struct{},
	results chan<- model.DTOAccrualInfo,
) *WorkerPool {
	return &WorkerPool{
		Client:         httpclient.New(clientAddr),
		WaitGroup:      wg,
		Jobs:           jobs,
		RateDataCh:     rateDataCh,
		RequestCounter: requestCounter,
		Results:        results,
	}
}

func (pool *WorkerPool) Start(ctx context.Context, maxRequestCount uint64) context.CancelFunc {
	pool.Sema = semaphore.New(maxRequestCount)
	workerCtx, workerCancel := context.WithCancel(ctx)
	workerCount := runtime.NumCPU() * model.DefaultWorkerCountMultiplier
	for i := 0; i < workerCount; i++ {
		pool.WaitGroup.Add(1)
		go pool.worker(workerCtx, workerCancel)
	}

	return workerCancel
}

func (pool *WorkerPool) worker(ctx context.Context, cancelAll context.CancelFunc) {
	defer pool.WaitGroup.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case orderID, ok := <-pool.Jobs:
			if !ok {
				return
			}
			pool.requestCounter <- struct{}{}
			if err := pool.Sema.AcquireWithTimeout(model.DefaultTimeout); err != nil {
				// TODO: log
				pool.Results <- pool.dummy(orderID, model.StatusAgentFailed)
				continue
			}
			data, err := pool.Client.GetOrderInfo(strconv.FormatUint(orderID, 10))
			pool.Sema.Release()
			if err != nil {
				// TODO: log
				var tmrErr *serviceerrs.TooManyRequestsError
				if ctx.Err() == nil && errors.As(err, &tmrErr) {
					cancelAll()
					pool.rateDataCh <- *tmrErr
				}
				pool.results <- pool.dummy(orderID, model.StatusCalculatorFailed)
				continue
			}
			pool.results <- data
		}
	}
}

func (pool *WorkerPool) dummy(orderID uint64, status model.AccrualStatus) model.DTOAccrualInfo {
	return model.DTOAccrualInfo{
		Order:  strconv.FormatUint(orderID, 10),
		Status: string(status),
	}
}
