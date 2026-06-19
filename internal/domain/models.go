// Package domain contient les types métier, erreurs et interfaces du projet URLWatch.
// Les autres packages en dépendent (inversion de dépendance).
package domain

import "time"

// CheckResult représente le résultat de la vérification d'une URL unique.
type CheckResult struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
	Available  bool   `json:"available"`
	LatencyMs  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
}

// BatchSummary contient les statistiques agrégées d'un lot de vérifications.
type BatchSummary struct {
	Total     int   `json:"total"`
	Available int   `json:"available"`
	Failed    int   `json:"failed"`
	DurationMs int64 `json:"duration_ms"`
}

// Batch représente un lot de vérifications d'URLs avec ses résultats et son résumé.
type Batch struct {
	ID        string        `json:"id"`
	CreatedAt time.Time     `json:"created_at"`
	Results   []CheckResult `json:"results"`
	Summary   BatchSummary  `json:"summary"`
}

// BatchRequest représente la requête d'un client pour vérifier un lot d'URLs.
type BatchRequest struct {
	URLs             []string `json:"urls"`
	Concurrency      int      `json:"concurrency,omitempty"`        // niveau de parallélisme (défaut : 5)
	TimeoutSec       int      `json:"timeout_sec,omitempty"`        // timeout global du lot en secondes (défaut : 30)
	PerURLTimeoutSec int      `json:"per_url_timeout_sec,omitempty"` // timeout par URL en secondes (défaut : 10)
}

// Summarize calcule le résumé agrégé à partir d'une slice de CheckResult.
// Utilisation idiomatique d'un parcours de slice pour compter les succès/échecs.
func Summarize(results []CheckResult, totalDuration time.Duration) BatchSummary {
	available := 0
	for _, r := range results {
		if r.Available {
			available++
		}
	}
	return BatchSummary{
		Total:      len(results),
		Available:  available,
		Failed:     len(results) - available,
		DurationMs: totalDuration.Milliseconds(),
	}
}
