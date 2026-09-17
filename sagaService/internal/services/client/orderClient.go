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

type OrderClient struct {
	baseUrl string
	client  *http.Client
}

func NewOrderClient(cfg *config.Config) *OrderClient {
	return &OrderClient{
		baseUrl: cfg.OrderServiceUrl,
		client: &http.Client{
			Timeout:   120 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

func (c *OrderClient) CreateOrder(ctx context.Context, request *domains.CreateOrderRequest) (*domains.CreateOrderResponse, error) {
	url := fmt.Sprintf("%s/api/create_order", c.baseUrl)

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

	var result domains.CreateOrderResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *OrderClient) CancelOrder() {}
