package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"moduleGo/urlwatch/internal/domain"
	"moduleGo/urlwatch/internal/pool"
)

// Types pour le contrat d'erreur JSON.
type apiErr struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorBody struct {
	Error apiErr `json:"error"`
}

type Handler struct {
	checker domain.Checker
	store   domain.Store
	logger  *slog.Logger
}

func NewRouter(checker domain.Checker, store domain.Store, logger *slog.Logger) http.Handler {
	h := &Handler{checker: checker, store: store, logger: logger}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/checks", h.CreateBatch)
	mux.HandleFunc("GET /v1/checks/{id}", h.GetBatch)
	mux.HandleFunc("GET /healthz", h.Health)

	return recoveryMiddleware(logger, loggingMiddleware(logger, mux))
}

func (h *Handler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var req domain.BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondErr(w, http.StatusBadRequest, "invalid_request", "corps JSON invalide: "+err.Error())
		return
	}

	// Validation des URLs
	if len(req.URLs) == 0 {
		h.respondErr(w, http.StatusBadRequest, "invalid_request", "urls est obligatoire et ne peut pas être vide")
		return
	}
	if len(req.URLs) > 100 {
		h.respondErr(w, http.StatusBadRequest, "invalid_request", "urls ne peut pas contenir plus de 100 entrées")
		return
	}
	for _, raw := range req.URLs {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			h.respondErr(w, http.StatusBadRequest, "invalid_request",
				fmt.Sprintf("URL invalide (http/https requis): %s", raw))
			return
		}
	}

	// Defaults et bornes pour concurrency
	concurrency := req.Options.Concurrency
	if concurrency == 0 {
		concurrency = 8
	}
	if concurrency < 1 || concurrency > 50 {
		h.respondErr(w, http.StatusBadRequest, "invalid_request", "concurrency doit être entre 1 et 50")
		return
	}

	// Defaults et bornes pour timeout_ms (per-URL)
	timeoutMs := req.Options.TimeoutMs
	if timeoutMs == 0 {
		timeoutMs = 5000
	}
	if timeoutMs < 100 || timeoutMs > 30000 {
		h.respondErr(w, http.StatusBadRequest, "invalid_request", "timeout_ms doit être entre 100 et 30000")
		return
	}

	perURLTimeout := time.Duration(timeoutMs) * time.Millisecond
	// Timeout global : assez de temps pour tous les "rounds" de workers
	globalTimeout := perURLTimeout * time.Duration(len(req.URLs)/concurrency+1)
	ctx, cancel := context.WithTimeout(r.Context(), globalTimeout)
	defer cancel()

	start := time.Now()
	results := pool.Run(ctx, h.checker, req.URLs, concurrency, perURLTimeout)
	duration := time.Since(start)

	summary := domain.Summarize(results, duration)
	batch := domain.Batch{
		ID:        generateBatchID(),
		CreatedAt: time.Now().UTC(),
		Summary:   summary,
		Results:   results,
	}

	if err := h.store.Save(r.Context(), batch); err != nil {
		h.respondErr(w, http.StatusInternalServerError, "internal", "erreur de sauvegarde: "+err.Error())
		return
	}

	h.logger.Info("batch créé", "batch_id", batch.ID, "total", summary.Total, "up", summary.Up, "duration_ms", summary.DurationMs)
	h.respondJSON(w, http.StatusCreated, batch)
}

func (h *Handler) GetBatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	batch, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrBatchNotFound) {
			h.respondErr(w, http.StatusNotFound, "batch_not_found",
				fmt.Sprintf("aucun lot avec l'id %s", id))
			return
		}
		h.respondErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, batch)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) respondErr(w http.ResponseWriter, status int, code, message string) {
	h.respondJSON(w, status, errorBody{Error: apiErr{Code: code, Message: message}})
}

func generateBatchID() string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("b_%x", b)
}
