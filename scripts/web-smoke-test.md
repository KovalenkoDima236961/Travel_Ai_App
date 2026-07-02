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

RabbitMQ management UI:

```text
http://localhost:15672
```

Use local credentials `guest` / `guest`.

Prometheus and Grafana:

```text
http://localhost:9090
http://localhost:3001
```

Use local Grafana credentials `admin` / `admin`.

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
13. Confirm a generation status card appears quickly and the page remains responsive.
14. In RabbitMQ, confirm `trip.generation.jobs` briefly receives and consumes a
    message.
15. Wait for the card to show completion.
16. Confirm the itinerary appears.
17. If any generated items show `Auto-matched place`, confirm they also show a
    place address/provider and, when confidence is present, a percentage.
18. If at least two generated auto-matched places have coordinates, confirm map
    markers and distance estimates appear before any manual place attachment.
19. Check the itinerary generally prefers local, budget-friendly, hidden-gem style suggestions and avoids nightclub-focused recommendations. Do not treat exact AI wording as part of the test.

## Queue Worker Recovery

1. With the stack running, stop only the worker:

   ```bash
   docker compose -f infra/docker-compose.yml --env-file infra/.env stop worker-service
   ```

2. Create another generation or regeneration job from the Web App.
3. Confirm the status card remains `queued`.
4. In RabbitMQ, confirm a message remains in `trip.generation.jobs`.
5. Restart the worker:

   ```bash
   docker compose -f infra/docker-compose.yml --env-file infra/.env start worker-service
   ```

6. Confirm the message is consumed and the Web App status card completes.
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

## Observability

1. Open Prometheus at `http://localhost:9090`.
2. Go to `Status` -> `Target health` and confirm the app services and RabbitMQ
   are `UP`.
3. Open Grafana at `http://localhost:3001`.
4. Log in with `admin` / `admin`.
5. Open the `Worker Jobs` dashboard.
6. Trigger itinerary generation from the Web App.
7. Confirm `worker_jobs_started_total` and either
   `worker_jobs_completed_total` or `worker_jobs_failed_total` change.
8. Open the `External Providers` dashboard.
9. Trigger route, weather, or price calls from the trip page.
10. Confirm provider request/cache/fallback metrics change.
11. Check Trip Service and Worker Service logs for the same `correlationId`
    across job creation, RabbitMQ publish, message consume, and job completion.

## Budget Tracking

1. Create a trip with budget `EUR 700` and generate its itinerary (mock AI is
   fine).
2. On the trip detail page, confirm the `Budget` panel appears in the sidebar.
3. Confirm it shows the trip budget, an estimated total, and remaining amount,
   plus daily and category totals.
4. Confirm itinerary item cards show cost badges (for example `€18 ticket`), with
   `(approx.)` on low-confidence costs and `provider estimate` on provider-filled
   ticket/activity costs when price enrichment matched.
5. Click `Edit` in the `Budget` panel, change the amount, and `Save budget`.
6. Confirm the panel refreshes with the new budget and the itinerary revision is
   unchanged (no conflict warning, version history not affected).
7. Open the itinerary editor, expand an item's `Estimated cost`, change the
   amount/category, set currency to `JPY`, and save the itinerary.
8. Confirm the saved cost shows a `manual` marker and the budget panel shows an
   approximate converted total plus `JPY` under original currency totals.
9. If you intentionally enter an unsupported currency code through API/devtools,
   confirm the Budget panel shows a conversion warning and Trip Quality Checks
   reports a conversion issue without inventing an exchange rate.
10. Edit the budget to an amount **below** the estimated total and save.
11. Confirm the `Budget` panel switches to an over-budget warning style.
12. Open `Trip Quality Checks` and confirm budget issues appear (over budget,
    day over budget, expensive item, missing cost estimates, missing ticket
    prices, high ticket costs, or low-confidence provider estimates).
13. Click `Optimize Day N for budget` from the Budget panel or a budget-related
    Trip Quality Checks issue.
14. In the dialog, confirm the day, target reduction, currency, constraints, max
    walking increase, and optional instruction are prefilled with reasonable
    defaults.
15. Start the optimization job and confirm the shared generation job status card
    appears.
