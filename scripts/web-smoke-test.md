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
2. Register or log in.
3. Open `http://localhost:3000/settings`.
4. Set profile:
   - displayName: `Test Traveler`
   - homeCity: `Bratislava`
   - homeCountry: `Slovakia`
   - preferredCurrency: `EUR`
   - preferredLanguage: `en`
5. Set preferences:
   - travelStyles: `budget`, `food`, `hidden_gems`
   - pace: `balanced`
   - maxWalkingKmPerDay: `8`
   - foodPreferences: `local`, `cheap`
   - avoid: `nightclubs`
   - preferredTransport: `walking`, `public_transport`
6. Save profile and preferences.
7. Open the create trip page.
8. Create a trip with:
   - destination: `Rome`
   - days: `2`
   - budget: `500 EUR`
   - travelers: `2`
   - interests: `food`, `history`, `hidden_gems`
   - pace: `balanced`
9. Confirm the app redirects to the trip detail page.
10. Click `Generate itinerary`.
11. Wait for completion.
12. Confirm the itinerary appears.
13. Check the itinerary generally prefers local, budget-friendly, hidden-gem style suggestions and avoids nightclub-focused recommendations. Do not treat exact AI wording as part of the test.
14. Open `Version History`.
15. Confirm a `Generated` version exists.
16. Click `Edit itinerary`.
17. Open an itinerary item and click `Attach real place`.
18. Search for `Colosseum` with destination `Rome`.
19. Select `Colosseum`.
20. Confirm the item shows address, rating/category, and an `Open map` link.
21. Attach a second mock place with coordinates to another itinerary item.
22. Change one item name.
23. Add one item.
24. Remove one item.
25. Click `Save`.
26. Refresh the page.
27. Confirm the attached place address/rating/map link still appears.
28. Confirm Map View appears on the trip detail page.
29. Confirm map markers are visible.
30. Click a marker and confirm the popup shows item/place details.
31. Use the day filter and confirm markers change.
32. Refresh the page and confirm the map still shows markers.
33. Open `Version History` again.
34. Confirm a `Manual edit` version exists.
35. Preview the manual edit version and confirm it keeps the place metadata.
36. Preview the older generated version.
37. Restore the generated version.
38. Refresh the page.
39. Confirm the restored itinerary persists.
40. Open `Version History`.
41. Confirm the restore created another version.
42. Go to `/trips`.
43. Confirm the trip appears in the list.

## Troubleshooting

- CORS error in browser console: confirm Trip Service has
  `CORS_ALLOWED_ORIGINS=http://localhost:3000`, then rebuild/restart
  `trip-service`.
- Trip Service offline: check `docker compose -f infra/docker-compose.yml ps`
  and `docker compose -f infra/docker-compose.yml logs trip-service`.
- User Service offline: check `docker compose -f infra/docker-compose.yml ps`
  and `docker compose -f infra/docker-compose.yml logs user-service`.
- External Integrations Service offline: check
  `docker compose -f infra/docker-compose.yml logs external-integrations-service`.
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
