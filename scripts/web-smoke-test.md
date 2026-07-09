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
7. In the language selector, choose Ukrainian and confirm the settings and
   navigation labels update immediately.
8. Refresh the page and confirm Ukrainian remains selected.
9. Switch to French and confirm the UI changes without changing the URL.
10. Switch back to English before continuing with the baseline flow.
11. Open the create trip page.
12. Create a trip with:
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
19. On a ticketed attraction/activity/event item, click `Check availability`.
20. Confirm the card shows a provider **badge** (Ticketmaster, Mock, or
    "Fallback estimate"), a top-level status, a "Checked N minutes ago" label, a
    High/Medium/Low confidence label, and — when available — an option with a
    price (note the `From`/`Est.` qualifier), venue, date/start times, and a
    `View on provider` link. Click that link and confirm it opens the provider
    site in a **new tab**; no in-app checkout appears.
21. If the item already has an estimate, confirm the card shows the provider-vs-
    current price difference and warns when the provider price is notably higher.
22. Click `Apply price estimate` on a confident option and confirm the budget
    summary and cost analytics refresh after the item cost is saved. If the item
    had a cost-split rule, confirm the split is preserved.
23. For a low-confidence / unmatched result, confirm the card shows a
    "Possible match" warning and a `Verify to apply` hint instead of the apply
    button (medium-confidence apply prompts for confirmation first).
24. On a non-bookable item (rest/walk/note), confirm no availability card or a
    "not needed" state is shown — no error.
25. Open the workspace trip approval panel after applying an availability price
    and confirm the checklist reflects the checked item; if the applied match was
    low-confidence, the price changed, or fallback data was used, confirm the
    corresponding availability warning/info rows appear (they do not block
    submission).
26. To test fallback: unset `TICKETMASTER_API_KEY` (or force a provider error)
    with `AVAILABILITY_FALLBACK_TO_MOCK=true` and confirm the card shows a
    "Fallback estimate" badge and a clear not-verified warning.
27. Check the itinerary generally prefers local, budget-friendly, hidden-gem style suggestions and avoids nightclub-focused recommendations. Do not treat exact AI wording as part of the test.

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

## Workspace Shared Budget Flow

1. Log in as a workspace owner or admin.
2. Open `/workspaces` and create or open a workspace.
3. Create at least one workspace trip with item or accommodation costs.
4. Open `/workspaces/{workspaceId}/budgets`.
5. Click `Create budget`.
6. Enter:
   - name: `Smoke shared budget`
   - amount: `100`
   - currency: `EUR`
   - periodStart: `2026-01-01`
   - periodEnd: `2026-12-31`
   - isPrimary: checked
7. Confirm the budget appears with a utilization preview.
8. Open the budget detail page.
9. Confirm summary cards, utilization bar, cost by trip/category/source,
   expensive items, insights, and warnings render.
10. Download CSV and PDF.
11. Edit the budget amount lower than the estimated total.
12. Confirm the over-budget state and insight appear.
13. Open `/workspaces/{workspaceId}/analytics`.
14. Confirm the primary budget card appears.
15. Click `Use budget period` and confirm the date filters match the budget
    period.
16. Log in as a workspace viewer.
17. Confirm the viewer can open the budget list and detail page.
18. Confirm create, edit, make-primary, and archive controls are not visible.

## AI Policy-Aware Trip Repair Flow

1. Log in as a workspace owner/admin/member with trip edit access.
2. Create or open a workspace trip with a completed itinerary.
3. Add or enable a workspace policy that produces a warning or blocking result,
   such as a low max trip budget, daily budget, max item cost, late-activity
   rule, walking limit, or required rest rule.
4. Open the trip detail page and confirm the header/approval area shows approval
   risk reasons.
5. Click `Repair with AI` from the AI Repair Proposals panel or a
   `repair_with_ai` risk suggested action.
6. Select a repair mode, confirm high-severity issues are selected when using
   `Selected issues`, set max changed items, and submit.
