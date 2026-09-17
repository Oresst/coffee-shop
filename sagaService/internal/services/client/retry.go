package client

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"
)

const (
	maxRetries = 3
	baseDelay  = 200 * time.Millisecond
	maxDelay   = 2 * time.Second
)

func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= http.StatusInternalServerError
}

func doWithRetry(ctx context.Context, httpClient *http.Client, newRequest func() (*http.Request, error)) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := newRequest()
		if err != nil {
			return nil, err
		}

		resp, err := httpClient.Do(req.WithContext(ctx))
		if err == nil && !isRetryableStatus(resp.StatusCode) {
			return resp, nil
		}

		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		if attempt == maxRetries {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoffDelay(attempt)):
		}
	}

	return nil, fmt.Errorf("after %d attempts: %w", maxRetries+1, lastErr)
}

func backoffDelay(attempt int) time.Duration {
	d := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
	if d > maxDelay {
		d = maxDelay
	}
	return d/2 + time.Duration(rand.Int63n(int64(d)/2+1))
}
