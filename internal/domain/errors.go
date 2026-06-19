package domain

import (
	"errors"
	"fmt"
)

// --- Erreurs sentinelles ---

// ErrBatchNotFound est renvoyé par Store.Get quand l'identifiant est inconnu.
var ErrBatchNotFound = errors.New("batch introuvable")

// --- Erreur personnalisée (type implémentant error) ---

// ValidationError est une erreur de validation portant le nom du champ fautif
// et un message explicatif. Elle implémente l'interface error.
type ValidationError struct {
	Field   string // nom du champ en erreur (ex: "urls", "concurrency")
	Message string // description du problème
}

// Error implémente l'interface error pour ValidationError.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation du champ '%s': %s", e.Field, e.Message)
}

// --- Fonctions utilitaires de validation ---

// NewValidationError crée une nouvelle erreur de validation pour un champ donné.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// WrapNotFound enveloppe ErrBatchNotFound avec un contexte supplémentaire.
// Utilise le wrapping idiomatique avec %w pour permettre errors.Is().
func WrapNotFound(id string) error {
	return fmt.Errorf("impossible de récupérer le batch %q: %w", id, ErrBatchNotFound)
}
