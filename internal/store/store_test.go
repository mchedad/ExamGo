package store_test

import (
	"context"
	"errors"
	"testing"

	"moduleGo/urlwatch/internal/domain"
	"moduleGo/urlwatch/internal/store"
)

func TestMemoryStore_SaveAndGet(t *testing.T) {
	s := store.NewMemoryStore()
	ctx := context.Background()
	batch := domain.Batch{ID: "b_abc123", Summary: domain.BatchSummary{Total: 2, Up: 1, Down: 1}}

	if err := s.Save(ctx, batch); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := s.Get(ctx, "b_abc123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != batch.ID || got.Summary.Total != 2 {
		t.Errorf("Get() = %+v, want %+v", got, batch)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	s := store.NewMemoryStore()
	_, err := s.Get(context.Background(), "b_inexistant")
	if err == nil {
		t.Fatal("Get() devrait retourner une erreur")
	}
	if !errors.Is(err, domain.ErrBatchNotFound) {
		t.Errorf("Get() error = %v, want wrapping ErrBatchNotFound", err)
	}
}

func TestMemoryStore_SaveEmptyID(t *testing.T) {
	s := store.NewMemoryStore()
	err := s.Save(context.Background(), domain.Batch{})
	if err == nil {
		t.Fatal("Save() avec ID vide devrait retourner une erreur")
	}
	var valErr *domain.ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("Save() error devrait être un ValidationError, got %T", err)
	}
}

func TestMemoryStore_List(t *testing.T) {
	s := store.NewMemoryStore()
	ctx := context.Background()
	s.Save(ctx, domain.Batch{ID: "b_1"})
	s.Save(ctx, domain.Batch{ID: "b_2"})

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List() len = %d, want 2", len(list))
	}
}
