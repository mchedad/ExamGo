package domain

import "context"

// Checker définit l'interface pour vérifier une URL.
// L'implémentation concrète effectue une requête HTTP.
type Checker interface {
	Check(ctx context.Context, url string) URLResult
}

// Store définit l'interface de persistance des lots (batches).
// Permet de sauvegarder et de retrouver un batch par son identifiant.
type Store interface {
	Save(batch Batch) error
	Get(id string) (Batch, error)
	List() ([]Batch, error)
}
