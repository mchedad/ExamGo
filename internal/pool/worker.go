package pool

import (
	"context"
	"sync"
	"time"

	"moduleGo/urlwatch/internal/domain"
)

// Run vérifie les URLs en parallèle avec un worker pool borné (fan-out / fan-in).
func Run(ctx context.Context, checker domain.Checker, urls []string, concurrency int, perURLTimeout time.Duration) []domain.CheckResult {
	jobs := make(chan string, concurrency)
	results := make(chan domain.CheckResult, len(urls))

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go worker(ctx, &wg, checker, jobs, results, perURLTimeout)
	}

	go func() {
		defer close(jobs)
		for _, url := range urls {
			select {
			case jobs <- url:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	allResults := make([]domain.CheckResult, 0, len(urls))
	for r := range results {
		allResults = append(allResults, r)
	}
	return allResults
}

func worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	checker domain.Checker,
	jobs <-chan string,
	results chan<- domain.CheckResult,
	perURLTimeout time.Duration,
) {
	defer wg.Done()
	for url := range jobs {
		select {
		case <-ctx.Done():
			results <- domain.CheckResult{URL: url, Error: ctx.Err().Error()}
			continue
		default:
		}
		urlCtx, cancel := context.WithTimeout(ctx, perURLTimeout)
		results <- checker.Check(urlCtx, url)
		cancel()
	}
}
