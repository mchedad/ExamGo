package checker

import (
	"context"
	"net/http"
	"time"

	"moduleGo/urlwatch/internal/domain"
)

type HTTPChecker struct {
	client *http.Client
}

func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{client: &http.Client{}}
}

func (c *HTTPChecker) Check(ctx context.Context, url string) domain.CheckResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return domain.CheckResult{
			URL: url, LatencyMs: time.Since(start).Milliseconds(), Error: err.Error(),
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return domain.CheckResult{
			URL: url, LatencyMs: time.Since(start).Milliseconds(), Error: err.Error(),
		}
	}
	defer resp.Body.Close()

	return domain.CheckResult{
		URL:        url,
		StatusCode: resp.StatusCode,
		LatencyMs:  time.Since(start).Milliseconds(),
		OK:         resp.StatusCode >= 200 && resp.StatusCode < 400,
	}
}
