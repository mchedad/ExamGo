package domain

import "context"

// Checker définit l'interface pour vérifier une URL unique.
// L'implémentation par défaut fait un vrai appel HTTP ;
// une implémentation mock (déterministe) sera utilisée dans les tests.
type Checker interface {
	Check(ctx context.Context, url string) CheckResult
}

// Store persiste et relit les lots.
type Store interface {
	Save(ctx context.Context, b Batch) error
	Get(ctx context.Context, id string) (Batch, error)
	List(ctx context.Context) ([]Batch, error)
}
