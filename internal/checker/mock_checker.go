package checker

import (
	"context"

	"moduleGo/urlwatch/internal/domain"
)

// MockChecker est un checker simulé pour les tests.
// Il permet de contrôler les résultats retournés.
type MockChecker struct {
	Results map[string]domain.URLResult
}

// NewMockChecker crée un nouveau MockChecker avec des résultats prédéfinis.
func NewMockChecker(results map[string]domain.URLResult) *MockChecker {
	return &MockChecker{Results: results}
}

// Check retourne le résultat prédéfini pour l'URL donnée.
func (m *MockChecker) Check(_ context.Context, url string) domain.URLResult {
	if result, ok := m.Results[url]; ok {
		return result
	}
	return domain.URLResult{
		URL:   url,
		Error: "URL non trouvée dans le mock",
	}
}