7. Confirm an AI repair job status card appears and eventually completes.
8. Confirm a pending repair proposal appears with repair mode, status, changed
   item counts, risk before/after if available, cost before/after if available,
   addressed issues, and warnings.
9. Click `Preview repair` and confirm grouped added/removed/modified/moved
   changes plus side-by-side current and repaired itinerary days render.
10. Click `Apply repair`, confirm the itinerary changes, and verify the trip is
   refetched with a higher `itineraryRevision`.
11. Open version history and confirm a new AI policy repair version exists.
12. Check activity and confirm a repair-applied event exists without full
   itinerary JSON.
13. If the trip was approved or pending approval, confirm approval returned to
   draft and needs review again.
14. Generate another repair proposal, manually edit and save the itinerary, then
   try applying the old proposal. Confirm the UI shows a stale/conflict message
   and asks for a new repair proposal.
15. Generate a new repair proposal and click `Discard`; confirm the proposal is
   removed from the pending list and the itinerary does not change.

## Trip Template Flow

1. Create or open a completed trip with itinerary items and costs.
2. Click `Save as template`.
3. Save it as a private template.
4. Open `/templates` and confirm the template appears with private visibility,
   destination, duration, estimate, and tags.
5. Open the template detail page and confirm the itinerary preview uses day
   offsets instead of fixed dates.
6. Click `Use template`.
7. Enter a new destination and start date.
8. Create the trip and confirm the new trip is completed with copied itinerary
   items and the new start date.
9. Save a workspace trip as a workspace template.
10. Open `/workspaces/{workspaceId}/templates` and confirm members can see it.
11. Confirm a workspace viewer can view templates but cannot create a workspace
    trip from one.
12. Archive a template and confirm it disappears from active template lists.

Template limitations: live availability, booking links, provider snapshots,
comments, collaborators, public share links, and calendar sync metadata are not
copied. Prices are approximate and should be verified.

## AI Template Adaptation Flow

1. Open a template detail page (or a template library card) that you can use.
2. Confirm both `Use template directly` and `Adapt with AI` actions are shown.
   For a workspace template where you are only a viewer, confirm `Adapt with AI`
   is hidden.
3. Click `Adapt with AI`.
4. Enter a new destination, start date, duration, budget, pace, and a couple of
   interests. Optionally add special instructions. Leave the
   `deterministic copy` fallback checkbox enabled.
5. If a workspace is selected in the switcher, confirm the scope defaults to that
   workspace and that viewer-only workspaces are not selectable.
6. Submit and watch the inline job status advance (Queued → Adapting → Completed).
7. On completion, confirm the summary lists the major changes and warnings, then
   click `Open trip`.
8. On the created trip, confirm:
   - the banner says the trip was created by AI adapting a template,
   - the destination, dates, and duration match what you requested,
   - availability cards show an unchecked status (nothing was auto-checked),
   - for a workspace trip, the approval status is `draft` with a
     `Review approval checklist` call to action (nothing is auto-submitted).
9. To exercise fallback, run the flow with AI in mock mode (deterministic) or,
   if available, a mock failure mode; when fallback is used the completion screen
   and trip banner both say a deterministic template copy was created instead.

AI adaptation limitations: the result is a reviewable draft, not a confirmed
plan. Costs are estimates, availability and opening hours must be checked,
booking is never automatic, and substitutions can be imperfect.

## Collaborative Merge Recovery

Manual test safe merge:

1. Log in as user A in browser A.
2. Log in as user B in browser B and open the same shared trip as an editor.
3. Browser A clicks `Edit itinerary` at revision N.
4. Browser B edits Day 1 and saves revision N+1.
5. Browser A edits Day 3 and clicks `Save`.
6. Confirm Browser A sees `This itinerary changed while you were editing`.
7. Confirm the dialog offers `Apply safe merge`.
8. Apply the safe merge.
9. Confirm the final itinerary contains Browser B's Day 1 change and Browser A's
   Day 3 change.

Manual test conflict:

