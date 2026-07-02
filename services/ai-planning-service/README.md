# AI Planning Service

AI Planning Service is a FastAPI microservice for itinerary generation. It exposes:

- `GET /health`
- `POST /generate-itinerary`
- `POST /optimize-budget/day`
- `GET /destination-context`
- `GET /destination-context/{destination}`
- `POST /destination-context/{destination}/preview-prompt`
- `POST /knowledge/search`

The public request and response contract is shared with Trip Service and should remain stable.
The destination context and knowledge endpoints are internal/admin/debug endpoints for development.

`POST /generate-itinerary` also accepts optional `userProfile` and
`userPreferences` fields. Trip Service fetches those trusted values from User
Service and forwards them during generation; older requests that omit them still
work. Prompt generation includes a dedicated profile/preferences section when
present and keeps the final model output constrained to the existing itinerary
JSON schema.

Full and partial generation requests also accept optional `weatherForecast`.
Trip Service fetches this from External Integrations Service when enabled and
forwards it without persisting it. When present, prompt generation adds a
`WEATHER FORECAST` section and asks the model to prefer indoor activities on
rainy days, avoid long outdoor walks during high heat, schedule parks/viewpoints
on better weather days, and add indoor backups when rain chance is high.

Full and partial generation requests also accept optional `accommodation`
context forwarded by Trip Service. The context may include name, type, address,
attached place coordinates, check-in/check-out dates, notes, and estimated stay
cost. When present, prompt generation adds an `ACCOMMODATION CONTEXT` section
and asks the model to plan each day to start/end near the stay when practical,
avoid unnecessary zig-zag routes, account for early/late travel time to or from
the stay, and avoid booking suggestions unless the user asks for them.

Example accommodation payload:

```json
{
  "accommodation": {
    "name": "Hotel Roma",
    "type": "hotel",
    "address": "Via Roma 10",
    "place": {
      "provider": "mock",
      "providerPlaceId": "mock-hotel-roma",
      "name": "Hotel Roma",
      "address": "Via Roma 10",
      "latitude": 41.9028,
      "longitude": 12.4964
    },
    "checkInDate": "2026-08-10",
    "checkOutDate": "2026-08-12"
  }
}
```

Partial regeneration requests also accept attached place metadata inside
`currentItinerary.days[].items[].place`. When those places include optional
`openingHours`, prompt generation adds an `ATTACHED PLACE OPENING HOURS` section
for regeneration prompts. The section uses the shared convention
`dayOfWeek: 1 = Monday ... 7 = Sunday` and local `HH:mm` intervals, then asks
the model not to keep an attached place scheduled outside its hours. Full
generation is unchanged because opening hours are normally only known after a
place has been attached.

Example weather payload:

```json
{
  "weatherForecast": {
    "destination": "Rome",
    "provider": "mock",
    "days": [
      {
        "date": "2026-08-10",
        "condition": "hot",
        "temperatureMinC": 24,
        "temperatureMaxC": 35,
        "precipitationChance": 5,
        "windSpeedKph": 10,
        "summary": "Hot and sunny",
        "warnings": [
          "High heat: avoid long outdoor walks at midday"
        ]
      }
    ]
  }
}
```

## Validation And Repair

Ollama mode now validates generated itineraries in two layers:

- Pydantic/schema validation for the public response shape.
- Business validation for usable itineraries: exact day count, ordered day numbers, pace-based
  item counts, supported item types, HH:MM times, chronological ordering, non-empty text,
  non-negative costs, duplicate item detection, and budget sanity checks.

Personalization checks are soft warnings, not hard failures. The validator warns
when generated text mentions avoided terms, when dietary restrictions are
present but food items do not reflect them, or when a low walking limit conflicts
with repeated long-walk wording. Weather checks are also soft warnings: rainy
days without apparent indoor alternatives, long walks during high heat, and
exposed viewpoints during cold/windy conditions are logged but do not reject the
itinerary.

When the first local model response is invalid, the service can make one repair request to
Ollama with the original trip details, the validation error, and the invalid response. The
repair request must return the same public JSON shape. Raw model output is never returned to
API clients.

## Estimated Cost (Budget Tracking v1)

Itinerary items carry an optional structured `estimatedCost`:

```json
{
  "estimatedCost": {
    "amount": 18,
    "currency": "EUR",
    "category": "ticket",
    "confidence": "medium",
    "source": "ai"
  }
}
```

