package workerpool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/service/agent/internal/workerpool/mocks"
	"github.com/talx-hub/gopher-bonus/internal/service/dto"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
)

func ConfigureMockAccrualClient(t *testing.T) AccrualClient {
	t.Helper()

	mockClient := mocks.NewMockAccrualClient(t)
	mockClient.
		EXPECT().
		GetOrderInfo(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, orderID string) (dto.AccrualInfo, error) {
			if strings.HasPrefix(orderID, "2") {
				return dto.AccrualInfo{
					Order:   orderID,
					Status:  string(dto.StatusCalculatorProcessed),
					Accrual: json.Number(orderID),
				}, nil
			}

			if orderID == "429" {
				return dto.AccrualInfo{},
					&serviceerrs.TooManyRequestsError{
						RetryAfter: model.DefaultTimeout,
						RPM:        1,
					}
			}

			if orderID == "428" {
				const multiplier = 2
				time.Sleep(multiplier * model.DefaultTimeout)

				return dto.AccrualInfo{
					Order:   orderID,
					Status:  string(dto.StatusCalculatorProcessed),
					Accrual: json.Number(orderID),
				}, nil
			}

			if strings.HasPrefix(orderID, "5") {
				return dto.AccrualInfo{
					Order:  orderID,
					Status: string(dto.StatusCalculatorFailed),
				}, nil
			}
			return dto.AccrualInfo{}, nil
		})

	return mockClient
}

func ConfigureMockAlwaysTimeoutExceedSemaphore(t *testing.T) AccrualSemaphore {
	t.Helper()

	mockSema := mocks.NewMockAccrualSemaphore(t)
	mockSema.
		EXPECT().
		AcquireWithTimeout(mock.Anything).
		RunAndReturn(func(_ time.Duration) error {
			return serviceerrs.ErrSemaphoreTimeoutExceeded
		})

	return mockSema
}