1. Browser A enters edit mode.
2. Browser B edits Day 2 item 1 and saves.
3. Browser A edits Day 2 item 1 and saves.
4. Confirm the conflict dialog shows overlapping local and latest changes.
5. Choose `Keep latest` and apply; confirm the remote item remains.
6. Repeat and choose `Keep mine`; confirm the local item replaces the remote item.

Manual test discard:

1. Browser A gets a stale-save conflict.
2. Click `Discard my changes`.
3. Confirm the latest itinerary is shown and the local draft is discarded.

## Offline Trip Mode

Manual offline cache and sync test:

1. Start the full stack.
2. Log in.
3. Open a private trip online and confirm it loads.
4. Confirm the trip has an itinerary. Generate one first if needed.
5. Turn off network in browser devtools.
6. Reload the trip detail URL.
7. Confirm the cached trip appears with the offline banner and saved timestamp.
8. Click `Edit itinerary`.
9. Modify one itinerary item.
10. Click `Save`.
11. Confirm the pending offline change indicator appears.
12. Turn network back on.
13. Click `Sync now` or wait for auto-sync.
14. Confirm the pending indicator disappears and the change persists after refresh.

Manual offline conflict test:

1. Open the same private trip in browser A and browser B.
2. Browser A loads the trip online, then goes offline.
3. Browser A edits Day 2 and saves offline.
4. Browser B stays online, edits the same Day 2 item, and saves.
5. Browser A goes online and syncs.
6. Confirm the merge/conflict dialog opens with offline recovery copy.
7. Choose either latest or local resolution and apply.
8. Confirm the final itinerary matches the selected resolution.

Manual no-cache test:

1. Clear browser site data.
2. Go offline.
3. Open a private trip detail URL.
4. Confirm the page says the trip is not available offline yet.

## PWA Install And Offline Trips

Manual Chromium install test:

1. Start the app and log in with a supported Chromium-family browser.
2. Open `/trips`, then open a trip detail page successfully.
3. Wait at least 10 seconds.
4. Confirm the install banner appears only on authenticated app pages.
5. Click `Install`, accept the browser install prompt, and confirm the app opens
   or is listed as installed.
6. Confirm the install banner no longer appears in standalone mode.
7. Repeat in a normal browser tab, click `Not now`, reload, and confirm the
   prompt is suppressed for the dismissal window.

Manual iOS/Safari install test:

1. Open the app in iOS Safari and log in.
2. Open `/settings`.
3. Confirm `App and offline access` shows iOS manual install status.
4. Tap `Show install instructions`.
5. Confirm the steps say to use Safari Share -> Add to Home Screen.
6. Add to Home Screen manually, open Travel AI from the Home Screen, and confirm
   the app is detected as installed/standalone.

Manual offline trips management test:

1. Open a private trip online so it is cached.
2. Open `/offline-trips`.
3. Confirm the trip appears with destination, dates, cached time, revision, and
   an `Open` action.
4. Turn off network and open the cached trip from `/offline-trips`.
5. Confirm the offline banner and saved trip data appear.
6. Create an offline itinerary draft and return to `/offline-trips`.
7. Confirm the trip shows `Pending changes`, status, `Sync now`, and
   `Discard pending changes`.
8. Turn network back on and click `Sync now`; confirm the pending badge clears.
9. Click `Remove offline copy`, confirm, and verify the trip disappears.
10. Cache another trip, create an offline draft, click `Clear offline data`, and
    confirm the stronger unsynced-changes warning is shown before data is
    removed.

Manual update handling test:

1. Run the app with the service worker enabled.
2. Change `public/sw.js` or deploy a new build locally, then reload the app.
3. Confirm the update banner appears when the new worker is waiting.
4. With no pending offline changes, click `Refresh to update` and confirm the
   app reloads.
5. Repeat with a pending offline itinerary draft and confirm the banner links to
   `/offline-trips` instead of showing a direct refresh button.

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

## Provider Quotas Panel

Prerequisites: start the stack with the Ops Dashboard enabled and provider limits
on (`OPS_DASHBOARD_ENABLED=true`, `OPS_ADMIN_EMAILS` includes your login,
`PROVIDER_LIMITS_ENABLED=true`). Log in as an ops-admin user.

