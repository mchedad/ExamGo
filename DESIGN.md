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

### 2. Worker Pool (fan-out / fan-in)

Le package `pool` implémente un pattern **fan-out / fan-in** :
- **Fan-out** : les URLs sont distribuées via un canal aux workers.
- **Fan-in** : les résultats sont collectés via un canal unique.
- Le nombre de workers est borné par le paramètre `concurrency`.
- Le `context.Context` permet le **timeout** et l'**annulation**.

### 3. `cmd/urlwatch/main.go` mince

Le point d'entrée ne fait qu'assembler les dépendances. Aucune logique métier.

### 4. Package `internal`

Tous les packages internes sont sous `internal/` pour interdire leur importation par des projets externes, conformément aux conventions Go.

### 5. Concurrence et thread-safety

- Le `MemoryStore` utilise un `sync.RWMutex` pour protéger les accès concurrents.
- Le worker pool utilise des canaux Go pour la communication entre goroutines.

### 6. Logging structuré avec `log/slog`

Utilisation du package standard `log/slog` (Go 1.21+) pour un logging structuré JSON, facilitant l'observabilité.
