package workerpool

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/service/agent/internal/workerpool/mocks"
	"github.com/talx-hub/gopher-bonus/internal/service/dto"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
	"github.com/talx-hub/gopher-bonus/internal/utils/semaphore"
)

func TestWorkerPool_worker_general(t *testing.T) {
	tests := []struct {
		name         string
		jobs         []string
		results      []dto.AccrualInfo
		requestCount int
		rateData     []serviceerrs.TooManyRequestsError
	}{
		{
			name: "happy case #1",
			jobs: []string{"201", "202", "203", "204", "205", "206"},
			results: []dto.AccrualInfo{
				{Order: "201", Status: "PROCESSED", Accrual: "201"},
				{Order: "202", Status: "PROCESSED", Accrual: "202"},
				{Order: "203", Status: "PROCESSED", Accrual: "203"},
				{Order: "204", Status: "PROCESSED", Accrual: "204"},
				{Order: "205", Status: "PROCESSED", Accrual: "205"},
				{Order: "206", Status: "PROCESSED", Accrual: "206"},
			},
			requestCount: 6,
			rateData:     []serviceerrs.TooManyRequestsError{},
		},
		{
			name: "happy case #3",
			jobs: []string{"200"},
			results: []dto.AccrualInfo{
				{Order: "200", Status: "PROCESSED", Accrual: "200"},
			},
			requestCount: 1,
			rateData:     []serviceerrs.TooManyRequestsError{},
		},
		{
			name: "random error #1",
			jobs: []string{"500", "200", "201"},
			results: []dto.AccrualInfo{
				{Order: "500", Status: "CALCULATOR_FAILED"},
				{Order: "200", Status: "PROCESSED", Accrual: "200"},
				{Order: "201", Status: "PROCESSED", Accrual: "201"},
			},
			requestCount: 3,
			rateData:     []serviceerrs.TooManyRequestsError{},
		},
		{
			name: "random error #2",
			jobs: []string{"500", "501", "502"},
			results: []dto.AccrualInfo{
				{Order: "500", Status: "CALCULATOR_FAILED"},
				{Order: "501", Status: "CALCULATOR_FAILED"},
				{Order: "502", Status: "CALCULATOR_FAILED"},
			},
			requestCount: 3,
			rateData:     []serviceerrs.TooManyRequestsError{},
		},
		{
			name: "random error #3",
			jobs: []string{"200", "500", "201", "501"},
			results: []dto.AccrualInfo{
				{Order: "200", Status: "PROCESSED", Accrual: "200"},
				{Order: "500", Status: "CALCULATOR_FAILED"},
				{Order: "201", Status: "PROCESSED", Accrual: "201"},
				{Order: "501", Status: "CALCULATOR_FAILED"},
			},
			requestCount: 4,
			rateData:     []serviceerrs.TooManyRequestsError{},
		},
		{
			name: "too many requests #1",
			jobs: []string{"429"},
			results: []dto.AccrualInfo{
				{Order: "429", Status: "CALCULATOR_FAILED"},
			},
			requestCount: 1,
			rateData: []serviceerrs.TooManyRequestsError{
				{RetryAfter: model.DefaultTimeout, RPM: 1},
			},
		},
		{
			name: "too many requests #2",
			jobs: []string{"429", "200", "201", "202"},
			results: []dto.AccrualInfo{
				{Order: "429", Status: "CALCULATOR_FAILED"},
			},
			requestCount: 1,
			rateData: []serviceerrs.TooManyRequestsError{
				{RetryAfter: model.DefaultTimeout, RPM: 1},
			},
		},
		{
			name: "too many requests #3",
			jobs: []string{"201", "202", "429", "203", "204", "205"},
			results: []dto.AccrualInfo{
				{Order: "201", Status: "PROCESSED", Accrual: "201"},
				{Order: "202", Status: "PROCESSED", Accrual: "202"},
				{Order: "429", Status: "CALCULATOR_FAILED"},
			},
			requestCount: 3,
			rateData: []serviceerrs.TooManyRequestsError{
				{RetryAfter: model.DefaultTimeout, RPM: 1},
			},
		},
		{
			name: "multiple too many requests",
			jobs: []string{"200", "429", "429", "429"},
			results: []dto.AccrualInfo{
				{Order: "200", Status: "PROCESSED", Accrual: "200"},
				{Order: "429", Status: "CALCULATOR_FAILED"},
			},
			requestCount: 2,
			rateData: []serviceerrs.TooManyRequestsError{
				{RetryAfter: model.DefaultTimeout, RPM: 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			generateJobsWrapper := func() chan string {
				return GenerateJobs(t, ctx, tt.jobs)
			}
			pool, rateDataCh, requestCountCh, resultCh :=
				SetupWorkerPool(t,
					&sync.WaitGroup{},
					ConfigureMockAccrualClient(t),
					semaphore.New(model.DefaultRequestCount),
					generateJobsWrapper)

			results, requests, errs := TestWorker(t,
				ctx, cancel, rateDataCh, requestCountCh, resultCh, pool)

			assert.Equal(t, tt.results, results)
			assert.Equal(t, tt.requestCount, len(requests))
			assert.Equal(t, tt.rateData, errs)
		})
	}
}

func TestWorkerPool_worker_noJobs(t *testing.T) {
	t.Run("no jobs for worker", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockClientNoExpectations := mocks.NewMockAccrualClient(t)
		generateJobsWrapper := func() chan string {
			return GenerateJobs(t, ctx, []string{})
		}
		pool, rateDataCh, requestCountCh, resultCh :=
			SetupWorkerPool(t,
				&sync.WaitGroup{},
				mockClientNoExpectations,
				semaphore.New(model.DefaultRequestCount),
				generateJobsWrapper)

		results, requests, errs := TestWorker(t,
			ctx, cancel, rateDataCh, requestCountCh, resultCh, pool)

		assert.Equal(t, []dto.AccrualInfo{}, results)
		assert.Equal(t, 0, len(requests))
		assert.Equal(t, []serviceerrs.TooManyRequestsError{}, errs)
		mockClientNoExpectations.AssertNotCalled(t, "GetOrderInfo")
	})
}