1. Open `/ops` as the admin. Scroll to the **Provider Quotas** panel.
2. Confirm each provider row shows category, status, requests today, daily quota,
   remaining, minute limit, blocked today, and fallback today.
3. Trigger a route estimate, weather forecast, or place search from a trip page.
4. Click **Refresh** on the Provider Quotas panel (or wait for auto-refresh) and
   confirm the relevant provider's "requests today" increased.
5. Trigger an availability check from a trip item and confirm the availability
   provider row is present; with limits enabled and a cache miss, its
   "requests today" should increase.
6. Click **View details** on a provider and confirm the operation-level breakdown
   and the last-7-days usage list render.
7. To exercise a limit locally: stop the stack, set a very low quota for a real
   provider path with a mock fallback, and restart. A zero-cost setup is
   `ROUTE_PROVIDER=ors`, `ORS_API_KEY=dummy`, `ORS_BASE_URL=http://127.0.0.1:9`,
   `ROUTE_PROVIDER_FALLBACK_TO_MOCK=true`, `ORS_DAILY_QUOTA=1`.
8. Trigger a route estimate twice. The first consumes the quota (falling back to
   mock because ORS is unreachable); the second is quota-exceeded and served by
   the mock fallback.
9. Refresh the panel and confirm the routes provider shows a `quota_exceeded`
   status and non-zero blocked/fallback counts.
10. Confirm the **Reset (dev)** button is visible (it is hidden when the service is
   in production / `resetAllowed=false`). Click it, confirm the confirmation
   prompt, and confirm today's counters reset.

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

## Cost Analytics Dashboard

Trip dashboard:

1. Open a private trip with a budget, itinerary item costs, and accommodation
   cost.
2. From the trip detail page, click `View cost analytics`.
3. Confirm `/trips/{id}/analytics` shows summary cards for estimated total,
   budget usage, missing estimates, and low-confidence/provider-estimate risk.
4. Confirm cost-by-day, category, source, and confidence sections render with
   totals in the selected currency.
5. Confirm the expensive-items table includes itinerary/accommodation costs and
   links back to the relevant trip day or trip detail page.
6. Change the currency selector and confirm totals refresh without losing the
   current trip context.
7. Click a budget-related insight such as `Optimize Day N` and confirm it opens
   the trip planning flow for that trip.
8. Export CSV and PDF reports and confirm each includes summary, rollup,
   expensive item, warning, and planning-disclaimer content.
9. Log in as an accepted viewer collaborator and confirm the analytics page is
   readable, exports are available, and edit-oriented actions are hidden.
10. Open the public share URL in a logged-out window and confirm there is no
    link to private cost analytics.

## Cost Splitting Between Travelers

1. Open a completed private trip with at least one itinerary item cost and one
   accommodation cost.
2. In the cost-splitting section, add two travelers with distinct names and
   emails.
3. Confirm the default summary splits unconfigured costs equally and shows a
   default-split warning/count.
4. On an itinerary item with a cost, click `Split cost`, choose selected-equal,
   select one traveler, and save.
5. Confirm that item appears fully under that traveler in the per-traveler
   detail and the itinerary revision updates.
6. On the accommodation cost, choose custom percentages such as 25/75 and save.
7. Confirm the per-traveler totals and accommodation detail reflect those
   percentages.
8. Download the cost-splitting CSV and PDF and confirm they include the summary,
   traveler allocations, unassigned/default warnings, and planning disclaimer.
9. Remove one traveler and confirm stale split rules referencing that traveler
   are surfaced as invalid/unassigned instead of silently reallocating money.
10. Log in as an accepted viewer collaborator and confirm the cost-splitting
    summary is readable but traveler and split edit controls are hidden.

Limitations: cost splitting is planning-only. It does not create payments,
settlements, reimbursements, invoices, receipts, bookings, or accounting
records.

Workspace dashboard:

1. Create or open a workspace with at least two trips, including one over budget
   or close to budget and one with incomplete/missing item estimates.