16. Wait for the proposal card to appear.
17. Confirm the proposal shows status, approximate savings, base/proposed day
    totals, confidence, changes, tradeoffs/warnings, and a `Preview day` button.
18. Preview the proposal and confirm it shows current and proposed day items.
19. Click `Apply`.
20. Confirm the itinerary day changes, the budget summary refetches, and version
    history/activity include budget optimization entries.
21. Start a second optimization proposal and click `Discard`; confirm the
    itinerary does not change.
22. Log in as an accepted viewer collaborator and confirm proposals are visible
    but create/apply/discard controls are not available.
23. Export a private PDF and confirm the budget summary uses approximate
    converted totals and includes original currency totals when present.
24. Create a public share link, open `/share/{shareToken}` in a logged-out
    window, and confirm the private trip budget, optimization UI/proposals, and
    provider price review metadata are **not** shown (item cost badges may still
    appear).

## Accommodation Planning

1. Open a private trip as owner or editor.
2. Confirm the sidebar shows the `Accommodation` panel with `No accommodation added yet.`
3. Click `Add stay`.
4. Enter:
   - name: `Hotel Roma`
   - type: `hotel`
   - address: `Via Roma 10`
   - check-in/check-out dates matching the trip
   - estimated stay cost: `120 EUR`
   - notes: `Near Termini`
5. Save and confirm the panel shows the stay details and estimated cost.
6. Click `Edit`, then `Attach place`.
7. Search for `Hotel Roma` or `hotel Rome`, select a result, and confirm the
   name/address/place fields update.
8. Save and confirm the Map View includes an accommodation marker labelled
   `Accommodation`.
9. Confirm the `Distance estimate` panel shows `Includes stay`, and segment
   details include accommodation-to-first-stop and last-stop-to-accommodation
   legs when itinerary items have coordinates.
10. Confirm the `Budget` panel estimated total increases by the accommodation
    cost and includes an `accommodation` category row.
11. Generate or regenerate a day and confirm the new plan is practical around
    the stay location.
12. Export a private PDF and confirm it includes an `Accommodation` section.
13. Create/open a public share link and confirm no separate accommodation panel
    or structured accommodation section is shown.
14. Log in as an accepted viewer collaborator and confirm the viewer can see the
    accommodation but cannot edit or remove it.
15. As owner/editor, remove the accommodation and confirm the panel returns to
    `No accommodation added yet.` and the budget total no longer includes the
    stay cost.

## Collaborative Planning

1. Create two registered accounts: an owner and a collaborator.
2. Log in as the owner.
3. Create and generate a trip, or open an existing completed owner trip.
4. In the `Collaborators` panel, invite the collaborator account by exact email
   as `viewer`.
5. Log out and log in as the collaborator.
6. Open `/trips`.
7. Confirm the invitation appears under `Pending invitations`.
8. Accept the invitation.
9. Confirm the trip appears under `Shared with me` with a `Viewer` role.
10. Open the shared trip.
11. Confirm itinerary, map, weather, distance, export, and version preview are
    visible.
12. Confirm edit, regenerate, place-review action buttons, route optimization
    apply controls, restore buttons, share controls, and collaborator controls
    are hidden.
13. Log back in as the owner.
14. Change the collaborator role to `editor`.
15. Log in as the collaborator again.
16. Open the shared trip and confirm edit/regenerate controls are visible.
17. Make a small itinerary edit and save.
18. Open version history and confirm a new version exists.
19. Confirm share controls and collaborator controls are still hidden for the
    editor.
20. Open the same trip in two browsers as owner/editor or as the same user.
21. In browser A, click `Edit itinerary` and leave the draft open.
22. In browser B, regenerate a day or save a small itinerary edit.
23. Return to browser A and click `Save`.
24. Confirm the conflict warning appears:
    `This itinerary changed while you were editing`.
25. Click `Reload latest`.
26. Confirm the latest itinerary appears and the old draft is discarded.
27. Repeat the conflict check with `Cancel my changes` and confirm edit mode
    exits without forcing an overwrite.
28. Log back in as owner and remove the collaborator.
29. Log in as collaborator again and confirm the trip no longer opens and no
    longer appears under `Shared with me`.

## Google Calendar Sync

