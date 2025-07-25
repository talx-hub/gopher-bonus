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
	"github.com/talx-hub/gopher-bonus/internal/service/dto"
	"github.com/talx-hub/gopher-bonus/internal/utils/semaphore"
)

func TestWorkerPool_Start_count_workers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := func() chan string {
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
	wantWorkers := runtime.NumCPU() * model.DefaultWorkerCountMultiplier
	poolCancel := pool.Start(ctx, wantWorkers)
	time.Sleep(50 * time.Millisecond)

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

	jobs := func() chan string {
		return GenerateJobs(t, ctx, []string{
			"200", "200", "500", "201", "202", "203", "204", "205", "506", "207", "208", "209", "210", "211",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "500", "219", "220", "221", "222", "223", "224", "225",
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
	res, counts, errs := TestPool(t,
		ctx, cancel, wg, errsCh, countCh, resultsCh, pool, workerCount)

	calcFails := 0
	agentFails := 0
	ok := 0
	for _, r := range res {
		if r.Status == string(dto.StatusCalculatorFailed) {
			calcFails++
		}
		if r.Status == string(dto.StatusAgentFailed) {
			agentFails++
		}
		if r.Status == string(dto.StatusCalculatorProcessed) {
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

func TestWorkerPool_Start_tooManyRequests(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	prepared := []string{
		"212", "213", "214", "215", "216", "217", "500", "219", "220", "221", "222", "223", "224", "225",
		"212", "213", "214", "215", "216", "217", "500", "219", "220", "221", "222", "223", "224", "225",
		"429",
		"212", "213", "214", "215", "216", "217", "500", "219", "220", "221", "222", "223", "224", "225",
		"212", "213", "214", "215", "216", "217", "500", "219", "220", "221", "222", "223", "224", "225",
		"212", "213", "214", "215", "216", "217", "500", "219", "220", "221", "222", "223", "224", "225",
		"212", "213", "214", "215", "216", "217", "500", "219", "220", "221", "222", "223", "224", "225",
		"212", "213", "214", "215", "216", "217", "500", "219", "220", "221", "222", "223", "224", "225",
		"212", "213", "214", "215", "216", "217", "500", "219", "220", "221", "222", "223", "224", "225",
		"212", "213", "214", "215", "216", "217", "500", "219", "220", "221", "222", "223", "224", "225",
		"212", "213", "214", "215", "216", "217", "500", "219", "220", "221", "222", "223", "224", "225",
		"212", "213", "214", "215", "216", "217", "500", "219", "220", "221", "222", "223", "224", "225",
	}

	jobs := func() chan string {
		return GenerateJobs(t, ctx, prepared)
	}

	workerCount := 2
	semaCapacity := workerCount * 10
	wg := &sync.WaitGroup{}
	pool, errsCh, countCh, resultsCh := SetupWorkerPool(t,
		wg,
		ConfigureMockAccrualClient(t),
		semaphore.New(uint64(semaCapacity)),
		jobs,
	)
	res, counts, errs := TestPool(t,
		ctx, cancel, wg, errsCh, countCh, resultsCh, pool, workerCount)

	calcFails := 0
	agentFails := 0
	ok := 0
	for _, r := range res {
		if r.Status == string(dto.StatusCalculatorFailed) {
			calcFails++
		}
		if r.Status == string(dto.StatusAgentFailed) {
			agentFails++
		}
		if r.Status == string(dto.StatusCalculatorProcessed) {
			ok++
		}
	}

	triggerJobNo := 28
	safetyCoeff := 5
	requestsCanBeHandled := triggerJobNo + workerCount*safetyCoeff
	require.NotZero(t, len(counts))
	require.NotZero(t, calcFails)
	require.NotZero(t, ok)
	require.NotZero(t, len(errs))
	assert.Equal(t, 1, len(errs))
	assert.GreaterOrEqual(t, requestsCanBeHandled, len(counts))
}

func TestWorkerPool_Start_semaTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := func() chan string {
		return GenerateJobs(t, ctx, []string{
			"500", "500", "500", "501", "502", "503", "504", "505", "506", "507", "508", "509", "510", "511",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"500", "500", "500", "501", "502", "503", "504", "505", "506", "507", "508", "509", "510", "511",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"500", "500", "500", "501", "502", "503", "504", "505", "506", "507", "508", "509", "510", "511",
			"212", "213", "214", "215", "216", "217", "518", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"500", "500", "500", "501", "502", "503", "504", "505", "506", "507", "508", "509", "510", "511",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428", "428",
			"212", "213", "214", "215", "216", "217", "218", "219", "220", "221", "222", "223", "224", "225",
			"212", "213", "214", "215", "216", "217", "500", "219", "220", "221", "222", "223", "224", "225",
		})
	}

	workerCount := runtime.NumCPU() * model.DefaultWorkerCountMultiplier
	shrinkSemaCapacity := workerCount / model.DefaultWorkerCountMultiplier
	wg := &sync.WaitGroup{}
	pool, errsCh, countCh, resultsCh := SetupWorkerPool(t,
		wg,
		ConfigureMockAccrualClient(t),
		semaphore.New(uint64(shrinkSemaCapacity)),
		jobs,
	)
	res, counts, errs := TestPool(t,
		ctx, cancel, wg, errsCh, countCh, resultsCh, pool, workerCount)

	calcFails := 0
	agentFails := 0
	ok := 0
	for _, r := range res {
		if r.Status == string(dto.StatusCalculatorFailed) {
			calcFails++
		}
		if r.Status == string(dto.StatusAgentFailed) {
			agentFails++
		}
		if r.Status == string(dto.StatusCalculatorProcessed) {
			ok++
		}
	}

	assert.NotZero(t, len(counts))
	assert.NotZero(t, calcFails)
	assert.NotZero(t, ok)
	assert.NotZero(t, agentFails)
	assert.Zero(t, len(errs))
	assert.Equal(t, ok+calcFails, len(counts))
}
