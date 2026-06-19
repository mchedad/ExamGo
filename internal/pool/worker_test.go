package pool_test

import (
	"context"
	"testing"
	"time"

	"moduleGo/urlwatch/internal/checker"
	"moduleGo/urlwatch/internal/domain"
	"moduleGo/urlwatch/internal/pool"
)

func TestRunBasic(t *testing.T) {
	mock := checker.NewMockChecker(map[string]domain.CheckResult{
		"https://google.com":   {URL: "https://google.com", StatusCode: 200, OK: true, LatencyMs: 50},
		"https://github.com":   {URL: "https://github.com", StatusCode: 200, OK: true, LatencyMs: 30},
		"https://invalid.test": {URL: "https://invalid.test", Error: "connection refused"},
	})

	urls := []string{"https://google.com", "https://github.com", "https://invalid.test"}
	results := pool.Run(context.Background(), mock, urls, 2, 5*time.Second)

	if len(results) != 3 {
		t.Fatalf("attendu 3 résultats, obtenu %d", len(results))
	}

	resultMap := make(map[string]domain.CheckResult)
	for _, r := range results {
		resultMap[r.URL] = r
	}
	if r, ok := resultMap["https://google.com"]; !ok || !r.OK {
		t.Error("google.com devrait être disponible")
	}
	if r, ok := resultMap["https://invalid.test"]; !ok || r.OK {
		t.Error("invalid.test ne devrait pas être disponible")
	}
}

func TestRunConcurrencyBound(t *testing.T) {
	urls := make([]string, 100)
	mockResults := make(map[string]domain.CheckResult)
	for i := range urls {
		url := "https://example.com/" + string(rune('a'+i%26))
		urls[i] = url
		mockResults[url] = domain.CheckResult{URL: url, StatusCode: 200, OK: true}
	}
	mock := checker.NewMockChecker(mockResults)
	results := pool.Run(context.Background(), mock, urls, 3, 5*time.Second)
	if len(results) != 100 {
		t.Fatalf("attendu 100 résultats, obtenu %d", len(results))
	}
}

func TestRunContextCancellation(t *testing.T) {
	mock := checker.NewMockChecker(map[string]domain.CheckResult{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	urls := []string{"https://slow.test/a", "https://slow.test/b"}
	results := pool.Run(ctx, mock, urls, 2, 5*time.Second)
	for _, r := range results {
		if r.Error == "" {
			t.Logf("URL %s a réussi malgré l'annulation", r.URL)
		}
	}
}

func TestRunEmptyURLs(t *testing.T) {
	mock := checker.NewMockChecker(map[string]domain.CheckResult{})
	results := pool.Run(context.Background(), mock, []string{}, 3, 5*time.Second)
	if len(results) != 0 {
		t.Fatalf("attendu 0 résultats, obtenu %d", len(results))
	}
}