func TestWorkerPool_worker_manualCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobCtx, generateCancel := context.WithCancel(ctx)
	defer generateCancel()
	generateJobsWrapper := func() chan string {
		return GenerateInfiniteJobs(t, jobCtx)
	}
	pool, rateDataCh, requestCountCh, resultCh :=
		SetupWorkerPool(t,
			&sync.WaitGroup{},
			ConfigureMockAccrualClient(t),
			semaphore.New(model.DefaultRequestCount),
			generateJobsWrapper)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				t.Error("timed out waiting for job to complete")
				return
			}
		}
	}()

	time.AfterFunc(100*time.Millisecond, cancel)
	results, requests, errs := TestWorker(t,
		ctx, cancel, rateDataCh, requestCountCh, resultCh, pool)

	assert.NotEmpty(t, results)
	assert.NotZero(t, len(requests))
	assert.Equal(t, []serviceerrs.TooManyRequestsError{}, errs)
}

func TestWorkerPool_worker_semaphoreError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	generateJobsWrapper := func() chan string {
		return GenerateJobs(t, ctx, []string{"429", "200", "201", "500", "202", "501", "203"})
	}

	mockClientNoExpectations := mocks.NewMockAccrualClient(t)
	pool, rateDataCh, requestCountCh, resultCh :=
		SetupWorkerPool(t,
			&sync.WaitGroup{},
			mockClientNoExpectations,
			ConfigureMockAlwaysTimeoutExceedSemaphore(t),
			generateJobsWrapper)

	results, requests, errs := TestWorker(t,
		ctx, cancel, rateDataCh, requestCountCh, resultCh, pool)

	wantResults := []dto.AccrualInfo{
		{Order: "429", Status: string(dto.StatusAgentFailed)},
		{Order: "200", Status: string(dto.StatusAgentFailed)},
		{Order: "201", Status: string(dto.StatusAgentFailed)},
		{Order: "500", Status: string(dto.StatusAgentFailed)},
		{Order: "202", Status: string(dto.StatusAgentFailed)},
		{Order: "501", Status: string(dto.StatusAgentFailed)},
		{Order: "203", Status: string(dto.StatusAgentFailed)},
	}

	assert.Equal(t, wantResults, results)
	assert.Equal(t, 0, len(requests))
	assert.Equal(t, []serviceerrs.TooManyRequestsError{}, errs)
	mockClientNoExpectations.AssertNotCalled(t, "GetOrderInfo")
}
