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
33. Confirm the `Distance estimate` panel appears below the Map View.
34. Confirm the panel is labelled as approximate straight-line distance.
35. Confirm the day with at least two mapped places shows a mapped-stops count,
    an approximate distance (e.g. `approx. 1.6 km`), and an estimated walking
    time (e.g. `~19 min walking`). Exact figures depend on the attached places.
36. Expand the day's segment details and confirm per-segment distances appear
    (e.g. `Colosseum → Roman Forum: 0.6 km · ~8 min`).
37. Open `/settings`, set `maxWalkingKmPerDay` to a low value such as `1`, and
    save preferences.
38. Return to the trip detail page.
39. Confirm a day above the preference shows the `Above your walking preference`
    warning badge and the `Your preference: max 1 km/day` line.
40. Click `Edit itinerary` and confirm the distance estimates are hidden with a
    note that they are available after saving or leaving edit mode.
41. Leave edit mode and confirm the distance estimates reappear.
42. Open `Version History` again.
43. Confirm a `Manual edit` version exists.
44. Preview the manual edit version and confirm it keeps the place metadata.
45. Preview the older generated version.
46. Restore the generated version.
47. Refresh the page.
48. Confirm the restored itinerary persists.
49. Open `Version History`.
50. Confirm the restore created another version.
51. Go to `/trips`.
52. Confirm the trip appears in the list.

## Route Optimization

1. Log in and open a completed trip.
2. Click `Edit itinerary`.
3. Attach real/mock places with coordinates to at least three items in one day.
4. Click `Save`.
5. Confirm the `Distance estimate` panel appears for that day.
6. Confirm the day shows an `Optimize order` button (it appears only when the day
   has at least three mapped places and you are not in edit mode).
7. Click `Optimize order` for that day.
8. Confirm the dialog shows the current order and the suggested order side by
   side, and that it is labelled as approximate straight-line distance.
9. Confirm the distance comparison shows `Original`, `Optimized`, and
   `Estimated saving` (km and walking minutes).
10. Click `Apply optimized order`.
11. Confirm the dialog closes and a success message appears.
12. Refresh the page.
13. Confirm the new order persists.
14. Open `Version History`.
15. Confirm a `Manual edit` version exists for this change.

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
