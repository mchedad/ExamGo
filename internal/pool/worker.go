// Package pool implémente le cœur concurrent : worker pool avec fan-out / fan-in.
package pool

import (
	"context"
	"sync"

	"moduleGo/urlwatch/internal/domain"
)

// Run exécute la vérification d'une liste d'URLs en parallèle avec un niveau
// de concurrence borné. Il utilise un pattern fan-out / fan-in.
//
// - ctx : contexte pour le timeout et l'annulation
// - checker : l'implémentation du Checker à utiliser
// - urls : la liste des URLs à vérifier
// - concurrency : le nombre maximum de goroutines concurrentes
func Run(ctx context.Context, checker domain.Checker, urls []string, concurrency int) []domain.URLResult {
	// Canal pour distribuer les URLs aux workers (fan-out)
	jobs := make(chan string, len(urls))
	// Canal pour collecter les résultats (fan-in)
	results := make(chan domain.URLResult, len(urls))

	// Lancer les workers
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range jobs {
				select {
				case <-ctx.Done():
					results <- domain.URLResult{
						URL:   url,
						Error: ctx.Err().Error(),
					}
				default:
					results <- checker.Check(ctx, url)
				}
			}
		}()
	}

	// Envoyer les URLs dans le canal jobs
	for _, url := range urls {
		jobs <- url
	}
	close(jobs)

	// Attendre que tous les workers aient terminé, puis fermer le canal results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collecter tous les résultats
	var allResults []domain.URLResult
	for r := range results {
		allResults = append(allResults, r)
	}

	return allResults
}
