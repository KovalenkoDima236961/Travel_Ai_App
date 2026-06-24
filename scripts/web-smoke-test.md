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
   - startDate: `2026-08-10`
   - days: `2`
   - budget: `500 EUR`
   - travelers: `2`
   - interests: `food`, `history`, `hidden_gems`
   - pace: `balanced`
9. Confirm the app redirects to the trip detail page.
10. Confirm the `Weather context` card appears before the itinerary area.
11. Confirm it shows `Provider: mock`, `Mock forecast for local development`,
    forecast days, temperatures, rain chance, wind speed, and any warning badges.
12. Click `Generate itinerary`.
13. Wait for completion.
14. Confirm the itinerary appears.
15. If any generated items show `Auto-matched place`, confirm they also show a
    place address/provider and, when confidence is present, a percentage.
16. If at least two generated auto-matched places have coordinates, confirm map
    markers and distance estimates appear before any manual place attachment.
17. Check the itinerary generally prefers local, budget-friendly, hidden-gem style suggestions and avoids nightclub-focused recommendations. Do not treat exact AI wording as part of the test.
18. Open `Version History`.
19. Confirm a `Generated` version exists.
20. Click `Edit itinerary`.
21. Open an itinerary item and click `Attach real place`.
22. Search for `Colosseum` with destination `Rome`.
23. Select `Colosseum`.
24. Confirm the item shows address, rating/category, and an `Open map` link.
25. Confirm manually changing or removing an auto-matched place clears the
    `Auto-matched place` label after saving.
26. Attach a second mock place with coordinates to another itinerary item.
27. Change one item name.
28. Add one item.
29. Remove one item.
30. Click `Save`.
31. Refresh the page.
32. Confirm the attached place address/rating/map link still appears.
33. Confirm Map View appears on the trip detail page.
34. Confirm map markers are visible.
35. Click a marker and confirm the popup shows item/place details.
36. Use the day filter and confirm markers change.
37. Refresh the page and confirm the map still shows markers.
38. Confirm the `Distance estimate` panel appears below the Map View.
39. Confirm the panel explains that route estimates come from the External
    Integrations Service and fall back to a straight-line Haversine estimate.
40. Confirm the day with at least two mapped places shows a mapped-stops count
    and, with the External Integrations Service running, a
    `Route estimate: <km> · ~<time> walking` line plus a smaller
    `Straight-line fallback: <km>` line. Exact figures depend on the attached
    places. (If the service is down, a `Straight-line estimate` line is shown
    instead.)
41. Expand the day's segment details and confirm per-segment distances appear
    (e.g. `Colosseum → Roman Forum: 0.6 km · ~8 min`).
42. Open `/settings`, set `maxWalkingKmPerDay` to a low value such as `1`, and
    save preferences.
43. Return to the trip detail page.
44. Confirm a day above the preference shows the `Above your walking preference`
    warning badge and the `Your preference: max 1 km/day` line.
45. Click `Edit itinerary` and confirm the distance estimates are hidden with a
    note that they are available after saving or leaving edit mode.
46. Leave edit mode and confirm the distance estimates reappear.
47. Open `Version History` again.
48. Confirm a `Manual edit` version exists.
49. Preview the manual edit version and confirm it keeps the place metadata.
50. Preview the older generated version.
51. Restore the generated version.
52. Refresh the page.
53. Confirm the restored itinerary persists.
54. Open `Version History`.
55. Confirm the restore created another version.
56. Go to `/trips`.
57. Confirm the trip appears in the list.

## Public Trip Sharing

1. Log in and open a completed trip.
2. Click `Create share link` in the `Share itinerary` panel.
3. Confirm a public link appears.
4. Set expiration to `7 days`.
5. Enable `Require password`, enter and confirm a password with at least 6 characters, then save settings.
6. Click `Copy link`.
7. Open the copied `/share/{shareToken}` link in an incognito/private browser.
8. Confirm the password form appears.
9. Enter a wrong password and confirm a generic error appears.
10. Enter the correct password and confirm the itinerary is visible without logging in.
11. Confirm map, place details, distance estimates, and weather context render when the trip has the needed data.
12. Download PDF and `.ics` from the unlocked public page.
13. Confirm edit, regenerate, place-review, and version-history controls are not visible.
14. Return to the owner tab, remove the password, and save settings.
15. Refresh the public link and confirm the trip loads without a password.
16. Return to the owner tab and click `Disable link`.
17. Refresh the public link.
18. Confirm `This shared trip is unavailable, expired, or disabled.` appears.

## Export v1

1. Log in.
2. Create or open a completed trip.
3. Click `Download PDF`.
4. Open the downloaded PDF and verify it shows destination, dates, itinerary
   days, item times, places, weather if loaded, and distance summary when
   available.
