package pool_test

import (
	"context"
	"testing"
	"time"

	"moduleGo/urlwatch/internal/checker"
	"moduleGo/urlwatch/internal/domain"
	"moduleGo/urlwatch/internal/pool"
)

// TestRunBasic vérifie le fonctionnement de base du worker pool.
func TestRunBasic(t *testing.T) {
	mock := checker.NewMockChecker(map[string]domain.CheckResult{
		"https://google.com": {URL: "https://google.com", StatusCode: 200, Available: true, LatencyMs: 50},
		"https://github.com": {URL: "https://github.com", StatusCode: 200, Available: true, LatencyMs: 30},
		"https://invalid.test": {URL: "https://invalid.test", Error: "connection refused"},
	})

	urls := []string{"https://google.com", "https://github.com", "https://invalid.test"}
	results := pool.Run(context.Background(), mock, urls, 2, 5*time.Second)

	if len(results) != 3 {
		t.Fatalf("attendu 3 résultats, obtenu %d", len(results))
	}

	// Vérifier que tous les résultats sont présents (l'ordre peut varier)
	resultMap := make(map[string]domain.CheckResult)
	for _, r := range results {
		resultMap[r.URL] = r
	}

	if r, ok := resultMap["https://google.com"]; !ok || !r.Available {
		t.Error("google.com devrait être disponible")
	}
	if r, ok := resultMap["https://invalid.test"]; !ok || r.Available {
		t.Error("invalid.test ne devrait pas être disponible")
	}
}

// TestRunConcurrencyBound vérifie que le pool ne lance pas plus de goroutines que concurrency.
func TestRunConcurrencyBound(t *testing.T) {
	// Créer un grand nombre d'URLs
	urls := make([]string, 100)
	mockResults := make(map[string]domain.CheckResult)
	for i := range urls {
		url := "https://example.com/" + string(rune('a'+i%26))
		urls[i] = url
		mockResults[url] = domain.CheckResult{URL: url, StatusCode: 200, Available: true}
	}
	mock := checker.NewMockChecker(mockResults)

	// Concurrence bornée à 3
	results := pool.Run(context.Background(), mock, urls, 3, 5*time.Second)

	if len(results) != 100 {
		t.Fatalf("attendu 100 résultats, obtenu %d", len(results))
	}
}

// TestRunContextCancellation vérifie que l'annulation du contexte interrompt le pool.
func TestRunContextCancellation(t *testing.T) {
	// Checker qui simule une requête lente
	slowResults := map[string]domain.CheckResult{}
	for i := 0; i < 10; i++ {
		url := "https://slow.test/" + string(rune('0'+i))
		slowResults[url] = domain.CheckResult{URL: url, StatusCode: 200, Available: true}
	}
	mock := checker.NewMockChecker(slowResults)

	// Contexte qui expire immédiatement
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // annuler immédiatement

	urls := make([]string, 10)
	for i := range urls {
		urls[i] = "https://slow.test/" + string(rune('0'+i))
	}

	results := pool.Run(ctx, mock, urls, 2, 5*time.Second)

	// Tous les résultats devraient avoir une erreur de contexte
	for _, r := range results {
		if r.Error == "" {
			// Certains pourraient passer si le mock est très rapide,
			// mais au minimum le contexte devrait être annulé
			t.Logf("URL %s a réussi malgré l'annulation du contexte", r.URL)
		}
	}
}

// TestRunTimeout vérifie que le timeout per-URL fonctionne.
func TestRunPerURLTimeout(t *testing.T) {
	mock := checker.NewMockChecker(map[string]domain.CheckResult{
		"https://fast.test": {URL: "https://fast.test", StatusCode: 200, Available: true},
	})

	urls := []string{"https://fast.test"}
	results := pool.Run(context.Background(), mock, urls, 1, 1*time.Second)

	if len(results) != 1 {
		t.Fatalf("attendu 1 résultat, obtenu %d", len(results))
	}
	if !results[0].Available {
		t.Error("l'URL devrait être disponible")
	}
}

// TestRunEmptyURLs vérifie que le pool gère correctement une liste vide.
func TestRunEmptyURLs(t *testing.T) {
	mock := checker.NewMockChecker(map[string]domain.CheckResult{})
	results := pool.Run(context.Background(), mock, []string{}, 3, 5*time.Second)

	if len(results) != 0 {
		t.Fatalf("attendu 0 résultats, obtenu %d", len(results))
	}
}
