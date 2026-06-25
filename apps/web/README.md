# Travel AI Planner Web

Next.js Web App v1 for registering/logging in, managing profile and travel
preferences, creating trip requests, listing trips, opening trip details,
generating itineraries, viewing generated plans, showing mock weather context,
and editing completed itineraries. Completed trips with itineraries also show
version history with read-only preview and restore actions. Owners can create a
public read-only share link for a trip, set expiration/password controls, and
disable that link later. Owners can also invite registered users to collaborate
on private trips as viewers or editors.

The Web App prevents stale itinerary overwrites with Trip Service
`itineraryRevision` values. Manual edit mode captures the revision when editing
starts and sends it as `expectedItineraryRevision` on save. If Trip Service
returns `409 itinerary_conflict`, the page shows a conflict panel with
`Reload latest` and `Cancel my changes` actions instead of retrying or forcing an
overwrite. Generate, regenerate, restore, place review, and route optimization
actions send the latest displayed revision; on conflict they show a readable
message and refetch the trip.

## Source Layout

Application code lives under `src/`:

- `src/app`
- `src/components`
- `src/lib`
- `src/types`

## Setup

```bash
cd apps/web
cp .env.example .env.local
npm install
npm run dev
```

The app expects the service URLs in:

```bash
NEXT_PUBLIC_AUTH_SERVICE_URL=http://localhost:8082
NEXT_PUBLIC_TRIP_SERVICE_URL=http://localhost:8080
NEXT_PUBLIC_USER_SERVICE_URL=http://localhost:8083
NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL=http://localhost:8084
NEXT_PUBLIC_NOTIFICATION_SERVICE_URL=http://localhost:8086
TRIP_SERVICE_INTERNAL_URL=http://localhost:8080
NOTIFICATION_SERVICE_INTERNAL_URL=http://localhost:8086
```

`NOTIFICATION_SERVICE_INTERNAL_URL` is used by the server-side notification proxy
route (`app/api/notification-service/[...path]`); in Docker Compose it is the
internal hostname `http://notification-service:8086`.

## Backend

Start the repository backend services first, then run the web app. The frontend calls Auth Service endpoints:

- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`
- `GET /auth/me`

After login/register, the frontend stores the access and refresh token in
`localStorage` for development v1 and sends `Authorization: Bearer <accessToken>`
to Trip Service. Secure httpOnly cookies are recommended for production.

The frontend calls the protected Trip Service endpoints:

- `POST /trips`
- `GET /trips?limit=20&offset=0`
- `GET /trips/shared-with-me`
- `GET /trips/{id}`
- `POST /trips/{id}/generate`
- `PUT /trips/{id}/itinerary`
- `POST /trips/{id}/itinerary/days/{dayNumber}/regenerate`
- `POST /trips/{id}/itinerary/days/{dayNumber}/items/{itemIndex}/regenerate`
- `GET /trips/{id}/share`
- `POST /trips/{id}/share`
- `PATCH /trips/{id}/share`
- `DELETE /trips/{id}/share`
- `GET /trips/{id}/itinerary/versions`
- `GET /trips/{id}/itinerary/versions/{versionId}`
- `POST /trips/{id}/itinerary/versions/{versionId}/restore`
- `POST /trips/{id}/collaborators`
- `GET /trips/{id}/collaborators`
- `PATCH /trips/{id}/collaborators/{collaboratorId}`
- `DELETE /trips/{id}/collaborators/{collaboratorId}`
- `POST /trips/{id}/collaborators/{collaboratorId}/accept`
- `POST /trips/{id}/collaborators/{collaboratorId}/decline`
- `GET /trips/{id}/presence/stream`
- `POST /trips/{id}/presence/state`
- `GET /trips/{id}/presence`
- `GET /collaboration/invitations`
- `GET /trips/{id}/comments` (and `?dayNumber=&itemIndex=` for one item)
- `GET /trips/{id}/comments/counts`
- `POST /trips/{id}/comments`
- `PATCH /trips/{id}/comments/{commentId}`
- `DELETE /trips/{id}/comments/{commentId}`
- `GET /public/trips/{shareToken}/status` without Authorization for public share pages
- `POST /public/trips/{shareToken}/unlock` without Authorization to unlock protected shares
- `GET /public/trips/{shareToken}` without Authorization for unprotected shares or with `Authorization: Bearer <publicShareAccessToken>` for protected shares

The frontend calls External Integrations Service v1 directly for place search,
route estimates, and weather forecasts:

- `GET /places/search?query=Colosseum&destination=Rome`
- `GET /places/{placeId}`
- `POST /routes/estimate`
- `GET /weather/forecast?destination=Rome&startDate=2026-08-10&days=3`

The Web App does not call third-party place, route, or weather APIs directly.
If External Integrations Service is configured with `PLACE_PROVIDER=foursquare`,
the browser still calls the same `/places/search` and `/places/{placeId}`
endpoints and receives normalized `Place` objects.

Automatic place enrichment after AI generation is owned by Trip Service. The Web
App does not call enrichment directly; it renders returned `place` metadata and
shows an `Auto-matched place` confidence badge when an item has
`placeEnrichment.status === "matched"`. Manual place changes/removals in the
editor clear `placeEnrichment` so stale auto-match labels are not saved.

## Place Enrichment Review

Completed trip detail pages show a `Place Matches` review section when the
itinerary contains auto-enrichment metadata. The section lists auto-matched
places and no-match results from Trip Service:

- `Accept` keeps the attached place and marks `placeEnrichment.reviewStatus` as
  `accepted`.
- `Change` opens the existing place search dialog, replaces the attached place,
  and marks the review status as `changed`.
- `Remove` clears the attached place and marks the review status as `removed`.
- `Search manually` appears for no-match rows and saves the selected place with
  review status `changed`.

Each review action saves the full itinerary through the existing
`PUT /trips/{id}/itinerary` endpoint, so Trip Service records a `MANUAL_EDIT`
version. The review state lives inside the itinerary JSON for v1; there is no
separate review table or background worker.

## Collaborative Planning

The trips page shows `Pending invitations`, `My Trips`, and `Shared with me`.
Accepted shared trips open through the normal `/trips/{id}` route; Trip Service
decides access and returns `trip.access`.

Owner UI:

- Shows `Share itinerary` and `Collaborators` panels.
- Can invite registered users by exact email, choose `viewer` or `editor`,
  change roles, and remove collaborators.

Editor UI:

- Shows itinerary edit/regenerate, place review, route optimization, export,
  and version restore controls.
- Hides public share and collaborator management.

Viewer UI:

- Shows read-only itinerary, map/weather/distance, export, and version preview.
- Hides edit, regenerate, place review actions, route optimization apply,
  restore, public share, and collaborator management.

Current v1 limitations: registered users only, advisory presence only, no
real-time itinerary sync, no automatic merge, no diff viewer, and no locking.

## Real-time Trip Presence

Private trip detail pages show a `Currently here` presence card for owners and
accepted collaborators. The Web App opens a fetch-based Server-Sent Events
stream (`lib/presence/use-trip-presence-stream.ts`) to
`GET /trips/{id}/presence/stream`. Native `EventSource` is not used because the
stream requires `Authorization: Bearer <accessToken>`; chunks are parsed with
the shared SSE parser in `lib/notifications/sse-parser.ts`.

When an owner/editor enters manual itinerary edit mode, the page calls
`POST /trips/{id}/presence/state` with `{"state":"editing"}`. Saving, canceling,
leaving the page, or hiding the tab best-effort returns the state to
`viewing`; stream disconnects also unregister the session server-side. Viewers
connect and appear as `viewing`, but cannot enter edit mode.

If another collaborator is currently editing, an amber warning appears near the
itinerary edit controls. The warning is advisory only: it never blocks editing,
saving, regeneration, restore, or route optimization. Public share pages do not
mount presence UI and make no presence requests.

## Itinerary Comments

On private trip detail pages, each itinerary item in the read-only view shows a
`Comments` button with an active-comment count badge (powered by
`GET /trips/{id}/comments/counts`). A `comments across itinerary items` summary
renders above the itinerary. Owners and accepted collaborators (viewer or
editor) can open the per-item panel to read and post comments.

In the comment panel:

- Comments load via `GET /trips/{id}/comments?dayNumber=&itemIndex=`.
- Posting a comment uses `POST /trips/{id}/comments`; the textarea shows a
  `0/2000` counter and disables `Post` for empty/too-long bodies.
- A comment shows `You` for your own comments and `Collaborator` otherwise, plus
  `edited` when it was changed after creation.
- Authors can `Edit` (inline) and `Delete` their own comments; trip owners can
  `Delete` any comment. The backend enforces these rules — the UI only hides
  buttons. After any change the panel refetches item comments and counts.

Comments are shown in the read-only itinerary only and are hidden while editing
the itinerary. They are a private feature: the public `/share/{shareToken}` page
never renders comment UI and makes no comment requests.

Limitations: no real-time updates, no notifications, no mentions, and no
threaded replies. Comments are keyed by `dayNumber`/`itemIndex`, so heavy
itinerary reordering can leave a comment pointing at a different item.

## Recent Activity

Private trip detail pages render a `Recent activity` panel at the bottom
(`components/activity/ActivityFeed.tsx`). It shows a chronological, newest-first
audit log of important successful actions on the trip — generation, edits,
regenerations, version restores, comments, collaborator changes, and share
setting changes.

- Events load via `GET /trips/{id}/activity?limit=30&cursor=...`
  (`lib/api/activity.ts`) using `useInfiniteQuery`; a `Load more` button pages
  through older events with the returned `nextCursor`.
- Events are grouped by calendar day (`Today` / `Yesterday` / date) by
  `lib/activity/group-activity-by-date.ts` and rendered as readable lines by
  `lib/activity/format-activity-event.ts` (e.g. `You generated the itinerary`,
  `You commented on Day 2 · Louvre Museum`, `You invited anna@example.com as
  editor`). The actor is shown as `You` for the current viewer, `System` for
  actor-less events, and `Collaborator` otherwise (no display names in v1). The
  formatter is defensive: missing metadata degrades gracefully and an unknown
  event type renders `Activity recorded` rather than crashing.
- The panel is shown only to the owner and accepted collaborators (same access
  rule as comments) and renders nothing otherwise. It is never mounted on the
  public `/share/{shareToken}` page and makes no activity requests there.
- There are no real-time updates. After comment, collaborator, share, itinerary
  edit/regenerate, and version-restore actions the page invalidates the activity
  query (`activityKeys`) so the feed refreshes; otherwise it updates on reload.

Limitations: no real-time updates, no filtering/search, and generic actor
labels. In-app notifications for these events are surfaced separately by the
notification bell (see below).

## Notifications

The authenticated header (`components/layout/AppHeader.tsx`) shows a notification
bell (`components/notifications/NotificationBell.tsx`) backed by the Notification
Service. It is private, authenticated, per-user data — the bell renders nothing
for logged-out / public share viewers and makes no requests for them.

- The unread count is polled (~every 45s) via `GET /notifications/unread-count`
  using React Query (`lib/notifications/use-notifications.ts`); a red badge shows
  when the count is greater than zero (`99+` cap). Polling remains enabled as a
  fallback and slows to ~120s while the real-time stream is connected.
- The bell also mounts a fetch-based Server-Sent Events stream
  (`lib/notifications/use-notification-stream.ts`) to
  `GET /notifications/stream`. Native `EventSource` is not used because it cannot
  send an `Authorization` header; the app uses `fetch` with
  `Authorization: Bearer <accessToken>` and parses SSE chunks manually in
  `lib/notifications/sse-parser.ts`.
- On `notification.created`, the stream invalidates the notification list and
  unread-count React Query keys so the badge/dropdown update without a manual
  refresh. `heartbeat` events only keep the stream alive. If the stream
  disconnects, the hook reconnects with backoff while polling continues.
- Clicking the bell opens a dropdown
  (`components/notifications/NotificationsDropdown.tsx`) that fetches the latest
  10 notifications (`GET /notifications?limit=10`), with loading / empty
  (`No notifications yet.`) / error states, an unread dot indicator, relative
  timestamps, and a `Mark all as read` action. A `View all` link opens the full
  `/notifications` page (`app/notifications/page.tsx`) with `Load more` cursor
  pagination.
- Clicking a notification marks it read (`PATCH /notifications/{id}/read`),
  refreshes the unread count, and navigates to the related destination resolved
  by `lib/notifications/notification-navigation.ts` (`collaboration_invited` →
  `/trips`; anything with a `tripId` → `/trips/{tripId}`; otherwise `/trips`).
- All requests go through the same-origin proxy route
  `app/api/notification-service/[...path]` (which never forwards `internal/*`
  paths), attaching the user's bearer token via the shared `apiFetch` client. The
  base URL comes from `lib/config.ts` (`getNotificationApiBaseUrl`).

In-app notifications may **also trigger an email** depending on server
configuration: the Notification Service can send email for selected types
(collaboration invited, comment created, collaborator role changed/removed by
default). The `/settings` page includes a `Notification preferences` section
where authenticated users can enable or disable in-app and email delivery by
category: collaboration invitations, comments, role changes, and trip updates.
These settings are saved through `GET/PUT /notifications/preferences` on the
Notification Service via the same-origin notification proxy. Locally the default
`EMAIL_PROVIDER=mock` sends nothing externally (see the Notification Service and
infra READMEs).

Limitations: real-time updates are SSE-only with polling recovery (no
WebSockets/push), in-app notifications surface in the bell, and the bell shows a
count plus the latest items. No unsubscribe links, email digests, quiet hours,
or per-trip notification preferences yet.

## Public Trip Sharing

Authenticated trip detail pages include a `Share itinerary` panel. It calls
`GET /trips/{id}/share` on load, `POST /trips/{id}/share` to create or
re-enable a link, `PATCH /trips/{id}/share` to save expiration/password
settings, and `DELETE /trips/{id}/share` to disable it. The share URL points to
`/share/{shareToken}`.

The panel supports `Never`, `7 days`, `30 days`, and custom expiration, plus
password enable/change/remove. Passwords are sent only when creating, enabling,
or changing protection and are never shown after save.

The public share page calls `GET /public/trips/{shareToken}/status` first. If
the link is password protected, it shows an unlock form and calls
`POST /public/trips/{shareToken}/unlock`. The returned public share access token
is stored only in `sessionStorage` under
`public-share-access-token:{shareToken}` with its expiry, so refresh works until
the short-lived token expires. The app clears that stored token if the backend
returns `401`.

After unlock, the public page calls `GET /public/trips/{shareToken}` with the
public share bearer token. Unprotected links are fetched without Authorization.
It renders the sanitized read-only trip summary, itinerary, place metadata, map
view, distance estimate, weather context, and PDF/.ics export when data is
available. It does not render edit, regenerate, version history, route
optimization, place-match review, settings, logout, or any private preference
controls.

Disabled, expired, or unknown share links show:

```text
This shared trip is unavailable, expired, or disabled.
```

Public sharing v1 has one link per trip and no analytics or collaboration.

## Export v1

Private trip detail pages and public share pages support read-only export from
the browser:

- `Download PDF` creates a clean itinerary summary with trip facts, day-by-day
  items, visible place details, weather summary when loaded, and distance
  summary when available.
- `Download calendar (.ics)` creates one calendar event per itinerary item with
  a parseable time and a trip start date.

Calendar export skips untimed or unparseable itinerary items. Exported calendar
times are local floating times, so the importing calendar app interprets them in
the user's calendar timezone. Export v1 does not call Google Calendar, Apple
Calendar, Outlook, OAuth, or any external calendar API.

Exports are generated from a sanitized frontend model. They do not include user
email, user ID, preferences, tokens, itinerary version history, or internal
place-enrichment debug metadata.

To edit an itinerary, open a completed trip and click `Edit itinerary`. The
editor supports changing day titles and item fields, adding/removing days, and
adding/removing items. `Save` sends `PUT /trips/{id}/itinerary` with:

```json
{
  "itinerary": {
    "days": [
      {
        "day": 1,
        "title": "Edited Day",
        "items": [
          {
            "time": "10:00",
            "type": "activity",
            "name": "Edited Activity",
            "note": "",
            "estimatedCost": null,
            "place": {
              "provider": "mock",
              "providerPlaceId": "mock-colosseum-rome",
              "name": "Colosseum",
              "address": "Piazza del Colosseo, 1, 00184 Roma RM, Italy",
              "latitude": 41.8902,
              "longitude": 12.4922,
              "rating": 4.7,
              "ratingCount": 120000,
              "mapUrl": "https://maps.example.com/mock-colosseum-rome",
              "category": "landmark",
              "website": "https://example.com/colosseum",
              "openingHours": [
                { "dayOfWeek": 1, "open": "08:30", "close": "19:15" },
                { "dayOfWeek": 2, "open": "08:30", "close": "19:15" }
              ]
            }
          }
        ]
      }
    ]
  }
}
```

Itinerary editing v1 replaces the whole itinerary JSON. Partial regeneration
buttons call Trip Service to regenerate a day or item while preserving the rest
of the itinerary.

To attach a normalized place, open a completed trip, click
`Edit itinerary`, click `Attach real place` on an item, search, select a result,
then click `Save`. Existing itinerary items without `place` metadata continue to
render normally. Mock places can include optional `openingHours` using
`dayOfWeek` values `1 = Monday` through `7 = Sunday` and local `HH:mm` times.
Search results and attached-place displays include a provider label such as
`Provider: Mock` or `Provider: Foursquare`.
When an attached place has hours and the trip has a start date, the read-only
itinerary shows an advisory `Likely open at this time`, `May be closed at this
time`, or `May be closed on this day` badge plus that day's hours. The warning
is advisory because schedules are optional, simple local trip dates are used,
and there are no timezone, holiday, or special-date overrides. Real provider
results may omit rating, website, coordinates, or `openingHours`; the UI hides
missing ratings, shows unknown opening hours, and maps only places with valid
coordinates.

v1 intentionally has no flights, hotels, real weather provider, real Google
Places provider, real opening-hours provider, or turn-by-turn route geometry.
See Distance / Walking Estimate below for the approximate route and straight-line
distances the Web App shows.

## Weather Context

Trip detail pages show a `Weather context` card near the top of the page. When a
trip has `destination`, `startDate`, and `days`, the card calls
`GET /weather/forecast` on the External Integrations Service and renders daily
mock forecast rows with summary, temperature range, precipitation chance, wind
speed, provider label, and warning badges. When `provider` is `mock`, the UI
labels it as a local-development mock forecast.

If the trip has no start date, the card asks the user to add one. If the weather
service is unavailable or returns an error, the card shows `Weather forecast
unavailable.` and the rest of the trip detail page continues to work.

Weather is not persisted by the Web App or Trip Service. During itinerary
generation/regeneration, Trip Service may fetch weather and pass it to AI
Planning Service so prompts can adapt to rain, heat, cold, or wind.

## Map View

Completed trips with itinerary items that have attached places and valid
coordinates show a read-only Map View on the trip detail page. The map uses
Leaflet with OpenStreetMap tiles and renders markers only for itinerary items
with numeric latitude and longitude values. Use the day filter to view all
mapped places or only the mapped places for a single day.

Attach places in itinerary edit mode first, then save or leave edit mode to see
them on the map. Map View v1 does not support route optimization, marker
dragging, or editing places from the map. Marker popups show opening-hours
status when the attached place includes `openingHours`.

## Distance / Walking Estimate

Completed trips with itinerary items that have attached place coordinates show a
read-only Distance estimate panel below the Map View on the trip detail page. The
panel prefers a route estimate from the External Integrations Service and falls
back to a frontend straight-line (Haversine) estimate when that service is
unavailable.

- For each day with at least two mapped stops, the Web App calls
  `POST /routes/estimate` on the External Integrations Service
  (`mode: "walking"`, ordered stops) and shows
  `Route estimate: <km> · ~<time> walking` plus a `Route estimates by mock
  provider` badge and the smaller straight-line fallback figure.
- Route estimates are approximate (mock provider: Haversine × 1.25 at 5 km/h),
  read-only, and never modify the itinerary or get persisted by Trip Service.
- If the route service is slow, down, or returns an error, the panel shows
  `Route service unavailable. Showing straight-line estimate.` and uses the
  frontend Haversine estimate (Earth radius 6371 km, flat 5 km/h pace). The page
  never blocks on or crashes from route loading.
- Compares each day total with `maxWalkingKmPerDay` from the User/Profile
  Service, using the route distance when available and the straight-line
  distance otherwise. The warning line states which estimate was used.

Only itinerary items with a numeric, in-range `latitude` and `longitude` are
counted, so existing trips without place coordinates keep working. A day needs at
least two mapped stops before it contributes a distance. Preferences are fetched
non-blocking: if the request fails the estimates still render, just without the
preference comparison. Distance estimates are hidden during itinerary edit mode
and reappear after saving or leaving edit mode.

The straight-line logic lives in `src/lib/itinerary/distance-utils.ts`; route
stop extraction in `src/lib/itinerary/route-estimate-utils.ts`; the route
estimate fetching (TanStack Query, one query per day, keyed by stop coordinates,
no retries) in `src/lib/hooks/useRouteEstimates.ts` and `src/lib/api/routes.ts`.
All are covered by their respective `*.test.ts` files. This requires
`NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL` and the service's CORS to allow
`POST` from the browser origin.

## AI Quality Feedback Loop

Completed trip detail pages show a `Trip Quality Checks` card after weather
context. The card analyzes the current itinerary with existing frontend and
service signals:

- route estimates, falling back to Haversine distance summaries
- `maxWalkingKmPerDay` from user preferences
- weather forecast rain and heat thresholds
- place opening hours at the scheduled item time
- place enrichment confidence and review state
- missing map-ready place coordinates for enriched itinerary items

The checks are advisory and never regenerate automatically. Users stay in
control: `Improve day` and `Improve item` buttons build concise AI instructions
from the detected issues and call the existing partial regeneration endpoints:

- `POST /trips/{id}/itinerary/days/{dayNumber}/regenerate`
- `POST /trips/{id}/itinerary/days/{dayNumber}/items/{itemIndex}/regenerate`

After regeneration, the Web App refetches the trip and refreshes itinerary
version history, so Trip Service records the change through its existing version
logic. If route estimates, weather, preferences, opening hours, or enrichment
metadata are unavailable, the card simply omits those checks and continues with
the signals it has. In manual edit mode, AI improvement buttons are hidden and
the card asks the user to save or cancel edits first.

## Route Optimization v1

Completed trips can suggest a better visiting order for a single day. The
`Distance estimate` panel shows an `Optimize order` button for any day with at
least three mapped places (items with valid coordinates). Clicking it opens a
preview dialog; nothing is saved until the user confirms.

- Optimizes one day at a time. It never reorders across days.
- Uses a simple nearest-neighbour algorithm over straight-line (Haversine)
  distance. It is **not** real routing — no external routing APIs and no
  Google/Mapbox routing are involved, and the UI labels the figures as
  approximate.
- Keeps the first mapped place fixed so the day keeps its starting point.
- Reorders mapped places into the existing time slots: the place that lands in a
  position inherits that position's original time.
- Keeps unmapped items (notes, rest, free time) in their original positions.
- Shows current vs suggested order side by side, plus the original/optimized
  distance and the estimated saving (km and walking minutes). If the saving is
  negligible it says so but still allows applying.
- Applying saves the full itinerary through the existing
  `PUT /trips/{id}/itinerary` endpoint, which creates a `MANUAL_EDIT` version
  through the existing Trip Service versioning. The order persists after refresh.

Optimize controls only appear in read-only mode, so they are hidden while
editing the itinerary manually. The pure logic lives in
`src/lib/itinerary/route-optimization-utils.ts` and is covered by
`route-optimization-utils.test.ts`; the UI is `OptimizeDayOrderDialog`.

The Version History panel appears on completed trips that have an itinerary. It
fetches version summaries, displays source labels such as `Generated`,
`Manual edit`, `Regenerated day`, `Regenerated item`, and `Restored`, and loads
full itinerary JSON only when the user clicks `View`. `Restore` asks for
confirmation, replaces the current itinerary through Trip Service, refetches the
trip, and refreshes the version list. Viewing a version never changes the current
trip.

Version history v1 starts after the backend feature is deployed. It does not
support diff view, branching, named versions, version comparison,
drag-and-drop, full map views, payments, admin flows, or collaboration.

The frontend calls the protected User Service endpoints from `/settings`:

- `GET /users/me/profile`
- `PUT /users/me/profile`
- `GET /users/me/preferences`
- `PATCH /users/me/preferences`

The settings page also calls the protected Notification Service preference
endpoints:

- `GET /notifications/preferences`
- `PUT /notifications/preferences`

Browser requests go through the Next.js route proxy at `/api/trip-service/*`,
which forwards to `TRIP_SERVICE_INTERNAL_URL` when set, then falls back to
`NEXT_PUBLIC_TRIP_SERVICE_URL`. In Docker Compose, the browser-facing URL stays
`http://localhost:8080` while server-side proxy calls use
`http://trip-service:8080`. The proxy forwards the `Authorization` header.

Auth Service, Trip Service, and User Service enable CORS for
`http://localhost:3000`, so direct browser calls to
`NEXT_PUBLIC_AUTH_SERVICE_URL`, `NEXT_PUBLIC_TRIP_SERVICE_URL`, and
`NEXT_PUBLIC_USER_SERVICE_URL` remain possible during local development. External
Integrations Service enables CORS for `http://localhost:3000` and is called via
`NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL`.

Open the settings page at:

```text
http://localhost:3000/settings
```

Preferences are saved in User/Profile Service. Trip Service fetches them during
itinerary generation. The Web App does not send preferences directly to Trip
Service.

The current Trip Service validates the most active pace as `packed`; the UI labels that option as `Intensive`.

## Commands

```bash
npm run dev
npm run build
npm run start
npm run typecheck
npm test
```

`npm test` runs the Vitest unit tests for the pure itinerary utilities (for
example the distance/walking estimate helpers).