2. Open `/workspaces/{workspaceId}/analytics` from the workspace page.
3. Confirm the summary cards show workspace estimated total, trip count,
   over-budget trips, missing estimates, and low-confidence/provider risk.
4. Confirm the trips table lists both trips with budget utilization and links
   back to each trip.
5. Confirm category, source, and month charts render and reflect the selected
   workspace trips.
6. Change date filters between all trips, this year, next 12 months, and a
   custom range; confirm the trip table and totals update.
7. Change the currency selector and confirm rollups refresh in the selected
   currency.
8. Export CSV and PDF reports and confirm they include workspace summary,
   trip rollups, category/source/month sections, expensive items, warnings, and
   the planning disclaimer.
9. Log in as a workspace viewer and confirm the dashboard is readable while
   edit-oriented actions remain hidden.
10. Remove that viewer from the workspace and confirm the analytics URL returns
    an access-denied state.

## Real Availability Layer

1. Open a private trip as owner or editor with at least one museum, landmark,
   tour, activity, palace, castle, zoo, aquarium, theme park, ticket, or event
   itinerary item.
2. Confirm non-bookable items such as walks, restaurants, transport, hotels,
   notes, rests, and public parks do not show a `Check availability` control.
3. Click `Check availability` on a bookable item.
4. Confirm the initial state is replaced by a normalized result showing status
   (`available`, `limited`, `unavailable`, or `unknown`), provider label, checked
   time, match confidence, and any warnings.
5. With the default mock provider, a Colosseum-like attraction should return an
   available or limited HTTPS booking link, EUR price, price type, duration, and
   start times. A public park/walk should show no paid booking option if checked
   via API/devtools.
6. Click `Check availability` again for the same item and confirm the second
   result shows the cache indicator when `AVAILABILITY_CACHE_ENABLED=true`.
7. Click the booking link and confirm it opens externally; the app should not
   embed checkout or ask for payment details.
8. Click `Apply price estimate`, confirm the replace-cost prompt when the item
   already has an estimate, and accept.
9. Confirm the item now shows a provider-filled ticket/activity cost, the budget
   panel recalculates, and version history records a manual itinerary update.
10. Open `Trip Quality Checks` before checking a bookable item and confirm it can
    show an availability-unchecked issue. After applying a provider price, confirm
    a low-confidence match, a notably changed price, or fallback data each surface
    as the matching checklist warning/info row (none block submission).
11. Stop only the External Integrations Service and click `Check availability`;
    confirm the card shows a safe unavailable/error state and the trip page does
    not crash. Restart the service afterward.
12. **Real provider (optional):** set `AVAILABILITY_PROVIDER=ticketmaster` and a
    valid `TICKETMASTER_API_KEY`, restart External Integrations, and check
    availability on an event/concert item. Confirm the provider badge reads
    `Ticketmaster` and results carry venue/date/booking links. Because provider
    data changes, do not assert specific event names or prices. With a missing/
    invalid key and `AVAILABILITY_FALLBACK_TO_MOCK=true`, confirm the card falls
    back to a clearly-labelled "Fallback estimate".

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

## Browser Push Notifications

1. Generate local VAPID keys with `npx web-push generate-vapid-keys`.
2. Set `WEB_PUSH_ENABLED=true`, `WEB_PUSH_VAPID_PUBLIC_KEY`,
   `WEB_PUSH_VAPID_PRIVATE_KEY`, and `WEB_PUSH_SUBJECT=mailto:dev@example.com`
   in `infra/.env`, then restart Notification Service and the web app.
3. Log in with a supported browser and open `/settings`.
4. Confirm `Push notifications` appears above the notification preference
   matrix.
5. Click `Enable push notifications` and accept the browser permission prompt.
6. Confirm the UI says push notifications are enabled on this device.
7. Trigger a notification from another account, for example by adding a comment
   on a shared trip or completing an itinerary generation job.
8. Confirm a system notification appears.
9. Click the system notification and confirm the app opens the related trip or
   `/notifications`.
10. Return to `/settings`, click `Disable on this device`, then trigger another
    notification and confirm that device no longer receives push notifications.

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

