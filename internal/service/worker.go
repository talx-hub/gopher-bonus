package service

import (
	"context"
	"errors"
	"runtime"
	"strconv"
	"sync"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
	"github.com/talx-hub/gopher-bonus/internal/utils/semaphore"
)

type WorkerPool struct {
	client         AccrualClient
	sema           *semaphore.Semaphore
	wg             *sync.WaitGroup
	jobs           <-chan uint64
	rateDataCh     chan<- requestRateData
	requestCounter chan<- struct{}
	results        chan<- model.DTOAccrualInfo
}

func NewWorkerPool(
	client AccrualClient,
	wg *sync.WaitGroup,
	jobs <-chan uint64,
	rateDataCh chan<- requestRateData,
	requestCounter chan<- struct{},
	results chan<- model.DTOAccrualInfo,
) *WorkerPool {
	return &WorkerPool{
		client:         client,
		wg:             wg,
		jobs:           jobs,
		rateDataCh:     rateDataCh,
		requestCounter: requestCounter,
		results:        results,
	}
}

func (w *WorkerPool) Start(ctx context.Context, maxRequestCount uint64) context.CancelFunc {
	w.sema = semaphore.New(maxRequestCount)
	workerCtx, workerCancel := context.WithCancel(ctx)
	workerCount := runtime.NumCPU() * model.DefaultWorkerCountMultiplier
	for i := 0; i < workerCount; i++ {
		w.wg.Add(1)
		go w.Run(workerCtx, workerCancel)
	}

	return workerCancel
}

func (w *WorkerPool) Run(ctx context.Context, cancelAll context.CancelFunc) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case orderID, ok := <-w.jobs:
			if !ok {
				return
			}
			w.requestCounter <- struct{}{}
			if err := w.sema.AcquireWithTimeout(model.DefaultTimeout); err != nil {
				// TODO: log
				w.results <- dummy(orderID)
				continue
			}

			data, err := w.client.GetOrderInfo(strconv.FormatUint(orderID, 10))
			w.sema.Release()
			if err != nil {
				// TODO: log
				var tmrErr *serviceerrs.TooManyRequestsError
				if ctx.Err() == nil && errors.As(err, &tmrErr) {
					cancelAll()
					w.rateDataCh <- requestRateData{
						tmrErr.RetryAfter,
						tmrErr.RPM,
					}
				}
				w.results <- dummy(orderID)
				continue
			}
			w.results <- data
		}
	}
}

func dummy(orderID uint64) model.DTOAccrualInfo {
	return model.DTOAccrualInfo{
		Order:  strconv.FormatUint(orderID, 10),
		Status: model.StatusCalculatorProcessing,
	}
}
