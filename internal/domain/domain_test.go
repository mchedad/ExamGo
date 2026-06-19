package domain_test

import (
	"testing"
	"time"

	"moduleGo/urlwatch/internal/domain"
)

func TestSummarize(t *testing.T) {
	tests := []struct {
		name     string
		results  []domain.CheckResult
		duration time.Duration
		wantUp   int
		wantDown int
	}{
		{
			name:     "tous disponibles",
			results:  []domain.CheckResult{{OK: true}, {OK: true}, {OK: true}},
			duration: 500 * time.Millisecond,
			wantUp:   3, wantDown: 0,
		},
		{
			name:     "tous en echec",
			results:  []domain.CheckResult{{OK: false}, {OK: false}},
			duration: 1 * time.Second,
			wantUp:   0, wantDown: 2,
		},
		{
			name:     "mixte",
			results:  []domain.CheckResult{{OK: true}, {OK: false}, {OK: true}, {OK: false}},
			duration: 800 * time.Millisecond,
			wantUp:   2, wantDown: 2,
		},
		{
			name:     "liste vide",
			results:  []domain.CheckResult{},
			duration: 0,
			wantUp:   0, wantDown: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := domain.Summarize(tt.results, tt.duration)
			if s.Total != len(tt.results) {
				t.Errorf("Total = %d, want %d", s.Total, len(tt.results))
			}
			if s.Up != tt.wantUp {
				t.Errorf("Up = %d, want %d", s.Up, tt.wantUp)
			}
			if s.Down != tt.wantDown {
				t.Errorf("Down = %d, want %d", s.Down, tt.wantDown)
			}
			if s.DurationMs != tt.duration.Milliseconds() {
				t.Errorf("DurationMs = %d, want %d", s.DurationMs, tt.duration.Milliseconds())
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := domain.NewValidationError("urls", "ne peut pas être vide")
	if err.Field != "urls" {
		t.Errorf("Field = %q, want %q", err.Field, "urls")
	}
	want := "validation du champ 'urls': ne peut pas être vide"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}
