package domain

import "time"

type CheckResult struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code,omitempty"`
	OK         bool   `json:"ok"`
	LatencyMs  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
}

type BatchSummary struct {
	Total      int   `json:"total"`
	Up         int   `json:"up"`
	Down       int   `json:"down"`
	DurationMs int64 `json:"duration_ms"`
}

type Batch struct {
	ID        string        `json:"batch_id"`
	CreatedAt time.Time     `json:"created_at"`
	Summary   BatchSummary  `json:"summary"`
	Results   []CheckResult `json:"results"`
}

type CheckOptions struct {
	Concurrency int `json:"concurrency,omitempty"`
	TimeoutMs   int `json:"timeout_ms,omitempty"`
}

type BatchRequest struct {
	URLs    []string     `json:"urls"`
	Options CheckOptions `json:"options,omitempty"`
}

func Summarize(results []CheckResult, totalDuration time.Duration) BatchSummary {
	up := 0
	for _, r := range results {
		if r.OK {
			up++
		}
	}
	return BatchSummary{
		Total:      len(results),
		Up:         up,
		Down:       len(results) - up,
		DurationMs: totalDuration.Milliseconds(),
	}
}
