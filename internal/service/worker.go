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

func (w *WorkerPool) Run(ctx context.Context, notifyAll context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			w.wg.Done()
			return
		case orderID := <-w.jobs:
			w.requestCounter <- struct{}{}
			w.sema.Acquire()
			data, err := w.client.GetOrderInfo(strconv.FormatUint(orderID, 10))
			w.sema.Release()
			if err != nil {
				// TODO: log
				var tmrErr *serviceerrs.TooManyRequestsError
				if ctx.Err() == nil && errors.As(err, &tmrErr) {
					notifyAll()
					w.rateDataCh <- requestRateData{
						tmrErr.RetryAfter,
						tmrErr.RPM,
					}
				}
				w.results <- model.DTOAccrualInfo{
					Order:  strconv.FormatUint(orderID, 10),
					Status: model.StatusCalculatorProcessing,
				}
				continue
			}
			w.results <- data
		}
	}
}
