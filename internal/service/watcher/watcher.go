package watcher

import (
	"context"
	"log/slog"
	"time"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/model/order"
	"github.com/talx-hub/gopher-bonus/internal/service/dto"
	"github.com/talx-hub/gopher-bonus/internal/utils/logger"
)

type orderRepo interface {
	SelectOrdersForProcessing(context.Context) ([]string, error)
	UpdateAccrualStatus(context.Context, *order.Order) error
}

type Watcher struct {
	orderRepo   orderRepo
	ordersCh    chan<- string
	responsesCh <-chan dto.AccrualInfo
}

func New(
	orderRepo orderRepo,
	ordersCh chan string,
	responsesCh chan dto.AccrualInfo,
) *Watcher {
	return &Watcher{
		orderRepo:   orderRepo,
		ordersCh:    ordersCh,
		responsesCh: responsesCh,
	}
}

func (w *Watcher) Run(ctx context.Context) {
	log := logger.FromContext(ctx).With("service", "watcher")
	log.LogAttrs(ctx, slog.LevelInfo, "running")

	selectTicker := time.NewTicker(model.WatcherTickTimeout)
	defer selectTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.LogAttrs(ctx, slog.LevelInfo, "stop signal received, exiting...")
			selectTicker.Stop()

		case <-selectTicker.C:
			go func() {
				orders, err := w.orderRepo.SelectOrdersForProcessing(ctx)
				if err != nil {
					log.LogAttrs(ctx,
						slog.LevelError,
						"failed to select order numbers for accrual",
						slog.Any(model.KeyLoggerError, err),
					)
					return
				}
				for _, o := range orders {
					w.ordersCh <- o
					err = w.orderRepo.UpdateAccrualStatus(ctx,
						&order.Order{
							ID:     o,
							Status: order.StatusProcessing,
						})
					if err != nil {
						log.LogAttrs(ctx,
							slog.LevelError,
							`failed to update order status to "PROCESSING"`,
							slog.String("order_no", o),
							slog.Any(model.KeyLoggerError, err),
						)
					}
				}
			}()

		case resp, ok := <-w.responsesCh:
			if !ok {
				close(w.ordersCh)
				log.LogAttrs(ctx, slog.LevelInfo, "stopped")
				return
			}
			var realStatus order.Status
			switch dto.AccrualStatus(resp.Status) {
			case dto.StatusAgentFailed,
				dto.StatusCalculatorFailed,
				dto.StatusCalculatorProcessing,
				dto.StatusCalculatorRegistered:
				realStatus = order.StatusProcessing

			case dto.StatusCalculatorInvalid:
				realStatus = order.StatusInvalid

			case dto.StatusCalculatorProcessed,
				dto.StatusCalculatorNoContent:
				realStatus = order.StatusProcessed
			}

			a, err := model.FromString(string(resp.Accrual))
			if err != nil {
				log.LogAttrs(ctx,
					slog.LevelError,
					"failed to convert amount from string",
					slog.String("amount", string(resp.Accrual)),
					slog.Any(model.KeyLoggerError, err),
				)
			}

			o := order.Order{
				ID:     resp.Order,
				Status: realStatus,
				Amount: a,
			}
			if err := w.orderRepo.UpdateAccrualStatus(ctx, &o); err != nil {
				log.LogAttrs(ctx,
					slog.LevelError,
					"failed to update accrual info",
					slog.Any(model.KeyLoggerError, err),
				)
			}
		}
	}
}
