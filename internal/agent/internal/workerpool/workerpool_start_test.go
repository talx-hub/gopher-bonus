package workerpool

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
	"github.com/talx-hub/gopher-bonus/internal/utils/semaphore"
)

func TestWorkerPool_Start_count_workers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := func() chan uint64 {
		return GenerateInfiniteJobs(t, ctx)
	}
	wg := &sync.WaitGroup{}
	pool, ch1, ch2, ch3 := SetupWorkerPool(t,
		wg,
		ConfigureMockAccrualClient(t),
		semaphore.New(model.DefaultRequestCount),
		jobs,
	)
	listenCtx, listenCancel := context.WithCancel(context.Background())
	defer listenCancel()
	go ListenChannel(t, listenCtx, ch1)
	go ListenChannel(t, listenCtx, ch2)
	go ListenChannel(t, listenCtx, ch3)

	var mu sync.Mutex
	startedCount := 0
	pool.OnWorkerStart = func() {
		mu.Lock()
		defer mu.Unlock()
		startedCount++
	}
	poolCancel := pool.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	wantWorkers := runtime.NumCPU() * model.DefaultWorkerCountMultiplier
	ready := make(chan struct{})
	if startedCount == wantWorkers {
		cancel()
		poolCancel()
		wg.Wait()
		listenCancel()
		close(ready)
	} else if startedCount > wantWorkers {
		t.Fatalf("%d workers were started but should only have %d",
			wantWorkers, startedCount)
	}

	testCtx, testCancel := context.WithTimeout(context.Background(), time.Second)
	defer testCancel()

	select {
	case <-ready:
	case <-testCtx.Done():
		t.Fatalf("timeout: not all workers started")
	}
}

func TestWorkerPool_Start_generalPipeline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := func() chan uint64 {
		return GenerateJobs(t, ctx, []uint64{
			200, 200, 500, 201, 202, 203, 204, 205, 506, 207, 208, 209, 210, 211,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225,
			212, 213, 214, 215, 216, 217, 500, 219, 220, 221, 222, 223, 224, 225,
		})
	}

	workerCount := runtime.NumCPU() * model.DefaultWorkerCountMultiplier
	twiceShrinkSemaCapacity := workerCount / 2
	wg := &sync.WaitGroup{}
	pool, errsCh, countCh, resultsCh := SetupWorkerPool(t,
		wg,
		ConfigureMockAccrualClient(t),
		semaphore.New(uint64(twiceShrinkSemaCapacity)),
		jobs,
	)
	listenCtx, listenCancel := context.WithCancel(context.Background())
	defer listenCancel()

	resultsWg := &sync.WaitGroup{}

	resultsWg.Add(1)
	var errs []serviceerrs.TooManyRequestsError
	go func() {
		defer resultsWg.Done()
		errs = ListenChannel(t, listenCtx, errsCh)
	}()

	resultsWg.Add(1)
	var counts []struct{}
	go func() {
		defer resultsWg.Done()
		counts = ListenChannel(t, listenCtx, countCh)
	}()

	resultsWg.Add(1)
	var res []model.DTOAccrualInfo
	go func() {
		defer resultsWg.Done()
		res = ListenChannel(t, listenCtx, resultsCh)
	}()

	poolCancel := pool.Start(ctx)
	wg.Wait()
	close(errsCh)
	close(countCh)
	close(resultsCh)
	listenCancel()

	resultsWg.Wait()
	cancel()
	poolCancel()

	calcFails := 0
	agentFails := 0
	ok := 0
	for _, r := range res {
		if r.Status == string(model.StatusCalculatorFailed) {
			calcFails++
		}
		if r.Status == string(model.StatusAgentFailed) {
			agentFails++
		}
		if r.Status == string(model.StatusCalculatorProcessed) {
			ok++
		}
	}

	wantCalculatorFailures := 3
	require.NotZero(t, len(counts))
	assert.Equal(t, wantCalculatorFailures, calcFails)
	assert.Equal(t, ok+calcFails, len(counts))
	assert.NotZero(t, ok)
	assert.Zero(t, agentFails)
	assert.Zero(t, len(errs))
}