1. Start the full stack with `CALENDAR_PROVIDER=mock` for local testing, or set
   real Google OAuth credentials with `CALENDAR_PROVIDER=google`.
2. Log in as an owner and open a completed private trip with timed itinerary
   items.
3. Confirm the `Calendar sync` panel appears.
4. Click `Connect Google Calendar`.
5. Complete OAuth. With the mock provider this redirects immediately; with real
   Google, grant the `calendar.events` scope.
6. Confirm you return to the trip detail page with a connected account shown.
7. Click `Sync itinerary`.
8. Confirm the panel shows a sync summary and the sync status is no longer
   `Not synced`.
9. If using real Google, confirm events appear on the primary Google Calendar.
10. Modify the itinerary and save it.
11. Confirm the calendar sync panel reports the synced events are out of date.
12. Click `Update synced events`.
13. If using real Google, confirm the existing Google events update rather than
    duplicating.
14. Click `Remove synced events` and confirm.
15. If using real Google, confirm the events created by this app are removed.
16. Confirm PDF and `.ics` export still work.
17. Invite a collaborator as `viewer`, accept the invite, and confirm the viewer
    does not see sync actions.
18. Open the public `/share/{shareToken}` page and confirm calendar sync is not
    shown.

## Itinerary Comments

1. Log in as the owner and open a completed trip with an itinerary.
2. In the read-only itinerary, confirm each item shows a `Comments` button.
3. Click `Comments` on the first itinerary item.
4. Confirm the panel opens and shows `No comments yet.`
5. Type a comment and click `Post`.
6. Confirm the comment appears, labelled `You`, with `Edit` and `Delete` actions.
7. Close the panel and confirm the item's `Comments` button now shows a count of `1`.
8. Confirm the `comments across itinerary items` summary appears above the itinerary.
9. Invite a second account as `viewer` (or `editor`) and accept the invitation as
   that user (see `Collaborative Planning`).
10. As the collaborator, open the shared trip and open comments on the same item.
11. Confirm the owner's comment is visible, labelled `Collaborator`, with no
    `Edit`/`Delete` actions.
12. Add a comment as the collaborator, then `Edit` it and confirm it shows
    `edited`.
13. Confirm the collaborator's own comment shows `Edit` and `Delete`, but the
    owner's comment does not.
14. Log back in as the owner, open the same item, and confirm the owner can
    `Delete` the collaborator's comment (owner can delete any comment).
15. Confirm the deleted comment disappears and the count decreases.
16. Click `Edit itinerary` and confirm the `Comments` buttons are hidden while
    editing; leave edit mode and confirm they reappear.
17. Open the public `/share/{shareToken}` link (see `Public Trip Sharing`) in an
    incognito window and confirm no `Comments` buttons, comment panels, or comment
    summary appear, and that no comment requests are made.

## Real-time Trip Presence

1. Start the full stack.
2. Create owner and collaborator accounts.
3. As the owner, create and generate a trip.
4. Invite the collaborator as `editor`.
5. Accept the invitation as the collaborator.
6. Open the same private trip in two browsers.
7. Confirm each user sees the other in `Currently here`.
8. As the owner, click `Edit itinerary`.
9. Confirm the collaborator sees an editing warning for the owner.
10. Cancel or save the owner edit.
11. Confirm the collaborator sees the owner return to `viewing`.
12. Close one browser tab.
13. Confirm the other user disappears from the presence list.
14. Open the public `/share/{shareToken}` link.
15. Confirm presence is not shown and no presence requests are made.

## Soft Edit Locks

1. Log in as an owner in browser A.
2. Log in as an accepted `editor` collaborator in browser B.
3. Open the same completed private trip in both browsers.
4. In browser A, click `Edit itinerary`.
5. Confirm browser A enters edit mode and the lock status says
   `You are editing this itinerary`.
6. In browser B, confirm the lock status says another collaborator is editing.
7. In browser B, click `Edit itinerary`.
8. Confirm the `Someone is already editing` warning appears.
9. Click `Cancel` and confirm browser B stays out of edit mode.
10. In browser A, click `Cancel` or save the itinerary.
11. In browser B, click `Edit itinerary` again and confirm no warning appears.
12. Repeat the flow, but click `Continue anyway` in browser B.
13. Confirm both users can enter edit mode, presence can show multiple editors,
    and a stale save still shows the itinerary conflict warning.
