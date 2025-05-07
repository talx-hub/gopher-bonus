package workerpool

import (
	"context"
	"sync"
	"testing"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
)

func TestWorker(t *testing.T,
	ctx context.Context,
	cancel context.CancelFunc,
	rateDataCh chan serviceerrs.TooManyRequestsError,
	requestCountCh chan struct{},
	resultCh chan model.DTOAccrualInfo,
	pool *WorkerPool,
) ([]model.DTOAccrualInfo, []struct{}, []serviceerrs.TooManyRequestsError) {
	t.Helper()

	helperCtx, helperCancel := context.WithCancel(context.Background())
	defer helperCancel()
	helperWg := &sync.WaitGroup{}

	helperWg.Add(1)
	var results []model.DTOAccrualInfo
	go func() {
		defer helperWg.Done()
		results = ListenChannel(t, helperCtx, resultCh)
	}()

	helperWg.Add(1)
	var requests []struct{}
	go func() {
		defer helperWg.Done()
		requests = ListenChannel(t, helperCtx, requestCountCh)
	}()

	helperWg.Add(1)
	var errs []serviceerrs.TooManyRequestsError
	go func() {
		defer helperWg.Done()
		errs = ListenChannel(t, helperCtx, rateDataCh)
	}()

	pool.WaitGroup.Add(1)
	go func() {
		pool.worker(ctx, cancel)
	}()
	pool.WaitGroup.Wait()
	close(rateDataCh)
	close(requestCountCh)
	close(resultCh)
	helperCancel()

	helperWg.Wait()

	return results, requests, errs
}

func SetupWorkerPool(t *testing.T,
	client AccrualClient,
	sema AccrualSemaphore,
	jobGenerator func() chan uint64,
) (*WorkerPool, chan serviceerrs.TooManyRequestsError, chan struct{}, chan model.DTOAccrualInfo) {
	t.Helper()

	rateDataCh := make(chan serviceerrs.TooManyRequestsError)
	requestCountCh := make(chan struct{})
	resultCh := make(chan model.DTOAccrualInfo)

	pool := ConfigureWorkerPool(t,
		client,
		sema,
		jobGenerator,
		rateDataCh,
		requestCountCh,
		resultCh)

	return pool, rateDataCh, requestCountCh, resultCh
}

func ConfigureWorkerPool(t *testing.T,
	client AccrualClient,
	sema AccrualSemaphore,
	jobs func() chan uint64,
	rateDataCh chan<- serviceerrs.TooManyRequestsError,
	requestCounter chan<- struct{},
	results chan<- model.DTOAccrualInfo,
) *WorkerPool {
	t.Helper()

	return &WorkerPool{
		Client:         client,
		Sema:           sema,
		WaitGroup:      &sync.WaitGroup{},
		Jobs:           jobs(),
		RateDataCh:     rateDataCh,
		RequestCounter: requestCounter,
		Results:        results,
	}
}

func GenerateJobs(t *testing.T, ctx context.Context, s []uint64,
) chan uint64 {
	t.Helper()

	jobs := make(chan uint64, len(s))

	go func() {
		defer close(jobs)
		for _, j := range s {
			select {
			case <-ctx.Done():
				return
			case jobs <- j:
			}
		}
	}()

	return jobs
}

func GenerateInfiniteJobs(t *testing.T, ctx context.Context) chan uint64 {
	t.Helper()

	const bigCapacity = 1024
	infiniteJobsCh := make(chan uint64, bigCapacity)

	go func() {
		defer close(infiniteJobsCh)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				infiniteJobsCh <- 200
			}
		}
	}()

	return infiniteJobsCh
}

func ListenChannel[T any](t *testing.T, ctx context.Context, dataCh <-chan T,
) []T {
	t.Helper()

	results := make([]T, 0)
	for {
		select {
		case <-ctx.Done():
			return results
		case data, ok := <-dataCh:
			if !ok {
				return results
			}
			results = append(results, data)
		}
	}
}
