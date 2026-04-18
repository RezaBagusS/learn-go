package client

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type PartnerClient interface {
	SendWebhook(ctx context.Context, url string, payload []byte) (int, error)
}

type partnerClient struct {
	httpClient *http.Client
	maxRetries int
	log        *zap.Logger
}

// NewPartnerClient
func NewPartnerClient(timeoutSeconds int, maxRetries int, log *zap.Logger) PartnerClient {
	return &partnerClient{
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
		maxRetries: maxRetries,
		log:        log,
	}
}

func (c *partnerClient) SendWebhook(ctx context.Context, url string, payload []byte) (int, error) {
	var lastHTTPCode int
	var lastErr error

	// Loop
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			c.log.Warn("Retrying webhook request",
				zap.String("url", url),
				zap.Int("attempt", attempt),
				zap.Error(lastErr),
			)

			backoffDuration := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return lastHTTPCode, ctx.Err()
			case <-time.After(backoffDuration):
			}
		}

		// HTTP Request
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
		if err != nil {
			return 0, fmt.Errorf("failed to create request: %w", err)
		}

		// Set header JSON
		req.Header.Set("Content-Type", "application/json")

		// Request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("network error: %w", err)
			continue
		}

		lastHTTPCode = resp.StatusCode
		resp.Body.Close()

		// Status Code
		if lastHTTPCode >= 200 && lastHTTPCode < 300 {
			return lastHTTPCode, nil
		}

		lastErr = fmt.Errorf("received non-success http code: %d", lastHTTPCode)

		if lastHTTPCode >= 400 && lastHTTPCode < 500 {
			break
		}

	}

	return lastHTTPCode, fmt.Errorf("failed to send webhook after %d attempts: %v", c.maxRetries, lastErr)
}