14. Open the public `/share/{shareToken}` page and confirm no edit-lock status
    or edit-lock requests appear.

## Notification Preferences

1. Log in.
2. Open `/settings`.
3. Confirm `Notification preferences` appears below the profile and travel
   preference sections.
4. Disable `Email notifications` → `Comments` and save.
5. As a collaborator, create a comment on one of your trips.
6. Confirm the in-app `comment_created` notification still appears when
   `In-app notifications` → `Comments` remains enabled.
7. With the default mock email provider, confirm Notification Service logs do not
   show a mock comment email for that recipient after email comments are disabled.
8. Disable `In-app notifications` → `Comments` and save.
9. Create another comment from a collaborator.
10. Confirm no new in-app `comment_created` notification appears.
11. Re-enable both comment preferences and save.

## Real-time Notifications

1. Start the full stack and open `http://localhost:3000`.
2. Log in as a trip owner in one browser.
3. Log in as an accepted collaborator in a second browser.
4. Keep the authenticated header visible in both browsers so the notification
   bell is mounted.
5. As the owner, invite the collaborator, or as the collaborator, create a
   comment on the shared trip.
6. Confirm the other user's notification badge updates without a manual refresh.
7. Open the notification dropdown and confirm the new notification appears.
8. Stop only the Notification Service, or set `NOTIFICATION_SSE_ENABLED=false`
   and restart it.
9. Refresh the web app and confirm the notification UI still works through the
   polling fallback, even though real-time updates are unavailable.

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

## Real Routing + Weather Providers

This verifies real provider opt-in, fallback-to-mock, and the safe error path. The
Web App is not changed — it keeps calling `POST /routes/estimate` and
`GET /weather/forecast`. Set keys only in `infra/.env` (git-ignored); never commit
them.

1. Start the stack with the defaults (`ROUTE_PROVIDER=mock`,
   `WEATHER_PROVIDER=mock`).
2. Open a trip that has places with coordinates.
3. Confirm route estimates and the weather context render with `mock` provider
   labels.
4. Set `ROUTE_PROVIDER=ors` and `ORS_API_KEY=<your key>` in `infra/.env`.
5. Restart External Integrations Service:
   `docker compose -f infra/docker-compose.yml up -d --build external-integrations-service`.
6. Refresh the trip and confirm route estimates still appear (the response
   `provider` is now `ors`, or `mock` with `fallbackUsed` when ORS is unreachable).
7. Set `WEATHER_PROVIDER=openweathermap` and `OPENWEATHER_API_KEY=<your key>`,
   restart the service, and confirm the weather forecast still appears for a
   near-term trip date.
8. Temporarily break a key (e.g. set `ORS_API_KEY=invalid`) with
   `ROUTE_PROVIDER_FALLBACK_TO_MOCK=true`, restart, and confirm the app still
   works using the mock fallback (route estimates still render).
9. Set `ROUTE_PROVIDER_FALLBACK_TO_MOCK=false`, keep the invalid key, restart, and
   confirm the distance panel surfaces the route service as unavailable and the
   page does not crash (the endpoint returns
   `502 {"error":"route_provider_unavailable"}`).
10. Restore `ROUTE_PROVIDER=mock` / `WEATHER_PROVIDER=mock` (or valid keys with
    fallback enabled) and confirm normal behavior returns.

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

## Recent Activity (Activity Feed)

The activity feed records important successful actions on a private trip and is
visible only to the owner and accepted collaborators. It never appears on the
public share page. Private trip detail pages also open a fetch-based SSE stream
so newly persisted activity can refetch into the feed without a page refresh.

1. Log in as the trip owner.
2. Create a trip (e.g. destination `Rome`, `days=3`) and open its detail page.
3. Generate an itinerary.
4. Add a comment on an itinerary item.
5. Open `Share & Collaborators`, invite a collaborator, and update share
   settings (e.g. set a password or expiration).
6. Scroll to the `Recent activity` panel at the bottom of the page.
7. Confirm events appear grouped by day (`Today`/`Yesterday`/date), newest
   first, with readable text such as:
   - `You created the trip`
   - `You generated the itinerary`
   - `You commented on Day 2 · <item name>`
   - `You invited anna@example.com as editor`
   - `You updated share settings`
