package client

import (
	"bytes"
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

func NewInventoryClient(cfg *config.Config) (*InventoryClient, error) {
	return &InventoryClient{
		baseUrl: cfg.InventoryServiceUrl,
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}, nil
}

func (c *InventoryClient) Reserve(request *domains.ReserveRequest) (*domains.ReserveResponse, error) {
	url := fmt.Sprintf("%s/api/v1/inventory/reserve", c.baseUrl)

	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	response, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	var result domains.ReserveResponse
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *InventoryClient) Confirm() {}

func (c *InventoryClient) Cancel() {}