- `category`: `food | transport | ticket | activity | accommodation | shopping | other`.
- `confidence`: `low | medium | high`; `source`: `ai | manual | provider`.
- Generated output defaults `source` to `ai`, `confidence` to `low`, and an
  unknown `category` to `other`. `currency` is upper-cased; an invalid currency
  is dropped.
- The legacy bare-number form (`"estimatedCost": 18`) is still accepted and
  coerced to `{ "amount": 18 }` for backward compatibility.
- Repair is lenient: an unrepairable cost object becomes `null` rather than
  failing the whole generation; a negative amount is left intact so the business
  validator can reject it (`negative_cost`).

Generation and regeneration prompts ask the model to include an approximate
`estimatedCost` object for paid activities, museum/tickets, restaurants, cafes,
transport, shopping, and accommodation. The model should prefer the requested
currency (the trip budget currency, else the user's preferred currency, else
`EUR`), but may use a known local currency when that is more natural for the
destination. Currency values must be uppercase 3-letter codes. When uncertain,
the model is told to use `null` instead of inventing an exact price or exchange
rate, and `0` for genuinely free stops. **AI estimates are approximate, not real
prices.** Trip Service owns any exchange-rate conversion in budget summaries.
The mock generator emits the same structured costs so local development
exercises the full budget flow.

## Budget Optimization v1

`POST /optimize-budget/day` returns a reviewable proposal for reducing the cost
of one itinerary day. It does not update Trip Service state directly; Trip
Service stores the returned proposal and applies it only after a user confirms.

Request shape:

```json
{
  "trip": {
    "destination": "Rome",
    "startDate": "2026-08-10",
    "budget": { "amount": 700, "currency": "EUR" }
  },
  "dayNumber": 2,
  "currentDay": { "day": 2, "title": "Museum day", "items": [] },
  "budgetContext": {
    "currency": "EUR",
    "tripBudget": 700,
    "tripEstimatedTotal": 920,
    "dayEstimatedTotal": 185,
    "dailyBudgetShare": 100,
    "targetReductionAmount": 80,
    "expensiveItems": []
  },
  "constraints": {
    "preserveMustSeeItems": true,
    "maxWalkingIncreaseKm": 2,
    "keepMealCount": true,
    "avoidReplacingManualCosts": true
  },
  "accommodation": null,
  "weather": null,
  "userPreferences": null,
  "instruction": "Keep the historical theme."
}
```

Response shape:

```json
{
  "summary": "Reduced estimated cost by replacing a paid tour.",
  "scope": "day",
  "dayNumber": 2,
  "currency": "EUR",
  "baseDayEstimatedTotal": 185,
  "proposedDayEstimatedTotal": 113,
  "estimatedSavingsAmount": 72,
  "confidence": "medium",
  "changes": [],
  "preservedItems": [],
  "tradeoffs": [],
  "warnings": ["Ticket prices are approximate."],
  "proposedDay": { "day": 2, "title": "Budget Museum Day", "items": [] }
}
```

The prompt asks the model to prefer free or lower-cost alternatives, preserve
core user interests and must-see/high-value items, keep meals/rest balance, keep
routes realistic around accommodation, respect supplied weather/opening-hours
context, and explain tradeoffs. It must return strict JSON matching the schema.
Prices and savings are approximate estimates, not guaranteed prices.

Pydantic validates the proposed day, non-negative totals and savings, confidence
enum, change list, and structured `estimatedCost` values. In mock mode the
service deterministically replaces the most expensive item with a cheaper
alternative so Trip Service smoke tests can exercise proposal creation and
application without Ollama.

Limitations: v1 optimizes one day at a time, returns one proposal, does not
perform booking or ticket purchase, and relies on user review before any
itinerary mutation.

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
OLLAMA_REPAIR_ENABLED=true
OLLAMA_REPAIR_ATTEMPTS=1
LOG_LLM_PAYLOADS=false
DESTINATION_CONTEXT_ENABLED=true
DESTINATION_CONTEXT_DIR=app/data/destinations

