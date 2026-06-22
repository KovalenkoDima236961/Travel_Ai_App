# Web Smoke Test

Manual browser flow for the local full stack.

## Start The Stack

From the repository root:

```bash
docker compose -f infra/docker-compose.yml up --build
```

If you use `infra/.env`, pass it explicitly:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env up --build
```

Open:

```text
http://localhost:3000
```

## Browser Flow

1. Go to `http://localhost:3000`.
2. Open the create trip page.
3. Create a trip with:
   - destination: `Rome`
   - days: `2`
   - budget: `500 EUR`
   - travelers: `2`
   - interests: `food`, `history`, `hidden_gems`
   - pace: `balanced`
4. Confirm the app redirects to the trip detail page.
5. Click `Generate itinerary`.
6. Wait for completion.
7. Confirm the itinerary appears.
8. Refresh the page.
9. Confirm the itinerary persists.
10. Go to `/trips`.
11. Confirm the trip appears in the list.

## Troubleshooting

- CORS error in browser console: confirm Trip Service has
  `CORS_ALLOWED_ORIGINS=http://localhost:3000`, then rebuild/restart
  `trip-service`.
- Trip Service offline: check `docker compose -f infra/docker-compose.yml ps`
  and `docker compose -f infra/docker-compose.yml logs trip-service`.
- AI Planning Service offline: check
  `docker compose -f infra/docker-compose.yml logs ai-planning-service`.
- Ollama model not pulled: run
  `docker compose -f infra/docker-compose.yml exec ollama ollama pull llama3.1:8b`
  and
  `docker compose -f infra/docker-compose.yml exec ollama ollama pull nomic-embed-text`.
- Itinerary generation timeout: keep `TRIP_HTTP_WRITE_TIMEOUT` higher than
  `AI_PLANNING_TIMEOUT_SECONDS`, and keep `AI_PLANNING_TIMEOUT_SECONDS` higher
  than `OLLAMA_TIMEOUT_SECONDS`.
- AI service fallback to mock: with `OLLAMA_FALLBACK_TO_MOCK=true`, the AI
  service returns a mock itinerary if Ollama generation fails after its timeout.
  Check `ai-planning-service` logs for the original error.
- RAG returns no results: run `./scripts/index-knowledge.sh`, confirm
  `RAG_ENABLED=true`, and confirm `nomic-embed-text` is pulled in Ollama.
