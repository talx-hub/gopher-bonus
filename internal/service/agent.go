package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
	"github.com/talx-hub/gopher-bonus/internal/utils/semaphore"
)

type Agent struct {
	ordersCh    <-chan uint64
	responsesCh chan<- model.DTOAccrualCalculator
}

func New(
	ordersCh <-chan uint64,
	responsesCh chan<- model.DTOAccrualCalculator,
) *Agent {
	return &Agent{
		ordersCh:    ordersCh,
		responsesCh: responsesCh,
	}
}

type requestRateData struct {
	timeToSleep time.Duration
	rpm         uint64
}

func (a *Agent) Run(ctx context.Context, maxRequestCount int) {
	var currentRPM atomic.Uint64
	var requestCount atomic.Uint64
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				timer := time.NewTimer(1 * time.Minute)
				<-timer.C
				currentRPM.Store(requestCount.Swap(0))
			}
		}
	}()

	var wg sync.WaitGroup
	sema := semaphore.New(maxRequestCount)
	var stopRequests chan requestRateData
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			close(a.responsesCh)
			close(stopRequests)
			return
		case rateData := <-stopRequests:
			timer := time.NewTimer(rateData.timeToSleep)
			<-timer.C
			sema.ChangeMaxRequestsCount(currentRPM.Load(), rateData.rpm)
		case orderNo := <-a.ordersCh:
			go func() {
				wg.Add(1)
				sema.Acquire()
				defer sema.Release()

				data, err := a.requestAccrual(orderNo, &wg)
				requestCount.Add(1)
				var tmrErr *serviceerrs.TooManyRequestsError
				if err != nil && errors.As(err, &tmrErr) {
					stopRequests <- requestRateData{
						tmrErr.TimeToSleep, tmrErr.RPM}
					return
				}
				a.responsesCh <- data
			}()
		}
	}
}

func (a *Agent) requestAccrual(orderNo uint64, wg *sync.WaitGroup,
) (model.DTOAccrualCalculator, error) {
	defer wg.Done()

	return model.DTOAccrualCalculator{}, nil
}
