# AI Planning Service

AI Planning Service v1 is a FastAPI microservice for deterministic mock itinerary generation. It is intentionally not connected to a real LLM, RAG pipeline, scraper, or message broker yet. The current purpose is to prove HTTP service-to-service communication from Trip Service.

## Run locally

```bash
cd services/ai-planning-service
python3 -m venv .venv
source .venv/bin/activate
make install
uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload
```

Environment variables:

- `APP_ENV`, default `development`
- `HTTP_HOST`, default `0.0.0.0`
- `HTTP_PORT`, default `8000`
- `LOG_LEVEL`, default `INFO`

## Run with Docker

```bash
cd services/ai-planning-service
docker build -t ai-planning-service .
docker run --rm -p 8000:8000 ai-planning-service
```

Docker Compose example:

```yaml
services:
  ai-planning-service:
    build:
      context: ./services/ai-planning-service
    environment:
      APP_ENV: development
      HTTP_HOST: 0.0.0.0
      HTTP_PORT: "8000"
      LOG_LEVEL: INFO
    ports:
      - "8000:8000"
```

## Run tests

```bash
cd services/ai-planning-service
make test
```

## Development commands

```bash
make help
make fmt
make lint
make check
```

## Health check

```bash
curl http://localhost:8000/health
```

Expected response:

```json
{
  "status": "ok",
  "service": "ai-planning-service"
}
```

## Generate itinerary

```bash
curl -X POST http://localhost:8000/generate-itinerary \
  -H "Content-Type: application/json" \
  -d '{
    "tripId": "550e8400-e29b-41d4-a716-446655440000",
    "destination": "Rome",
    "startDate": "2026-08-10",
    "days": 4,
    "budgetAmount": 600,
    "budgetCurrency": "EUR",
    "travelers": 2,
    "interests": ["food", "history", "hidden_gems"],
    "pace": "balanced"
  }'
```

Example response:

```json
{
  "days": [
    {
      "day": 1,
      "title": "Day 1: Rome historic streets and local food",
      "items": [
        {
          "time": "09:00",
          "type": "place",
          "name": "Rome historic center walk",
          "note": "Start in Rome with older streets and landmark context before the day 1 crowds build.",
          "estimatedCost": 18
        },
        {
          "time": "12:30",
          "type": "food",
          "name": "Neighborhood trattoria or market stall",
          "note": "In Rome, pick a small local place and order a seasonal dish instead of the most visible tourist menu. Look one or two blocks away from the busiest square.",
          "estimatedCost": 15
        },
        {
          "time": "15:30",
          "type": "activity",
          "name": "Museum or archaeological site",
          "note": "Use the afternoon in Rome for a focused history stop on day 1.",
          "estimatedCost": 16
        }
      ]
    }
  ]
}
```
