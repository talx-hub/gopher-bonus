package workerpool

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/service/agent/internal/dto"
	"github.com/talx-hub/gopher-bonus/internal/service/agent/internal/workerpool/mocks"
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
				accrual, err := strconv.ParseFloat(orderID, 64)
				if err != nil {
					return dto.AccrualInfo{},
						fmt.Errorf("unexpected test error: %w", err)
				}
				return dto.AccrualInfo{
					Order:   orderID,
					Status:  string(dto.StatusCalculatorProcessed),
					Accrual: accrual,
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

				const accrual = 428
				return dto.AccrualInfo{
					Order:   orderID,
					Status:  string(dto.StatusCalculatorProcessed),
					Accrual: accrual,
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
