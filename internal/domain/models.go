// Package domain contient les types métier, erreurs et interfaces du projet URLWatch.
// Les autres packages en dépendent (inversion de dépendance).
package domain

import "time"

// URLResult représente le résultat de la vérification d'une URL unique.
type URLResult struct {
	URL        string        `json:"url"`
	StatusCode int           `json:"status_code"`
	Latency    time.Duration `json:"latency_ms"`
	Available  bool          `json:"available"`
	Error      string        `json:"error,omitempty"`
}

// BatchSummary contient les statistiques agrégées d'un lot de vérifications.
type BatchSummary struct {
	Total     int           `json:"total"`
	Available int           `json:"available"`
	Failed    int           `json:"failed"`
	Duration  time.Duration `json:"duration_ms"`
}

// Batch représente un lot de vérifications d'URLs avec ses résultats et son résumé.
type Batch struct {
	ID        string       `json:"id"`
	URLs      []string     `json:"urls"`
	Results   []URLResult  `json:"results"`
	Summary   BatchSummary `json:"summary"`
	CreatedAt time.Time    `json:"created_at"`
}

// BatchRequest représente la requête d'un client pour vérifier un lot d'URLs.
type BatchRequest struct {
	URLs        []string `json:"urls"`
	Concurrency int      `json:"concurrency,omitempty"` // niveau de parallélisme (défaut : 5)
	TimeoutSec  int      `json:"timeout_sec,omitempty"` // délai d'expiration par URL en secondes (défaut : 10)
}
