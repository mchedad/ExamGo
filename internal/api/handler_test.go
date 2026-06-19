package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"moduleGo/urlwatch/internal/api"
	"moduleGo/urlwatch/internal/domain"
)

// --- Mocks ---

type mockChecker struct {
	results map[string]domain.CheckResult
}

func (m *mockChecker) Check(_ context.Context, url string) domain.CheckResult {
	if r, ok := m.results[url]; ok {
		return r
	}
	return domain.CheckResult{URL: url, Error: "mock: not found"}
}

type mockStore struct {
	batches map[string]domain.Batch
}

func newMockStore() *mockStore {
	return &mockStore{batches: make(map[string]domain.Batch)}
}

func (m *mockStore) Save(_ context.Context, b domain.Batch) error {
	m.batches[b.ID] = b
	return nil
}

func (m *mockStore) Get(_ context.Context, id string) (domain.Batch, error) {
	b, ok := m.batches[id]
	if !ok {
		return domain.Batch{}, fmt.Errorf("mock: %w", domain.ErrBatchNotFound)
	}
	return b, nil
}

func (m *mockStore) List(_ context.Context) ([]domain.Batch, error) {
	var result []domain.Batch
	for _, b := range m.batches {
		result = append(result, b)
	}
	return result, nil
}

func newTestRouter() http.Handler {
	checker := &mockChecker{results: map[string]domain.CheckResult{
		"https://go.dev":          {URL: "https://go.dev", StatusCode: 200, OK: true, LatencyMs: 50},
		"https://exemple.invalid": {URL: "https://exemple.invalid", OK: false, Error: "dns: no such host", LatencyMs: 2001},
	}}
	store := newMockStore()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil)) // discard
	return api.NewRouter(checker, store, logger)
}

// --- Tests Handlers ---

func TestPostChecks_Success(t *testing.T) {
	router := newTestRouter()

	body := `{"urls":["https://go.dev","https://exemple.invalid"],"options":{"concurrency":2,"timeout_ms":3000}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/checks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp domain.Batch
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Summary.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Summary.Total)
	}
	if resp.Summary.Up != 1 {
		t.Errorf("up = %d, want 1", resp.Summary.Up)
	}
	if resp.Summary.Down != 1 {
		t.Errorf("down = %d, want 1", resp.Summary.Down)
	}
	if resp.ID == "" {
		t.Error("batch_id should not be empty")
	}
	if len(resp.Results) != 2 {
		t.Errorf("results len = %d, want 2", len(resp.Results))
	}
}

func TestGetChecks_NotFound(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/checks/b_inexistant", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if errResp.Error.Code != "batch_not_found" {
		t.Errorf("error code = %q, want %q", errResp.Error.Code, "batch_not_found")
	}
}

func TestPostChecks_ValidationErrors(t *testing.T) {
	router := newTestRouter()

	tests := []struct {
		name string
		body string
	}{
		{"urls vide", `{"urls":[]}`},
		{"urls manquant", `{}`},
		{"url non http", `{"urls":["ftp://example.com"]}`},
		{"url invalide", `{"urls":["pas une url"]}`},
		{"concurrency hors bornes", `{"urls":["https://go.dev"],"options":{"concurrency":100}}`},
		{"timeout_ms hors bornes", `{"urls":["https://go.dev"],"options":{"timeout_ms":50}}`},
		{"json invalide", `{pas du json}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/checks", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHealthz(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
