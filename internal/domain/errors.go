package domain

import "errors"

// Erreurs métier du domaine URLWatch.
var (
	// ErrBatchNotFound est renvoyé quand un batch n'existe pas dans le store.
	ErrBatchNotFound = errors.New("batch introuvable")

	// ErrEmptyURLs est renvoyé quand la liste d'URLs est vide.
	ErrEmptyURLs = errors.New("la liste d'URLs ne peut pas être vide")

	// ErrInvalidConcurrency est renvoyé quand le niveau de parallélisme est invalide.
	ErrInvalidConcurrency = errors.New("le niveau de parallélisme doit être supérieur à 0")
)
