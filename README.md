# URLWatch

Microservice de **vérification d'URLs en masse** écrit en Go.

## Description

URLWatch permet d'envoyer un lot d'URLs à vérifier. Le service les interroge **en parallèle** (code HTTP, latence, disponibilité), **agrège** les résultats et les **expose** via une API REST. Chaque lot de vérifications (« batch ») est conservé et consultable a posteriori.

## Build & Run

```bash
# Build
go build ./...

# Vérification statique
go vet ./...

# Tests
go test ./...

# Lancement
go run ./cmd/urlwatch
```

Le serveur démarre par défaut sur le port **8080**.

## API REST

### Health check

```bash
curl http://localhost:8080/api/health
```

### Créer un batch de vérification

```bash
curl -X POST http://localhost:8080/api/batches \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://www.google.com",
      "https://www.github.com",
      "https://httpstat.us/500",
      "https://domaine-inexistant.xyz"
    ],
    "concurrency": 3,
    "timeout_sec": 5
  }'
```

### Consulter un batch par ID

```bash
curl http://localhost:8080/api/batches/{id}
```

### Lister tous les batches

```bash
curl http://localhost:8080/api/batches
```

## Architecture

Voir [DESIGN.md](DESIGN.md) pour la justification architecturale.

## Structure du projet

```
urlwatch/
├── go.mod
├── README.md
├── DESIGN.md
├── JOURNAL_IA.md
├── .gitignore
├── cmd/
│   └── urlwatch/
│       └── main.go          # point d'entrée, câblage des dépendances
└── internal/
    ├── domain/              # types métier, erreurs, interfaces
    ├── checker/             # implémentation HTTP du Checker (+ mock)
    ├── pool/                # worker pool concurrent (fan-out / fan-in)
    ├── store/               # persistance en mémoire
    └── api/                 # handlers HTTP, routage, middleware
```
