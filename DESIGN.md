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

Les interfaces `Checker` et `Store` sont définies dans le package `domain`. Les packages `checker` et `store` en dépendent — jamais l'inverse. Cela permet :
- De **remplacer** une implémentation (ex. : store SQLite au lieu de mémoire) sans toucher à l'API.
- De **tester** facilement avec des mocks.

### 2. Worker Pool (fan-out / fan-in) — Cœur concurrent

Le package `pool` implémente un pattern **fan-out / fan-in** avec un nombre borné de workers.

#### Choix des channels

| Canal | Type | Buffer | Justification |
|-------|------|--------|---------------|
| `jobs` | `chan string` | `concurrency` | Borné au nombre de workers. Un buffer de taille `concurrency` permet à l'émetteur de pousser des URLs sans bloquer tant qu'il y a des workers libres, sans allouer de mémoire pour toutes les URLs (contrairement à `len(urls)` qui serait du gaspillage pour de gros lots). |
| `results` | `chan CheckResult` | `len(urls)` | Chaque URL produit exactement un résultat. Un buffer de cette taille **garantit que les workers ne bloquent jamais en écriture**, même si le collecteur n'a pas encore commencé à lire. Cela élimine tout risque de deadlock. |

#### Channels directionnels

Dans la signature de la fonction `worker`, les canaux sont typés de manière directionnelle :
- `jobs <-chan string` — lecture seule : le worker consomme les URLs
- `results chan<- CheckResult` — écriture seule : le worker produit les résultats

Cela apporte une **sécurité à la compilation** : le compilateur Go interdit toute opération interdite (un worker ne peut pas fermer le canal `jobs`, par exemple).

#### Distribution des URLs dans une goroutine séparée

L'envoi des URLs dans le canal `jobs` se fait dans une goroutine dédiée (et non sur la goroutine principale). C'est **indispensable** pour éviter un deadlock : si le canal `jobs` est plus petit que `len(urls)`, l'émetteur bloquerait en attendant que des workers consomment, mais les workers ne consommeraient pas car la goroutine principale serait bloquée avant d'arriver à la collecte des résultats.

Cette goroutine écoute aussi `ctx.Done()` : si le contexte global est annulé, elle arrête d'envoyer de nouvelles URLs et ferme le canal `jobs` via `defer close(jobs)`.

#### Timeout à deux niveaux

- **Timeout global** (`TimeoutSec`) : un `context.WithTimeout` enveloppe l'ensemble du lot. Si le délai expire, toutes les vérifications en cours sont annulées.
- **Timeout per-URL** (`PerURLTimeoutSec`) : chaque worker crée un sous-contexte `context.WithTimeout(ctx, perURLTimeout)` avant d'appeler `checker.Check`. Si le contexte global expire, les sous-contextes sont aussi annulés automatiquement (propriété de `context.WithTimeout` sur un parent annulé).

#### Synchronisation

- **`sync.WaitGroup`** : chaque worker fait `defer wg.Done()` pour garantir le décrément même en cas de panique. La goroutine de fan-in attend `wg.Wait()` puis ferme `results`.
- **`sync.RWMutex`** dans `MemoryStore` : protège les accès concurrents au map des batches (lecture partagée via `RLock`, écriture exclusive via `Lock`).

#### Anti-patterns évités

| Anti-pattern | Comment il est évité |
|---|---|
| Goroutine par URL sans borne | Exactement `concurrency` goroutines sont lancées, jamais plus |
| Ignorer `ctx.Done()` | Les workers testent `ctx.Done()` avant chaque check + le sub-context per-URL propage l'annulation |
| Channel jamais fermé (deadlock) | `close(jobs)` via `defer` dans l'émetteur + `close(results)` après `wg.Wait()` |
| WaitGroup mal géré | `wg.Add(1)` avant le `go`, `defer wg.Done()` dans chaque worker |
| Data race | Aucune donnée partagée entre goroutines — toute communication passe par des channels |

### 6. Logging structuré avec `log/slog`

Utilisation du package standard `log/slog` (Go 1.21+) pour un logging structuré JSON, facilitant l'observabilité.
