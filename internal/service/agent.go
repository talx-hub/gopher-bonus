package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
	"github.com/talx-hub/gopher-bonus/internal/utils/semaphore"
)

type AccrualClient interface {
	GetOrderInfo(orderID string) (model.DTOAccrualCalculator, error)
}

type Agent struct {
	ordersCh    chan uint64
	responsesCh chan<- model.DTOAccrualCalculator
	client      AccrualClient
}

func New(
	ordersCh chan uint64,
	responsesCh chan<- model.DTOAccrualCalculator,
	accrualAddress string,
) *Agent {
	return &Agent{
		ordersCh:    ordersCh,
		responsesCh: responsesCh,
		client:      newCustomClient(accrualAddress),
	}
}

type requestRateData struct {
	timeToSleep time.Duration
	rpm         uint64
}

func (a *Agent) Run(ctx context.Context, maxRequestCount int) {
	var requestCount atomic.Uint64

	var wg sync.WaitGroup
	sema := semaphore.New(maxRequestCount)
	stopRequests := make(chan requestRateData)
	var stopRequestsFlag atomic.Bool
	startEpoch := time.Now()
	var timer *time.Timer
	rateData := requestRateData{}
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			close(a.responsesCh)
			close(stopRequests)
			return
		case rateData = <-stopRequests:
			timer = time.NewTimer(rateData.timeToSleep)
		case <-timer.C:
			sema.ChangeMaxRequestsCount(requestCount.Load(), startEpoch, rateData.rpm)
			stopRequestsFlag.Store(false)
		case orderNo := <-a.ordersCh:
			wg.Add(1)
			go func() {
				defer wg.Done()
				sema.Acquire()
				defer sema.Release()

				requestCount.Add(1)
				data, err := a.client.GetOrderInfo(strconv.FormatUint(orderNo, 10))
				if err != nil {
					// TODO: log
					fmt.Println(err)

					needToNotify := !stopRequestsFlag.CompareAndSwap(false, true)
					var tmrErr *serviceerrs.TooManyRequestsError
					if needToNotify && errors.As(err, &tmrErr) {
						stopRequests <- requestRateData{
							tmrErr.RetryAfter,
							tmrErr.RPM,
						}
					}
					a.ordersCh <- orderNo
					return
				}

				a.responsesCh <- data
			}()
		}
	}
}