8. Confirm timestamps render and, if there are more than 30 events, a
   `Load more` button fetches older events.
9. Accept the invitation as the collaborator, then log in as that collaborator
   and open the shared trip.
10. Confirm the `Recent activity` panel is visible and the owner's actions show
    as `Collaborator` (not `You`); the collaborator's own actions show as `You`.
11. Keep the owner trip page open in browser A and the collaborator private trip
    page open in browser B. Add a comment in browser A and confirm browser B's
    `Recent activity` panel updates without refresh.
12. Update the trip budget or accommodation in browser B and confirm browser A's
    activity feed updates without refresh.
13. Trigger generation job completion or failure if practical and confirm the
    corresponding activity appears live.
14. Open the public share link (`/share/<shareToken>`) in a separate
    session/browser.
15. Confirm there is no activity panel on the public page and no live private
    activity stream data is exposed.

## Notifications (Notification Service)

In-app notifications are private, per-user data served by the Notification
Service. The header shows a bell with an unread badge for logged-in users; the
badge is polled (~every 45s) and refreshes immediately after you act on
notifications. The bell never appears for logged-out / public share viewers.

Use two browsers (or one normal + one private window) so two users are logged in
at once: the trip **owner** and a **second user** (collaborator).

1. Start the full stack (`docker compose -f infra/docker-compose.yml up --build`).
2. In browser A, log in as the owner and create a trip (e.g. `Rome`, `days=3`).
   Generate the itinerary and open its detail page.
3. In browser B, register/log in as a second user. Confirm the bell shows **no**
   unread badge yet.
4. In browser A, open `Share & Collaborators` and invite the second user (by the
   email they registered with) as `viewer` or `editor`.
5. In browser B, within ~45s (or reload), confirm the bell shows an unread
   badge. Open the dropdown and confirm an invitation notification:
   - `You were invited to collaborate on a trip`.
6. Click the invitation notification. Confirm it is marked read (badge
   decreases) and you navigate to `/trips` (where the invitation can be
   accepted). Accept the invitation.
7. In browser A (owner), confirm the bell shows an unread badge and the dropdown
   shows `Collaboration invitation accepted`.
8. In browser B (collaborator), open the shared trip and add a comment on an
   itinerary item.
9. In browser A (owner), confirm a new `New comment` notification appears.
   Click it and confirm you navigate to the trip detail page (`/trips/{id}`).
10. In the owner's dropdown, click `Mark all as read` and confirm the unread
    badge clears. Open `/notifications` (View all) and confirm the full list
    renders with `Load more` when there are more than 30 items.
11. Confirm the actor never notifies themselves: the comment author (browser B)
    does **not** receive their own `New comment` notification.
12. Open the public share link (`/share/<shareToken>`) in a logged-out session
    and confirm there is **no** notification bell.

## Troubleshooting

- CORS error in browser console: confirm Trip Service has
  `CORS_ALLOWED_ORIGINS=http://localhost:3000`, then rebuild/restart
  `trip-service`.
- Notification bell missing or badge not updating: confirm `notification-service`
  is healthy (`docker compose -f infra/docker-compose.yml ps`) and that
  `NEXT_PUBLIC_NOTIFICATION_SERVICE_URL` is reachable from the browser. The
  unread count polls about every 45 seconds; reload to force a refresh.
- Trip Service offline: check `docker compose -f infra/docker-compose.yml ps`
  and `docker compose -f infra/docker-compose.yml logs trip-service`.
- User Service offline: check `docker compose -f infra/docker-compose.yml ps`
  and `docker compose -f infra/docker-compose.yml logs user-service`.
- External Integrations Service offline: check
  `docker compose -f infra/docker-compose.yml logs external-integrations-service`.
  The Distance estimate panel falls back to straight-line estimates and does not
  crash the page.
- External Integrations CORS error in browser console: confirm the External
  Integrations Service allows the web methods it needs (default
  `CORS_ALLOWED_METHODS=GET,POST,DELETE,OPTIONS`), then rebuild/restart
  `external-integrations-service`.
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
