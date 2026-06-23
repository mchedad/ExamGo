# URLWatch

Microservice de **vérification d'URLs en masse** en Go.

Par Chedad Mehdi

## Build & Run

```bash
go build ./...
go vet ./...
go test ./...

# Lancement (LOG_LEVEL: DEBUG, INFO, WARN, ERROR)
LOG_LEVEL=INFO go run ./cmd/urlwatch
```

Le serveur démarre sur le port **8080**.

## API REST

### Health check

```bash
curl http://localhost:8080/healthz
```

### Créer un lot de vérifications

```bash
curl -X POST http://localhost:8080/v1/checks \
  -H "Content-Type: application/json" \
  -d '{
    "urls": ["https://go.dev", "https://github.com", "https://exemple.invalid"],
    "options": { "concurrency": 4, "timeout_ms": 2000 }
  }'
```

Réponse `201 Created` :
```json
{
  "batch_id": "b_4f3c1a",
  "created_at": "2026-06-18T10:00:00Z",
  "summary": { "total": 3, "up": 2, "down": 1, "duration_ms": 812 },
  "results": [
    { "url": "https://go.dev", "status_code": 200, "ok": true, "latency_ms": 120 },
    { "url": "https://exemple.invalid", "ok": false, "error": "dns: no such host", "latency_ms": 2001 }
  ]
}
```

### Consulter un lot par ID

```bash
curl http://localhost:8080/v1/checks/{batch_id}
```

### Contrat d'erreur

```json
{ "error": { "code": "batch_not_found", "message": "aucun lot avec l'id b_x" } }
```

## Architecture

Voir [DESIGN.md](DESIGN.md).
