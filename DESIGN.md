# DESIGN.md — Justification architecturale

## Vue d'ensemble

URLWatch suit une **architecture en couches** avec **inversion de dépendance** :

```
cmd/urlwatch/main.go  →  câblage uniquement
       │
       ▼
internal/api          →  couche de présentation (HTTP)
       │
       ▼
internal/pool         →  couche de concurrence (worker pool)
       │
       ▼
internal/domain       →  types métier & interfaces (centre de l'architecture)
       │
    ┌──┴──┐
    ▼     ▼
checker  store        →  implémentations concrètes
```

## Choix architecturaux

### 1. Inversion de dépendance via `internal/domain`

Les interfaces `Checker` et `Store` sont définies dans `domain`. Les packages concrets en dépendent — jamais l'inverse. Cela permet de remplacer une implémentation (ex : store SQLite) ou de tester avec des mocks sans toucher aux couches supérieures.

### 2. Worker Pool (fan-out / fan-in)

#### Choix des channels

| Canal | Buffer | Justification |
|-------|--------|---------------|
| `jobs` | `concurrency` | Évite d'allouer de la mémoire pour toutes les URLs. L'émetteur peut pousser sans bloquer tant qu'il y a des workers libres. |
| `results` | `len(urls)` | Les workers ne bloquent jamais en écriture → aucun risque de deadlock. |

Les canaux sont **directionnels** dans la signature de `worker` (`<-chan` / `chan<-`) pour une sécurité à la compilation.

L'envoi des URLs se fait dans une **goroutine séparée** : si le canal `jobs` est plus petit que `len(urls)`, l'émetteur bloquerait sur le goroutine principal avant d'arriver à la collecte. Cette goroutine écoute aussi `ctx.Done()` pour arrêter l'envoi en cas d'annulation.

#### Timeout à deux niveaux

- **Global** : `context.WithTimeout` enveloppe le lot entier (calculé selon le nombre d'URLs et la concurrence).
- **Per-URL** : chaque worker crée un sous-contexte avec `timeout_ms`. Si le parent expire, les sous-contextes sont annulés automatiquement.

#### Anti-patterns évités

| Anti-pattern | Comment il est évité |
|---|---|
| Goroutine par URL sans borne | Exactement `concurrency` goroutines |
| Ignorer `ctx.Done()` | Test avant chaque check + sub-context per-URL |
| Channel non fermé (deadlock) | `defer close(jobs)` + `close(results)` après `wg.Wait()` |
| WaitGroup mal géré | `wg.Add(1)` avant le `go`, `defer wg.Done()` |
| Data race | Communication exclusivement par channels |

### 3. Choix de `net/http` (stdlib) plutôt que Gin

Nous utilisons `net/http` de la bibliothèque standard plutôt que Gin pour les raisons suivantes :

1. **Zéro dépendance externe** : le projet ne dépend que de la stdlib Go, ce qui simplifie la maintenance et les mises à jour.
2. **Go 1.22+ enhanced routing** : depuis Go 1.22, `http.ServeMux` supporte nativement les patterns avec méthode (`"POST /v1/checks"`) et les paramètres de chemin (`{id}`), comblant l'écart principal avec Gin.
3. **Contrôle total** : pas de magie ni de conventions cachées. Le middleware est une simple fonction `func(http.Handler) http.Handler`.
4. **Adapté à un microservice** : pour une API REST avec 3 endpoints, un framework complet comme Gin serait surdimensionné.
5. **Cohérence pédagogique** : utiliser la stdlib permet de maîtriser les mécanismes fondamentaux de Go (Handler, HandlerFunc, middleware pattern).

### 4. Contrat d'erreur uniforme

Toutes les erreurs renvoient le même format JSON `{ "error": { "code": "...", "message": "..." } }` avec le code HTTP approprié (400, 404, 405, 500). Cela simplifie le parsing côté client.

### 5. Logging structuré (`log/slog`)

- Handler JSON pour une observabilité machine.
- Niveau configurable via `LOG_LEVEL` (env var).
- `/healthz` exclu des logs pour éviter la pollution par les sondes de vivacité.
- Middleware de recovery : une panic est catchée, loggée et transformée en réponse 500 propre.
