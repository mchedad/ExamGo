// Package checker fournit l'implémentation HTTP du Checker.
package checker

import (
	"context"
	"net/http"
	"time"

	"moduleGo/urlwatch/internal/domain"
)

// HTTPChecker implémente domain.Checker en effectuant de vraies requêtes HTTP.
type HTTPChecker struct {
	client *http.Client
}

// NewHTTPChecker crée un nouveau HTTPChecker avec un client HTTP par défaut.
func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{
		client: &http.Client{},
	}
}

// Check vérifie une URL en effectuant une requête HTTP GET.
// Le context permet de gérer le timeout et l'annulation.
func (c *HTTPChecker) Check(ctx context.Context, url string) domain.CheckResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return domain.CheckResult{
			URL:       url,
			LatencyMs: time.Since(start).Milliseconds(),
			Error:     err.Error(),
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return domain.CheckResult{
			URL:       url,
			LatencyMs: time.Since(start).Milliseconds(),
			Error:     err.Error(),
		}
	}
	defer resp.Body.Close()

	return domain.CheckResult{
		URL:        url,
		StatusCode: resp.StatusCode,
		LatencyMs:  time.Since(start).Milliseconds(),
		Available:  resp.StatusCode >= 200 && resp.StatusCode < 400,
	}
}