5. Click `Download calendar (.ics)`.
6. Import the `.ics` file into a calendar app or inspect the file contents.
7. Confirm events match itinerary item times.
8. Confirm untimed itinerary items are skipped in the `.ics` file.
9. Create a public share link.
10. Open the `/share/{shareToken}` link in an incognito/private browser.
11. If the link is password protected, unlock it first.
12. Download the PDF and `.ics` from the public page without logging in.
13. Confirm the public export works after unlock.
14. Confirm the public page and exports do not show edit, regenerate,
    place-review, version-history controls, user email, user ID, preferences,
    tokens, or private/internal metadata.

## Opening Hours

1. Log in and open a completed trip with a start date.
2. Click `Edit itinerary`.
3. Open an itinerary item and click `Attach real place`.
4. Search for `Colosseum` with destination `Rome`.
5. Select `Colosseum` and confirm the search result says opening hours are
   available.
6. Set the item time to `10:00` and click `Save`.
7. Confirm the read-only itinerary shows `Likely open at this time` and daily
   hours for the attached place.
8. Click `Edit itinerary`, change the same item time to `22:00`, and save.
9. Confirm the itinerary shows `May be closed at this time` and the `Opening
   hours warnings` summary lists the item.
10. Open `Version History`, preview the manual edit version, and confirm the
    restored/saved itinerary keeps the attached place `openingHours`.

## Weather Context

1. Log in.
2. Create a trip with destination `Rome`, start date `2026-08-10`, and `days=3`.
3. Open the trip detail page.
4. Confirm the `Weather context` card appears and renders three forecast days.
5. Confirm mock provider labeling and warning badges are visible when thresholds match.
6. Click `Generate itinerary` and confirm generation still completes.
7. Stop only the External Integrations Service:
   `docker compose -f infra/docker-compose.yml stop external-integrations-service`.
8. Refresh the trip detail page.
9. Confirm the weather card shows `Weather forecast unavailable.` and the page does not crash.
10. With `WEATHER_CONTEXT_FAIL_OPEN=true`, generate or partially regenerate an itinerary and confirm it still works.
11. Restart the service:
    `docker compose -f infra/docker-compose.yml start external-integrations-service`,
    refresh, and confirm the weather card returns.

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

## Route Estimate (External Integrations Service)

This verifies the service-backed route estimate and its straight-line fallback.

1. Log in.
2. Open a completed trip.
3. Click `Edit itinerary`.
4. Attach mock places with coordinates to at least two items in one day.
5. Click `Save`.
6. Open the trip detail page.
7. In the `Distance estimate` panel, confirm the day shows a
   `Route estimate: <km> · ~<time> walking` line and a
   `Route estimates by mock provider` badge.
8. Confirm the smaller `Straight-line fallback: <km>` line is still shown.
9. Expand the segment details and confirm they are labelled `(route)` with
   per-segment distance and time.
10. Stop only the External Integrations Service:
    `docker compose -f infra/docker-compose.yml stop external-integrations-service`.
11. Refresh the trip detail page.
12. Confirm the app does not crash and the panel falls back to
    `Route service unavailable. Showing straight-line estimate.` with the
    straight-line Haversine figures (badge shows `Straight-line fallback estimate`).
13. Confirm the walking-preference warning still works, now compared against the
    straight-line estimate (the line ends with `(straight-line estimate)`).
14. Restart the service:
    `docker compose -f infra/docker-compose.yml start external-integrations-service`,
    refresh, and confirm the route estimate returns.

## AI Quality Feedback Loop

1. Log in.
2. Create or open a completed trip with a generated itinerary.
3. Attach places with coordinates to create a high walking-distance day, or open
   `/settings`, set `maxWalkingKmPerDay` to a low value such as `1`, save, and
   return to the trip.
4. Confirm the `Trip Quality Checks` card appears after `Weather context`.
5. Confirm the card shows a walking-distance warning for the affected day.
6. Click `Improve day`.
7. Confirm regeneration runs, the trip updates, and the success message appears.
8. Attach a place with opening hours to an itinerary item, set the item time
   outside those hours, and save.
9. Confirm the `Trip Quality Checks` card shows `Place may be closed`.
10. Click `Improve item` for that item.
11. Confirm item regeneration runs and the trip updates.
12. If the itinerary has pending or low-confidence auto-matches, confirm the
    card shows place-match checks and review-only items point to `Place Matches`.
13. Click `Edit itinerary`.
14. Confirm the quality card remains advisory and says to save or cancel edits
    before improving with AI.
15. Open `Version History`.
16. Confirm regenerated day/item changes created versions.

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
  The Distance estimate panel falls back to straight-line estimates and does not
  crash the page.
- Route estimate CORS error in browser console: confirm the External
  Integrations Service allows `POST` (default `CORS_ALLOWED_METHODS=GET,POST,OPTIONS`),
  then rebuild/restart `external-integrations-service`.
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
