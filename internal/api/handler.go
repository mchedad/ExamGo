// Package api fournit les handlers HTTP, le routage et les middlewares.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"moduleGo/urlwatch/internal/domain"
	"moduleGo/urlwatch/internal/pool"

	"crypto/rand"
	"fmt"
)

// Handler regroupe les dépendances pour les handlers HTTP.
type Handler struct {
	checker domain.Checker
	store   domain.Store
	logger  *slog.Logger
}

// NewRouter crée et configure le routeur HTTP avec tous les endpoints.
func NewRouter(checker domain.Checker, store domain.Store, logger *slog.Logger) http.Handler {
	h := &Handler{
		checker: checker,
		store:   store,
		logger:  logger,
	}

	mux := http.NewServeMux()

	// Endpoints REST
	mux.HandleFunc("POST /api/batches", h.CreateBatch)
	mux.HandleFunc("GET /api/batches/{id}", h.GetBatch)
	mux.HandleFunc("GET /api/batches", h.ListBatches)
	mux.HandleFunc("GET /api/health", h.Health)

	// Appliquer le middleware de logging
	return loggingMiddleware(logger, mux)
}

// CreateBatch gère la création d'un nouveau lot de vérifications.
func (h *Handler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var req domain.BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "requête JSON invalide: "+err.Error())
		return
	}

	// Validation
	if len(req.URLs) == 0 {
		h.respondError(w, http.StatusBadRequest, domain.ErrEmptyURLs.Error())
		return
	}

	// Valeurs par défaut
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	timeout := req.TimeoutSec
	if timeout <= 0 {
		timeout = 10
	}

	// Créer un context avec timeout
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Lancer la vérification concurrente
	start := time.Now()
	results := pool.Run(ctx, h.checker, req.URLs, concurrency)
	duration := time.Since(start)

	// Calculer le résumé
	available := 0
	for _, res := range results {
		if res.Available {
			available++
		}
	}

	batch := domain.Batch{
		ID:   generateID(),
		URLs: req.URLs,
		Results: results,
		Summary: domain.BatchSummary{
			Total:     len(req.URLs),
			Available: available,
			Failed:    len(req.URLs) - available,
			Duration:  duration,
		},
		CreatedAt: time.Now(),
	}

	// Persister le batch
	if err := h.store.Save(batch); err != nil {
		h.respondError(w, http.StatusInternalServerError, "erreur de sauvegarde: "+err.Error())
		return
	}

	h.logger.Info("batch créé",
		"id", batch.ID,
		"total", batch.Summary.Total,
		"available", batch.Summary.Available,
		"duration", batch.Summary.Duration,
	)

	h.respondJSON(w, http.StatusCreated, batch)
}

// GetBatch retourne un batch par son identifiant.
func (h *Handler) GetBatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	batch, err := h.store.Get(id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, batch)
}

// ListBatches retourne la liste de tous les batches.
func (h *Handler) ListBatches(w http.ResponseWriter, r *http.Request) {
	batches, err := h.store.List()
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, batches)
}

// Health retourne un status de santé du service.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// respondJSON envoie une réponse JSON.
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("erreur encodage JSON", "error", err)
	}
}

// respondError envoie une réponse d'erreur JSON.
func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

// loggingMiddleware log chaque requête HTTP entrante.
func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("requête HTTP",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
		)
	})
}

// generateID génère un identifiant unique pour un batch.
func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
