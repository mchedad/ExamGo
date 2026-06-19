package checker

import (
	"context"

	"moduleGo/urlwatch/internal/domain"
)

type MockChecker struct {
	Results map[string]domain.CheckResult
}

func NewMockChecker(results map[string]domain.CheckResult) *MockChecker {
	return &MockChecker{Results: results}
}

func (m *MockChecker) Check(_ context.Context, url string) domain.CheckResult {
	if result, ok := m.Results[url]; ok {
		return result
	}
	return domain.CheckResult{URL: url, Error: "URL non trouvée dans le mock"}
}
