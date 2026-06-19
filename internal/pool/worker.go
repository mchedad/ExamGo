// Package pool implémente le cœur concurrent : worker pool avec fan-out / fan-in.
package pool

import (
	"context"
	"sync"
	"time"

	"moduleGo/urlwatch/internal/domain"
)

// Run exécute la vérification d'une liste d'URLs en parallèle avec un worker pool borné.
//
// Pattern fan-out / fan-in :
//   - Fan-out : les URLs sont distribuées aux workers via le canal `jobs`
//   - Fan-in  : les résultats sont collectés via le canal `results`
//
// Choix des channels (justification dans DESIGN.md) :
//   - jobs    : bufferisé à `concurrency` — limite la mémoire sans bloquer les workers
//   - results : bufferisé à `len(urls)` — les workers n'attendent jamais pour écrire
//
// Garanties :
//   - Le nombre de goroutines ne dépasse jamais `concurrency`
//   - Tous les channels sont fermés, toutes les goroutines se terminent
//   - Le contexte global et le timeout per-URL sont respectés
//
// Paramètres :
//   - ctx            : contexte global (timeout du lot, annulation)
//   - checker        : implémentation du Checker à utiliser
//   - urls           : liste des URLs à vérifier
//   - concurrency    : nombre maximum de goroutines simultanées (workers)
//   - perURLTimeout  : délai d'expiration pour chaque vérification individuelle
func Run(ctx context.Context, checker domain.Checker, urls []string, concurrency int, perURLTimeout time.Duration) []domain.CheckResult {
	// --- Channels ---

	// jobs : bufferisé à `concurrency`.
	// On ne bufferise pas à len(urls) car cela allouerait inutilement de la mémoire
	// pour de gros lots. Un buffer de taille `concurrency` suffit : l'émetteur peut
	// pousser autant de jobs qu'il y a de workers libres sans bloquer.
	jobs := make(chan string, concurrency)

	// results : bufferisé à len(urls).
	// Chaque URL produit exactement un résultat. Un buffer de cette taille garantit
	// que les workers ne bloquent JAMAIS en écriture, même si le collecteur n'a
	// pas encore commencé à lire. Cela évite tout deadlock.
	results := make(chan domain.CheckResult, len(urls))

	// --- Fan-out : lancement des workers bornés ---
	// On lance exactement `concurrency` goroutines — ni plus, ni moins.
	// C'est ce qui borne le parallélisme : pas de goroutine par URL.
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go worker(ctx, &wg, checker, jobs, results, perURLTimeout)
	}

	// --- Distribution des URLs dans une goroutine séparée ---
	// IMPORTANT : on envoie les jobs dans une goroutine dédiée pour éviter un deadlock.
	// Si on envoyait depuis la goroutine principale avec un canal jobs plus petit que
	// len(urls), l'envoi bloquerait avant que les workers aient pu consommer.
	// De plus, on écoute ctx.Done() pour arrêter l'envoi si le contexte est annulé.
	go func() {
		defer close(jobs) // Fermeture GARANTIE : les workers sortiront de `range jobs`
		for _, url := range urls {
			select {
			case jobs <- url:
				// URL envoyée avec succès au prochain worker disponible
			case <-ctx.Done():
				// Le contexte global a expiré ou a été annulé.
				// On arrête d'envoyer de nouvelles URLs.
				// Les URLs non envoyées ne seront pas traitées.
				return
			}
		}
	}()

	// --- Fan-in : attente de fin et fermeture de results ---
	// On attend dans une goroutine séparée que TOUS les workers aient terminé,
	// puis on ferme le canal results. Cela permet à la boucle de collecte
	// (range results) de se terminer proprement.
	go func() {
		wg.Wait()
		close(results)
	}()

	// --- Collecte des résultats ---
	// Pre-allouer la slice à la capacité maximale pour éviter les réallocations.
	allResults := make([]domain.CheckResult, 0, len(urls))
	for r := range results {
		allResults = append(allResults, r)
	}

	return allResults
}

// worker est une goroutine qui consomme des URLs depuis le canal jobs (lecture seule),
// les vérifie avec le checker, et envoie les résultats dans le canal results (écriture seule).
//
// Comportement vis-à-vis du contexte :
//   - Avant chaque vérification, le worker teste ctx.Done() via un select non-bloquant.
//     Si le contexte global est déjà annulé, il produit un résultat d'erreur sans appeler le checker.
//   - Chaque vérification utilise un sous-contexte avec timeout per-URL, dérivé du contexte global.
//     Si le contexte global expire pendant une vérification, le sous-contexte est aussi annulé
//     (propriété de context.WithTimeout sur un parent annulé).
//
// Le worker se termine quand le canal jobs est fermé (sortie de `range jobs`).
// Le defer wg.Done() garantit que le WaitGroup est décrémenté même en cas de panique.
func worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	checker domain.Checker,
	jobs <-chan string,             // canal directionnel : lecture seule
	results chan<- domain.CheckResult, // canal directionnel : écriture seule
	perURLTimeout time.Duration,
) {
	defer wg.Done()

	for url := range jobs {
		// Vérifier si le contexte global est déjà annulé AVANT de lancer la requête.
		// On utilise un select non-bloquant (avec default) pour ne pas attendre.
		select {
		case <-ctx.Done():
			// Le contexte est expiré/annulé : on produit un résultat d'erreur
			// sans effectuer de requête réseau inutile.
			results <- domain.CheckResult{
				URL:   url,
				Error: ctx.Err().Error(),
			}
			continue
		default:
			// Le contexte est encore actif : on procède à la vérification.
		}

		// Créer un sous-contexte avec timeout per-URL.
		// Ce sous-contexte hérite du contexte global : si le parent est annulé,
		// le sous-contexte l'est aussi automatiquement.
		urlCtx, cancel := context.WithTimeout(ctx, perURLTimeout)
		result := checker.Check(urlCtx, url)
		cancel() // Libérer les ressources du timer du contexte (bonne pratique)

		results <- result
	}
}
