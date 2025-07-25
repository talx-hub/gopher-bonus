package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/service/dto"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
	"github.com/talx-hub/gopher-bonus/internal/utils/logger"
)

type HTTPClient struct {
	client         http.Client
	accrualAddress string
}

func New(accrualAddress string) *HTTPClient {
	return &HTTPClient{
		client:         http.Client{},
		accrualAddress: accrualAddress,
	}
}

func (c *HTTPClient) GetOrderInfo(ctx context.Context, orderID string,
) (dto.AccrualInfo, error) {
	tCtx, cancel := context.WithTimeout(ctx, model.DefaultTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(
		tCtx, http.MethodGet, c.accrualAddress+"/api/orders/"+orderID, http.NoBody)
	if err != nil {
		return dto.AccrualInfo{},
			fmt.Errorf("failed to create the request: %w", err)
	}
	resp, err := c.client.Do(request)
	if err != nil {
		return dto.AccrualInfo{},
			fmt.Errorf("failed to send request to Accrual: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log := logger.FromContext(ctx)
			log.LogAttrs(
				ctx,
				slog.LevelError,
				"failed to close the response body",
				slog.Any(model.KeyLoggerError, err),
			)
		}
	}()
	if err != nil {
		return dto.AccrualInfo{},
			fmt.Errorf("failed to read the body: %w", err)
	}

	data, err := c.handleRequestData(resp, body)
	if err == nil ||
		errors.Is(err, &serviceerrs.TooManyRequestsError{}) ||
		errors.Is(err, serviceerrs.ErrNoContent) {
		return data, err
	}

	return data, fmt.Errorf("request accrual failed: %w", err)
}

func (c *HTTPClient) handleRequestData(resp *http.Response, body []byte,
) (dto.AccrualInfo, error) {
	switch resp.StatusCode {
	case http.StatusOK:
		if ct := resp.Header.Get(model.HeaderContentType); ct != "application/json" {
			return dto.AccrualInfo{},
				fmt.Errorf("unexpected content type %s", ct)
		}
		data := dto.AccrualInfo{}
		if err := json.Unmarshal(body, &data); err != nil {
			return dto.AccrualInfo{},
				fmt.Errorf("request decoding error: %w", err)
		}
		return data, nil
	case http.StatusNoContent:
		return dto.AccrualInfo{}, serviceerrs.ErrNoContent
	case http.StatusTooManyRequests:
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter == "" {
			return dto.AccrualInfo{},
				errors.New("empty retry-after value")
		}
		ra, err := strconv.Atoi(retryAfter)
		if err != nil {
			return dto.AccrualInfo{},
				fmt.Errorf("retry after atoi failed: %w", err)
		}

		rpm, err := c.parseBody(body)
		if err != nil {
			return dto.AccrualInfo{},
				fmt.Errorf("failed to parse the body: %w", err)
		}

		return dto.AccrualInfo{},
			&serviceerrs.TooManyRequestsError{
				RetryAfter: time.Duration(ra) * time.Second,
				RPM:        rpm,
			}
	case http.StatusInternalServerError:
		return dto.AccrualInfo{},
			fmt.Errorf("accrual service error Body: %s", string(body))
	}

	return dto.AccrualInfo{},
		fmt.Errorf("unexpected status: %d Body: %q",
			resp.StatusCode, string(body))
}

func (c *HTTPClient) parseBody(b []byte) (uint64, error) {
	msg := string(b)
	const prefix = "No more than "
	const suffix = " requests per minute allowed"

	if !strings.HasPrefix(msg, prefix) || !strings.HasSuffix(msg, suffix) {
		return 0, fmt.Errorf("unexpected message format: %s", msg)
	}

	numStr := strings.TrimSuffix(strings.TrimPrefix(msg, prefix), suffix)

	var n uint64
	_, err := fmt.Sscanf(numStr, "%d", &n)
	if err != nil {
		return 0, fmt.Errorf("failed to parse number: %w", err)
	}

	return n, nil
}
