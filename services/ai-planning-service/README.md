# AI Planning Service

AI Planning Service is a FastAPI microservice for itinerary generation. It exposes:

- `GET /health`
- `POST /generate-itinerary`

The public request and response contract is shared with Trip Service and should remain stable.

## Generator Modes

The service supports two generator modes:

- `mock`: deterministic local mock generator. This is the default and does not require Ollama.
- `ollama`: local LLM generation through the Ollama HTTP API, with optional fallback to `mock`.

Set the mode with:

```bash
ITINERARY_GENERATOR_MODE=mock
# or
ITINERARY_GENERATOR_MODE=ollama
```

If `ITINERARY_GENERATOR_MODE` is empty, the service defaults to `mock`. If it is unknown,
startup fails with a clear error.

## Environment

Use [.env.example](.env.example) as the local template:

```bash
cp .env.example .env
set -a; source .env; set +a
```

Defaults:

```bash
APP_ENV=development
HTTP_HOST=0.0.0.0
HTTP_PORT=8000
LOG_LEVEL=INFO

ITINERARY_GENERATOR_MODE=mock

OLLAMA_BASE_URL=http://ollama:11434
OLLAMA_MODEL=llama3.1:8b
OLLAMA_TIMEOUT_SECONDS=60
OLLAMA_TEMPERATURE=0.2
OLLAMA_NUM_PREDICT=2048
OLLAMA_FALLBACK_TO_MOCK=true
```

`OLLAMA_FALLBACK_TO_MOCK=true` means Ollama connection errors, non-2xx responses, invalid
Ollama API JSON, missing `response`, or invalid itinerary JSON will be logged and served by
the deterministic mock generator. With `false`, `/generate-itinerary` returns:

```json
{
  "error": "Failed to generate itinerary"
}
```

## Run Locally In Mock Mode

```bash
cd services/ai-planning-service
python3 -m venv .venv
source .venv/bin/activate
make install
ITINERARY_GENERATOR_MODE=mock uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload
```

## Run Locally With Ollama

Install and start Ollama, then pull the model:

```bash
ollama pull llama3.1:8b
```

Start the service against local Ollama:

```bash
cd services/ai-planning-service
source .venv/bin/activate
ITINERARY_GENERATOR_MODE=ollama \
OLLAMA_BASE_URL=http://127.0.0.1:11434 \
OLLAMA_MODEL=llama3.1:8b \
OLLAMA_FALLBACK_TO_MOCK=true \
uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload
```

## Run With Docker Compose

The repository-level compose file includes an `ollama` service:

```bash
cd infra
docker compose up --build
```

Ollama does not download models just because the container starts. Pull the model once:

```bash
docker compose exec ollama ollama pull llama3.1:8b
```

The relevant compose settings are:

```yaml
services:
  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama-data:/root/.ollama

  ai-planning-service:
    environment:
      ITINERARY_GENERATOR_MODE: ollama
      OLLAMA_BASE_URL: http://ollama:11434
      OLLAMA_MODEL: llama3.1:8b
      OLLAMA_TIMEOUT_SECONDS: "60"
      OLLAMA_TEMPERATURE: "0.2"
      OLLAMA_NUM_PREDICT: "2048"
      OLLAMA_FALLBACK_TO_MOCK: "true"
    depends_on:
      - ollama

volumes:
  ollama-data:
```

## Run With Docker

```bash
cd services/ai-planning-service
docker build -t ai-planning-service .
docker run --rm -p 8000:8000 -e ITINERARY_GENERATOR_MODE=mock ai-planning-service
```

For local host Ollama from Docker, set an accessible base URL for your platform, for example:

```bash
docker run --rm -p 8000:8000 \
  -e ITINERARY_GENERATOR_MODE=ollama \
  -e OLLAMA_BASE_URL=http://host.docker.internal:11434 \
  -e OLLAMA_MODEL=llama3.1:8b \
  ai-planning-service
```

## Health Check

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

## Generate Itinerary

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

Example response shape:

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
          "note": "Start in Rome with older streets and landmark context before crowds build.",
          "estimatedCost": 18
        }
      ]
    }
  ]
}
```

## Development Commands

```bash
make help
make fmt
make lint
make test
make check
```

## Troubleshooting

`Ollama connection refused`: confirm Ollama is running and `OLLAMA_BASE_URL` is reachable from
the AI Planning Service process. Use `http://127.0.0.1:11434` for local runs and
`http://ollama:11434` inside Docker Compose.

`model not found`: pull the configured model with `ollama pull llama3.1:8b` locally or
`docker compose exec ollama ollama pull llama3.1:8b` in Compose.

`invalid JSON from local model`: lower `OLLAMA_TEMPERATURE`, increase `OLLAMA_NUM_PREDICT` if
responses are truncated, or keep `OLLAMA_FALLBACK_TO_MOCK=true` while developing.

`fallback to mock behavior`: when fallback is enabled, the service logs the Ollama failure and
returns the deterministic mock itinerary instead of exposing raw LLM output or stack traces.
