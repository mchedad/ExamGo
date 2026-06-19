// Package store fournit les implémentations de persistance des batches.
package store

import (
	"context"
	"fmt"
	"sync"

	"moduleGo/urlwatch/internal/domain"
)

// MemoryStore est une implémentation en mémoire de domain.Store.
// Thread-safe grâce à un RWMutex.
type MemoryStore struct {
	mu      sync.RWMutex
	batches map[string]domain.Batch
}

// NewMemoryStore crée un nouveau store en mémoire.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		batches: make(map[string]domain.Batch),
	}
}

// Save persiste un batch dans le store.
func (s *MemoryStore) Save(_ context.Context, b domain.Batch) error {
	if b.ID == "" {
		return domain.NewValidationError("id", "l'identifiant du batch ne peut pas être vide")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batches[b.ID] = b
	return nil
}

// Get récupère un batch par son identifiant.
// Retourne une erreur wrappant ErrBatchNotFound si l'ID est inconnu,
// exploitable via errors.Is(err, domain.ErrBatchNotFound).
func (s *MemoryStore) Get(_ context.Context, id string) (domain.Batch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	batch, ok := s.batches[id]
	if !ok {
		return domain.Batch{}, fmt.Errorf("store.Get(%q): %w", id, domain.ErrBatchNotFound)
	}
	return batch, nil
}

// List retourne tous les batches stockés.
func (s *MemoryStore) List(_ context.Context) ([]domain.Batch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Batch, 0, len(s.batches))
	for _, b := range s.batches {
		result = append(result, b)
	}
	return result, nil
}