RAG_ENABLED=false
RAG_KNOWLEDGE_DIR=app/data/knowledge
RAG_CHROMA_DIR=app/data/chroma
RAG_COLLECTION_NAME=travel_knowledge
RAG_TOP_K=5
RAG_MIN_SCORE=0.0
ANONYMIZED_TELEMETRY=false
OLLAMA_EMBEDDING_MODEL=nomic-embed-text
OLLAMA_EMBEDDING_TIMEOUT_SECONDS=30
```

`OLLAMA_REPAIR_ENABLED=true` and `OLLAMA_REPAIR_ATTEMPTS=1` allow one repair call after invalid
JSON, schema validation failure, or business validation failure. Values above `1` are clamped
to `1`; `0` disables repair attempts.

`OLLAMA_FALLBACK_TO_MOCK=true` means Ollama connection errors, non-2xx responses, invalid
Ollama API JSON, missing `response`, invalid itinerary JSON, repair failures, or business
validation failures will be logged and served by the deterministic mock generator. With
`false`, `/generate-itinerary` returns:

```json
{
  "error": "Failed to generate itinerary"
}
```

`LOG_LLM_PAYLOADS=false` keeps full prompts and raw model responses out of logs. Setting it to
`true` only logs those payloads when `APP_ENV=development`; outside development, payload
logging remains disabled.

`DESTINATION_CONTEXT_ENABLED=true` loads curated file-based destination context from
`DESTINATION_CONTEXT_DIR`. Set it to `false` to disable context lookup; itinerary prompt
preview still works, but returns `destinationContextFound=false`.

`RAG_ENABLED=false` disables local-document retrieval. When set to `true`, Ollama mode searches
the ChromaDB collection named by `RAG_COLLECTION_NAME` and injects up to `RAG_TOP_K` retrieved
chunks into the itinerary and repair prompts. `RAG_TOP_K` is clamped to `1..10`. Search scores
are returned as `1 / (1 + chroma_distance)`, so higher is better.

## Run Locally In Mock Mode

```bash
cd services/ai-planning-service
python3 -m venv .venv
source .venv/bin/activate
make install
ITINERARY_GENERATOR_MODE=mock uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload
```

## Run Locally With Ollama

Install and start Ollama, then pull the generation and embedding models:

```bash
ollama pull llama3.1:8b
ollama pull nomic-embed-text
```

Start the service against local Ollama:

```bash
cd services/ai-planning-service
source .venv/bin/activate
ITINERARY_GENERATOR_MODE=ollama \
OLLAMA_BASE_URL=http://127.0.0.1:11434 \
OLLAMA_MODEL=llama3.1:8b \
OLLAMA_FALLBACK_TO_MOCK=true \
OLLAMA_REPAIR_ENABLED=true \
OLLAMA_REPAIR_ATTEMPTS=1 \
uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload
```

## Run With Docker Compose

The repository-level compose file includes an `ollama` service:

```bash
cd infra
docker compose up --build
```

Ollama does not download models just because the container starts. Pull the models once:

```bash
docker compose exec ollama ollama pull llama3.1:8b
docker compose exec ollama ollama pull nomic-embed-text
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
      OLLAMA_REPAIR_ENABLED: "true"
      OLLAMA_REPAIR_ATTEMPTS: "1"
      LOG_LLM_PAYLOADS: "false"
      RAG_ENABLED: "false"
      RAG_KNOWLEDGE_DIR: app/data/knowledge
      RAG_CHROMA_DIR: app/data/chroma
      RAG_COLLECTION_NAME: travel_knowledge
      RAG_TOP_K: "5"
      RAG_MIN_SCORE: "0.0"
      ANONYMIZED_TELEMETRY: "false"
      OLLAMA_EMBEDDING_MODEL: nomic-embed-text
      OLLAMA_EMBEDDING_TIMEOUT_SECONDS: "30"
    volumes:
      - ai-planning-chroma:/app/app/data/chroma
    depends_on:
      - ollama

volumes:
  ollama-data:
  ai-planning-chroma:
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
    "pace": "balanced",
    "userProfile": {
      "displayName": "Test Traveler",
      "homeCity": "Bratislava",
      "homeCountry": "Slovakia",
      "preferredCurrency": "EUR",
      "preferredLanguage": "en"
    },
    "userPreferences": {
      "travelStyles": ["budget", "food", "hidden_gems"],
      "pace": "balanced",
      "maxWalkingKmPerDay": 8,
      "foodPreferences": ["local", "cheap"],
      "avoid": ["nightclubs"],
      "preferredTransport": ["walking", "public_transport"],
      "accommodationStyle": ["budget_hotel"],
      "dietaryRestrictions": []
    }
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

## Destination Context Debug Endpoints

These endpoints are internal/admin/debug endpoints for development. No authentication is
implemented yet. Prompt preview may expose prompt details and should be protected before
production.

List loaded destination contexts:

```bash
curl http://localhost:8000/destination-context
```

Get one destination context by destination name or alias:

```bash
curl http://localhost:8000/destination-context/rome
```

