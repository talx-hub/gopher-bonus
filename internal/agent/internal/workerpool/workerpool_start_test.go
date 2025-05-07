package workerpool

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/talx-hub/gopher-bonus/internal/model"
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
