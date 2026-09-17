package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/yourusername/saga-service/internal/config"
	"github.com/yourusername/saga-service/internal/domains"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"net/http"
	"time"
)

type InventoryClient struct {
	baseUrl string
	client  *http.Client
}

func NewInventoryClient(cfg *config.Config) *InventoryClient {
	return &InventoryClient{
		baseUrl: cfg.InventoryServiceUrl,
		client: &http.Client{
			Timeout:   120 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

func (c *InventoryClient) Reserve(ctx context.Context, request *domains.ReserveRequest) (*domains.ReserveResponse, error) {
	url := fmt.Sprintf("%s/api/v1/inventory/reserve", c.baseUrl)

	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	response, err := doWithRetry(ctx, c.client, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	var result domains.ReserveResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *InventoryClient) Confirm(ctx context.Context, request *domains.ConfirmReserveRequest) error {
	url := fmt.Sprintf("%s/api/v1/inventory/reserve/confirm", c.baseUrl)

	body, err := json.Marshal(request)
	if err != nil {
		return err
	}

	response, err := doWithRetry(ctx, c.client, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	return nil
}

func (c *InventoryClient) Cancel(ctx context.Context, request *domains.CancelReserveRequest) error {
	url := fmt.Sprintf("%s/api/v1/inventory/reserve/cancel", c.baseUrl)

	body, err := json.Marshal(request)
	if err != nil {
		return err
	}

	response, err := doWithRetry(ctx, c.client, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	return nil
}