Preview the exact itinerary prompt that would be sent to Ollama:

```bash
curl -X POST http://localhost:8000/destination-context/rome/preview-prompt \
  -H "Content-Type: application/json" \
  -d '{
    "tripId": "7b6e1f4e-7d8a-4e0e-9e7b-3cf87b0a5c92",
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

Example list response:

```json
{
  "items": [
    {
      "destination": "Paris",
      "aliases": ["paris, france"],
      "source": "file"
    },
    {
      "destination": "Rome",
      "aliases": ["roma"],
      "source": "file"
    }
  ]
}
```

If `DESTINATION_CONTEXT_ENABLED=false`, `GET /destination-context` returns an empty list,
destination lookups return `404`, and prompt preview returns a prompt without destination
context.

## Local Knowledge RAG V1

RAG v1 is local-document retrieval only. It is not scraping, Reddit/X/blog ingestion, external
travel APIs, or a cloud LLM integration.

ChromaDB anonymized telemetry is disabled by default with
`ANONYMIZED_TELEMETRY=false`, which avoids noisy PostHog telemetry errors during
local startup and indexing.

The service now has two knowledge layers:

- Destination context JSON in `app/data/destinations`: curated structured tips for prompt
  shaping and debug prompt preview.
- Local RAG documents in `app/data/knowledge`: markdown/text notes split into chunks, embedded
  with Ollama, stored in ChromaDB, searched at generation time, and injected as `RAG CONTEXT`.

Add local knowledge by creating `.md` or `.txt` files under a destination folder:

```text
app/data/knowledge/rome/food.md
app/data/knowledge/paris/budget.md
```

Index the files after Ollama is running and `nomic-embed-text` is pulled:

```bash
cd services/ai-planning-service
source .venv/bin/activate
RAG_CHROMA_DIR=app/data/chroma \
OLLAMA_BASE_URL=http://127.0.0.1:11434 \
python -m app.scripts.index_knowledge
```

Enable RAG for generation:

```bash
RAG_ENABLED=true \
ITINERARY_GENERATOR_MODE=ollama \
OLLAMA_BASE_URL=http://127.0.0.1:11434 \
uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload
```

Test search directly:

```bash
curl -X POST http://localhost:8000/knowledge/search \
  -H "Content-Type: application/json" \
  -d '{
    "destination": "Rome",
    "interests": ["food", "hidden_gems"],
    "query": "local food and non-touristy areas",
    "topK": 5
  }'
```

If RAG is disabled, the endpoint returns `{"items": []}`. If the Chroma collection is missing,
embedding fails, or there are no matching chunks, itinerary generation continues without RAG
context. The public `/generate-itinerary` response schema is unchanged; request
payloads may include optional `userProfile` and `userPreferences`.

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

`embedding model not found`: pull `nomic-embed-text` locally with `ollama pull nomic-embed-text`
or in Compose with `docker compose exec ollama ollama pull nomic-embed-text`.

`RAG disabled`: confirm `RAG_ENABLED=true` in the AI Planning Service environment. The default is
`false`.

`Chroma collection missing`: run `python -m app.scripts.index_knowledge` from
`services/ai-planning-service` after pulling the embedding model.

`no RAG search results`: check that the destination folder name matches the requested
destination after normalization, for example `rome` for `"Rome"`, and lower `RAG_MIN_SCORE` if
it was raised above `0.0`.

`invalid JSON from local model`: keep `OLLAMA_REPAIR_ENABLED=true` so the service asks Ollama
for one corrected JSON response. Lower `OLLAMA_TEMPERATURE`, increase `OLLAMA_NUM_PREDICT` if
responses are truncated, or keep `OLLAMA_FALLBACK_TO_MOCK=true` while developing.

`repair failed`: the repaired output still failed JSON/schema/business validation. With
`OLLAMA_FALLBACK_TO_MOCK=true`, the service returns mock data. With fallback disabled, it
returns `{"error": "Failed to generate itinerary"}` with HTTP 500.

`fallback to mock behavior`: when fallback is enabled, the service logs the Ollama failure and
returns the deterministic mock itinerary instead of exposing raw LLM output or stack traces.

`model gives too many/few itinerary items`: the business validator enforces exactly 3 relaxed,
4 balanced, or 5 intensive items per day. The repair prompt repeats this requirement before
fallback/error handling runs.

`budget validation failure`: when `budgetAmount` is provided and at least one item has an
estimated cost, the known estimated total cannot exceed the budget by more than 30%. Use null
for uncertain item costs rather than inflated guesses.
