package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/serviceerrs"
)

type customClient struct {
	httpClient     http.Client
	accrualAddress string
}

func newCustomClient(accrualAddress string) *customClient {
	return &customClient{
		httpClient:     http.Client{Timeout: model.DefaultTimeout},
		accrualAddress: accrualAddress,
	}
}

func (c *customClient) GetOrderInfo(orderID string,
) (model.DTOAccrualInfo, error) {
	u := url.URL{
		Scheme: "http",
		Host:   c.accrualAddress,
		Path:   fmt.Sprintf("/api/orders/%s", orderID),
	}

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return model.DTOAccrualInfo{},
			fmt.Errorf("request accrual error: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	defer func() {
		if err = resp.Body.Close(); err != nil {
			// TODO: log
		}
	}()
	if err != nil {
		return model.DTOAccrualInfo{},
			fmt.Errorf("request read body error: %w", err)
	}

	data, err := c.handleRequestData(resp, body)
	if err == nil || errors.Is(err, &serviceerrs.TooManyRequestsError{}) {
		return data, err
	}

	return data, fmt.Errorf("request accrual failed: %w", err)
}

func (c *customClient) handleRequestData(resp *http.Response, body []byte,
) (model.DTOAccrualInfo, error) {
	switch resp.StatusCode {
	case http.StatusOK:
		if ct := resp.Header.Get(model.HeaderContentType); ct != "application/json" {
			return model.DTOAccrualInfo{},
				fmt.Errorf("unexpected content type %s", ct)
		}
		data := model.DTOAccrualInfo{}
		if err := json.Unmarshal(body, &data); err != nil {
			return model.DTOAccrualInfo{},
				fmt.Errorf("request decoding error: %w", err)
		}
		return data, nil
	case http.StatusNoContent:
		return model.DTOAccrualInfo{},
			errors.New("no content for this order")
	case http.StatusTooManyRequests:
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter == "" {
			return model.DTOAccrualInfo{},
				errors.New("empty retry-after value")
		}
		ra, err := strconv.Atoi(retryAfter)
		if err != nil {
			return model.DTOAccrualInfo{},
				fmt.Errorf("retry after atoi failed: %w", err)
		}

		rpm, err := c.parseTooManyRequestsBody(body)
		if err != nil {
			return model.DTOAccrualInfo{},
				fmt.Errorf("failed to parse N requests allowed: %w", err)
		}

		return model.DTOAccrualInfo{},
			&serviceerrs.TooManyRequestsError{
				RetryAfter: time.Duration(ra) * time.Second,
				RPM:        rpm,
			}
	case http.StatusInternalServerError:
		// TODO: log
		fmt.Println("Server error. Try again later.")
	}

	// TODO: log
	return model.DTOAccrualInfo{},
		fmt.Errorf("unexpected status: %d\nBody: %s", resp.StatusCode, string(body))
}

func (c *customClient) parseTooManyRequestsBody(b []byte) (uint64, error) {
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
