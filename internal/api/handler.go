// Package api fournit les handlers HTTP, le routage et les middlewares.
package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"moduleGo/urlwatch/internal/domain"
	"moduleGo/urlwatch/internal/pool"
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

	// Validation avec erreur personnalisée ValidationError
	if len(req.URLs) == 0 {
		valErr := domain.NewValidationError("urls", "la liste d'URLs ne peut pas être vide")
		h.respondError(w, http.StatusBadRequest, valErr.Error())
		return
	}
	if req.Concurrency < 0 {
		valErr := domain.NewValidationError("concurrency", "le niveau de parallélisme ne peut pas être négatif")
		h.respondError(w, http.StatusBadRequest, valErr.Error())
		return
	}
	if req.TimeoutSec < 0 {
		valErr := domain.NewValidationError("timeout_sec", "le délai d'expiration ne peut pas être négatif")
		h.respondError(w, http.StatusBadRequest, valErr.Error())
		return
	}

	// Valeurs par défaut
	concurrency := req.Concurrency
	if concurrency == 0 {
		concurrency = 5
	}
	timeout := req.TimeoutSec
	if timeout == 0 {
		timeout = 10
	}

	// Créer un context avec timeout
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Lancer la vérification concurrente
	start := time.Now()
	results := pool.Run(ctx, h.checker, req.URLs, concurrency)
	duration := time.Since(start)

	// Calculer le résumé via la fonction d'agrégation du domaine
	summary := domain.Summarize(results, duration)

	batch := domain.Batch{
		ID:        generateID(),
		CreatedAt: time.Now(),
		Results:   results,
		Summary:   summary,
	}

	// Persister le batch
	if err := h.store.Save(r.Context(), batch); err != nil {
		// Utiliser errors.As pour détecter une ValidationError
		var valErr *domain.ValidationError
		if errors.As(err, &valErr) {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.respondError(w, http.StatusInternalServerError, "erreur de sauvegarde: "+err.Error())
		return
	}

	h.logger.Info("batch créé",
		"id", batch.ID,
		"total", batch.Summary.Total,
		"available", batch.Summary.Available,
		"duration_ms", batch.Summary.DurationMs,
	)

	h.respondJSON(w, http.StatusCreated, batch)
}

// GetBatch retourne un batch par son identifiant.
func (h *Handler) GetBatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	batch, err := h.store.Get(r.Context(), id)
	if err != nil {
		// Utiliser errors.Is pour traduire ErrBatchNotFound en 404
		if errors.Is(err, domain.ErrBatchNotFound) {
			h.respondError(w, http.StatusNotFound, err.Error())
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, batch)
}

// ListBatches retourne la liste de tous les batches.
func (h *Handler) ListBatches(w http.ResponseWriter, r *http.Request) {
	batches, err := h.store.List(r.Context())
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
