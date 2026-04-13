package healthchecks

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL    = "https://hc-ping.com"
	requestTimeout    = 10 * time.Second
	maxRetries        = 3
	initialRetryDelay = 1 * time.Second
)

// Client sends monitoring signals to healthchecks.io.
// All methods are safe to call concurrently and never return errors —
// failures are logged but do not interrupt the caller's workflow.
type Client struct {
	baseURL    string
	uuid       string
	http       *http.Client
	logger     *slog.Logger
	retryDelay time.Duration
}

// NewClient creates a healthchecks.io client that pings the check identified by uuid.
func NewClient(logger *slog.Logger, uuid string) *Client {
	return &Client{
		baseURL:    defaultBaseURL,
		uuid:       uuid,
		http:       &http.Client{Timeout: requestTimeout},
		logger:     logger,
		retryDelay: initialRetryDelay,
	}
}

// Start sends a "start" signal with a run ID to measure execution time.
func (c *Client) Start(ctx context.Context, rid string) {
	c.ping(ctx, "/start", rid, "")
}

// Success sends a "success" signal with diagnostic information.
func (c *Client) Success(ctx context.Context, rid string, body string) {
	c.ping(ctx, "", rid, body)
}

// Fail sends a "failure" signal with error details.
func (c *Client) Fail(ctx context.Context, rid string, body string) {
	c.ping(ctx, "/fail", rid, body)
}

func (c *Client) ping(ctx context.Context, suffix string, rid string, body string) {
	url := fmt.Sprintf("%s/%s%s?rid=%s", c.baseURL, c.uuid, suffix, rid)

	method := http.MethodHead
	if body != "" {
		method = http.MethodPost
	}

	delay := c.retryDelay
	for attempt := range maxRetries {
		var reqBody io.Reader = http.NoBody
		if body != "" {
			reqBody = strings.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			c.logger.Warn("healthchecks: failed to create request",
				"error", err, "attempt", attempt+1)
			return
		}
		if body != "" {
			req.Header.Set("Content-Type", "text/plain; charset=utf-8")
		}

		resp, err := c.http.Do(req)
		if err != nil {
			c.logger.Warn("healthchecks: request failed",
				"error", err, "attempt", attempt+1, "suffix", suffix)
			if attempt < maxRetries-1 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(delay):
					delay *= 2
				}
			}
			continue
		}
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return
		}

		c.logger.Warn("healthchecks: unexpected status",
			"status", resp.StatusCode, "suffix", suffix, "attempt", attempt+1)
		if attempt < maxRetries-1 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				delay *= 2
			}
		}
	}
}
