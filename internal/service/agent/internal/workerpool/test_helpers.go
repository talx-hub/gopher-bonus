package workerpool

import (
	"context"
	"sync"
	"testing"

	"github.com/talx-hub/gopher-bonus/internal/service/dto"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
)

func TestPool(t *testing.T,
	ctx context.Context,
	cancel context.CancelFunc,
	poolWG *sync.WaitGroup,
	rateDataCh chan serviceerrs.TooManyRequestsError,
	requestCountCh chan struct{},
	resultCh chan dto.AccrualInfo,
	pool *WorkerPool,
	workerCount int,
) ([]dto.AccrualInfo, []struct{}, []serviceerrs.TooManyRequestsError) {
	t.Helper()

	helperCtx, helperCancel := context.WithCancel(context.Background())
	defer helperCancel()

	helperWG := &sync.WaitGroup{}

	helperWG.Add(1)
	var errs []serviceerrs.TooManyRequestsError
	go func() {
		defer helperWG.Done()
		errs = ListenChannel(t, helperCtx, rateDataCh)
	}()

	helperWG.Add(1)
	var requests []struct{}
	go func() {
		defer helperWG.Done()
		requests = ListenChannel(t, helperCtx, requestCountCh)
	}()

	helperWG.Add(1)
	var results []dto.AccrualInfo
	go func() {
		defer helperWG.Done()
		results = ListenChannel(t, helperCtx, resultCh)
	}()

	poolCancel := pool.Start(ctx, workerCount)
	poolWG.Wait()
	close(rateDataCh)
	close(requestCountCh)
	close(resultCh)
	helperCancel()

	helperWG.Wait()

	cancel()
	poolCancel()

	return results, requests, errs
}

func TestWorker(t *testing.T,
	ctx context.Context,
	cancel context.CancelFunc,
	rateDataCh chan serviceerrs.TooManyRequestsError,
	requestCountCh chan struct{},
	resultCh chan dto.AccrualInfo,
	pool *WorkerPool,
) ([]dto.AccrualInfo, []struct{}, []serviceerrs.TooManyRequestsError) {
	t.Helper()

	helperCtx, helperCancel := context.WithCancel(context.Background())
	defer helperCancel()
	helperWg := &sync.WaitGroup{}

	helperWg.Add(1)
	var results []dto.AccrualInfo
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
	wg *sync.WaitGroup,
	client AccrualClient,
	sema AccrualSemaphore,
	jobGenerator func() chan string,
) (*WorkerPool, chan serviceerrs.TooManyRequestsError, chan struct{}, chan dto.AccrualInfo) {
	t.Helper()

	rateDataCh := make(chan serviceerrs.TooManyRequestsError)
	requestCountCh := make(chan struct{})
	resultCh := make(chan dto.AccrualInfo)

	pool := New(
		client,
		sema,
		wg,
		jobGenerator(),
		rateDataCh,
		requestCountCh,
		resultCh)

	return pool, rateDataCh, requestCountCh, resultCh
}

func GenerateJobs(t *testing.T, ctx context.Context, s []string,
) chan string {
	t.Helper()

	jobs := make(chan string, len(s))

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

func GenerateInfiniteJobs(t *testing.T, ctx context.Context) chan string {
	t.Helper()

	const bigCapacity = 1024
	infiniteJobsCh := make(chan string, bigCapacity)

	go func() {
		defer close(infiniteJobsCh)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				infiniteJobsCh <- "200"
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
