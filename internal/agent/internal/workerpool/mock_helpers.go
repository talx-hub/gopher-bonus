package workerpool

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/talx-hub/gopher-bonus/internal/agent/internal/workerpool/mocks"
	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
)

func ConfigureMockAccrualClient(t *testing.T) AccrualClient {
	t.Helper()

	mockClient := mocks.NewMockAccrualClient(t)
	mockClient.
		EXPECT().
		GetOrderInfo(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, orderID string) (model.DTOAccrualInfo, error) {
			if strings.HasPrefix(orderID, "2") {
				accrual, err := strconv.Atoi(orderID)
				if err != nil {
					return model.DTOAccrualInfo{},
						fmt.Errorf("unexpected test error: %w", err)
				}
				return model.DTOAccrualInfo{
					Order:   orderID,
					Status:  string(model.StatusCalculatorProcessed),
					Accrual: accrual,
				}, nil
			}

			if orderID == "429" {
				return model.DTOAccrualInfo{},
					&serviceerrs.TooManyRequestsError{
						RetryAfter: model.DefaultTimeout,
						RPM:        1,
					}
			}

			if orderID == "428" {
				const multiplier = 2
				time.Sleep(multiplier * model.DefaultTimeout)

				const accrual = 428
				return model.DTOAccrualInfo{
					Order:   orderID,
					Status:  string(model.StatusCalculatorProcessed),
					Accrual: accrual,
				}, nil
			}

			if strings.HasPrefix(orderID, "5") {
				return model.DTOAccrualInfo{
					Order:  orderID,
					Status: string(model.StatusCalculatorFailed),
				}, nil
			}
			return model.DTOAccrualInfo{}, nil
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