## Workspaces (User Service + Trip Service + Web)

Use two browsers (or one normal + one private window): workspace owner A and
member B.

1. Login as A and open `/workspaces`.
2. Create a workspace, then confirm the header switcher includes **All trips**,
   **Personal**, and the new workspace.
3. Open workspace settings and invite B as `member`.
4. Login as B in the second browser, open `/workspace-invitations`, and accept.
5. In browser A, switch to the workspace and create a trip. Confirm the trip
   card and detail page show a workspace badge.
6. In browser B, switch to the workspace and confirm the workspace trip appears.
7. As B, edit the workspace trip itinerary or queue a generation action.
8. As A, change B's role to `viewer`.
9. As B, confirm the trip still opens but edit/generation controls are hidden or
   blocked by the API.
10. As A, remove B from the workspace.
11. As B, confirm the workspace disappears from the switcher and the workspace
    trip no longer opens unless B is separately invited as a trip collaborator.
12. Confirm A's personal trips still appear under **Personal**, and public share
    links remain anonymous read-only.

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

## Workspace Approval Workflow (Trip Service + Web)

Use two browsers: workspace owner A and member B (from the Workspaces flow).

1. As A, create a workspace and invite B as `member`; B accepts.
2. As B, switch to the workspace and create a workspace trip, then generate or
   edit an itinerary so it has at least one day and one activity.
3. As B, open the trip detail page. Confirm the **Approval** panel shows a
   **Draft** badge and a checklist. Warnings (missing budget, availability, etc.)
   are listed but do not block submission.
4. As B, click **Submit for approval**. Optionally acknowledge warnings and add a
   note, then submit. Confirm the badge changes to **Pending approval**.
5. As A, open `/workspaces/<id>/approvals`. Confirm the trip appears under
   **Pending** with the submitter, checklist status, and estimated total, and
   that the counts row reflects one pending item.
6. As A, click **Request changes**, enter a required note, and submit. Confirm
   the trip moves to **Changes requested** and B receives a notification.
7. As B, confirm the trip detail shows **Changes requested** with the reviewer
   note, edit as needed, and resubmit.
8. As A, click **Approve** (optional note). Confirm the badge shows **Approved**
   and B is notified.
9. As B, edit the itinerary (or budget/travelers/cost split). Confirm the badge
   returns to **Draft** and the panel notes it must be resubmitted. The panel
   also warns before editing an approved trip.
10. As A, change B's role to `viewer`. Confirm B can still open the approval
    panel read-only but the **Submit for approval** button is gone and the API
    rejects submission.
11. Open a personal trip and confirm the approval panel reads
    **Approval not required** with no actions.
12. While offline (DevTools → Network → Offline), confirm approval actions are
    disabled with an "Approval actions require internet" note.

## Workspace Policy Rules v1

1. Create a workspace trip with an itinerary containing estimated costs.
2. Open **Workspace settings → Planning policy** as an owner/admin.
3. Enable **Maximum trip budget**, set a limit below the trip estimate, choose
   **Blocking**, and save.
4. Open the trip and confirm the Workspace policy panel shows the violation.
5. Try **Submit for approval** and confirm the policy blocker disables/rejects
   submission.
6. Change the rule severity to **Warning**, save, and re-check the trip.
7. Confirm submission is now allowed with a visible warning.
8. Sign in as a member/viewer and confirm the policy is read-only.
9. Generate, regenerate, or adapt a workspace trip and confirm the UI explains
   that policy is AI guidance and the backend check remains authoritative.
10. Open a personal trip and confirm no workspace policy applies.

## Troubleshooting

### Optional Ops Dashboard Check

The `/ops` dashboard is intentionally disabled by default. To include it in the
API smoke test, start the stack with `OPS_DASHBOARD_ENABLED=true`, set
`OPS_ADMIN_EMAILS` to the smoke user email, run the script with
`SMOKE_AUTH_EMAIL=<same-email>` and `SMOKE_EXPECT_OPS_DASHBOARD=true`, then open
`http://localhost:3000/ops` with that account.

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
