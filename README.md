# Travel AI App

AI travel planning project with Go Auth Service, Go Trip Service, Go User
Service, Go External Integrations Service, Go Notification Service,
Python/FastAPI AI Planning Service, and a Next.js web app.

Auth Service v1 lives in `services/auth-service` and supports email/password
registration, login, refresh token rotation, logout, and JWT-backed `/auth/me`.
Trip Service validates those JWT access tokens locally with the shared
`JWT_ACCESS_SECRET` and scopes `/trips` data by the authenticated `sub` user ID.
Trip Service also records itinerary version snapshots after generation, manual
edits, partial regeneration, and restores; users can preview older versions and
restore them without deleting history. Conflict Detection v1 adds integer
`itineraryRevision` values and requires itinerary mutations to send
`expectedItineraryRevision`; stale mutations return HTTP `409
itinerary_conflict` instead of silently overwriting newer changes. Authenticated
trip owners can also create one public read-only share link per trip. Public
share links use opaque random tokens, expose only sanitized trip/itinerary data
at `/share/{shareToken}`, and can be disabled by the owner. Share Controls v1
adds optional expiration and password protection; protected public viewers
unlock with a short-lived public share token that is separate from normal user
auth JWTs and scoped to one share token.
Collaborative Planning v1 lets trip owners invite existing registered users by
email as `viewer` or `editor` collaborators. Pending invitees can accept from
the web app, accepted viewers get read-only private trip access, and accepted
editors can edit/regenerate/restore itinerary versions without managing public
sharing or collaborators. Public share links remain independent and read-only.
Itinerary Comments v1 lets owners and accepted collaborators (viewer or editor)
leave comments on individual itinerary items. Comments live in a dedicated
`itinerary_comments` table (never in the itinerary JSON), are linked by
`trip_id`/`day_number`/`item_index`, and are soft-deleted. Authors can edit and
delete their own comments; trip owners can delete any comment. Comments are a
private authenticated feature and are never exposed on public share links/pages.
Real-time Trip Presence v1 lets owners and accepted collaborators see who else
is viewing or editing a private trip. Trip Service exposes an authenticated SSE
stream and advisory state update endpoint backed by an in-memory,
single-instance presence manager. The Web App shows `Currently here` on private
trip detail pages and warns when another collaborator is editing. Presence is
not a lock, does not sync documents, and is never shown on public share pages;
revision-checked writes are the backend protection against stale itinerary
saves.
Soft Edit Locks v1 add advisory, in-memory itinerary edit locks in Trip Service.
Owners/editors attempt to acquire or renew a temporary lock before manual edit
mode, viewers can only read lock status, and public share viewers have no
access. If another editor holds the lock, the Web App warns the user but allows
`Continue anyway`; `itineraryRevision` conflict detection remains the final
safety mechanism. Locks are instance-local, expire automatically, and are not
hard blocking.
Background Jobs v1 moves slow AI full generation and day/item regeneration to a
PostgreSQL-backed `trip_generation_jobs` queue processed by an in-process Trip
Service worker. The Web App creates jobs, shows a status card, polls job state,
and refetches the trip when the job completes. Jobs check
`expectedItineraryRevision` when queued and again through the final
revision-aware save, so newer itinerary edits are not overwritten; stale jobs
fail visibly with `itinerary_conflict`. There is no RabbitMQ, Kafka, Redis
queue, separate worker service, distributed locking, or progress streaming in
v1.
Activity Feed / Audit Log v1 records important successful actions on a trip
(creation, generation, edits, regenerations, version restores, comments,
collaborator changes, and share setting changes) as persistent rows in a
dedicated `trip_activity_events` table. The owner and accepted collaborators read
a chronological, newest-first feed via `GET /trips/{id}/activity` (cursor
paginated); pending/removed/non-collaborators get `404` and there is no public
route, so public share viewers never see activity. Events are recorded only
after an action succeeds, recording failures never fail the action, and metadata
is small and sanitized (no secrets, passwords, tokens, comment bodies, or full
itinerary JSON). The web app shows a `Recent activity` panel on private trip
detail pages. Real-time Activity Feed v1 adds an authenticated, fetch-based SSE
stream at `GET /trips/{id}/activity/stream`; connected private collaborators
receive best-effort `activity.created` events and the web app refetches the feed
live. The persistent activity endpoint remains the source of truth; the stream
is in-memory only, has no replay, and is never available to public share viewers.
Notification Service v1 lives in `services/notification-service` and owns
private, per-user in-app notifications in its own database. After a successful
collaboration/comment/itinerary action, Trip Service calls the Notification
Service **synchronously over HTTP** (internal `POST /internal/notifications/batch`,
authenticated with a shared `INTERNAL_SERVICE_TOKEN`) to create notifications for
the affected users — owner and accepted collaborators, never the actor
themselves. Notification creation is fail-open: a failure is logged and never
breaks the originating Trip Service action. Users read their own notifications
from user-facing endpoints (`GET /notifications`, `GET /notifications/unread-count`,
`PATCH /notifications/{id}/read`, `PATCH /notifications/read-all`) that require a
valid Auth Service JWT, so users only ever see their own notifications and public
share viewers have no access. The web app shows a header notification bell with a
real-time SSE-backed unread badge, polling fallback, a dropdown of recent
notifications, and a `/notifications` page; clicking a notification marks it
read and navigates to the related trip.
The Notification Service also supports **optional email notifications (v1)**: for
selected types (collaboration invited, comment created, collaborator role
changed/removed by default) it resolves the recipient's email from Auth Service
(internal `POST /internal/users/batch`) and sends a short email after the in-app
rows are created. Notification Preferences v1 lets each authenticated user
control global category preferences for in-app and email delivery through
`GET/PUT /notifications/preferences`; missing rows use defaults where in-app
categories are enabled and email trip updates are disabled. Email is behind a
provider switch (`EMAIL_PROVIDER=mock` by default — sends nothing externally;
`smtp` for real delivery) and is fail-open by default, so an email failure never
affects in-app notification creation. Real-time notification delivery uses
authenticated Server-Sent Events from Notification Service with an in-memory,
single-instance connection manager; polling remains the recovery path. No push,
WebSockets, RabbitMQ, background workers, per-trip notification preferences,
quiet hours, unsubscribe links, or digests in v1 — the synchronous HTTP design
is deliberately simple and replaceable by an event bus / async worker later.
User/Profile Service v1 lives in `services/user-service` and owns travel
profiles/preferences for authenticated users, also scoped by the JWT `sub`.
AI Planning Service owns itinerary generation and local travel knowledge.
When a user generates an itinerary, Trip Service fetches that user's profile and
preferences from User Service by forwarding the user's JWT, then sends optional
`userProfile` and `userPreferences` to AI Planning Service for prompt
personalization. Trip Service can also fetch a mock weather forecast from
External Integrations Service and forward optional `weatherForecast` context so
AI prompts can adapt to rain, heat, cold, or wind. After AI generation, Trip
Service can also call External Integrations Service to auto-attach
high-confidence place metadata to suitable itinerary items; enrichment is
optional and fail-open by default. Access tokens and full preference payloads
should not be logged.
External Integrations Service v1 lives in
`services/external-integrations-service` and owns place search/details, route
estimation, weather forecast, exchange-rate, and attraction/ticket price provider
boundaries.
Place search/details use the deterministic mock provider by default and can
optionally use Foursquare via `PLACE_PROVIDER=foursquare`; mock remains the local
no-key default. Routing and weather likewise support real providers behind clean
provider abstractions while keeping the `POST /routes/estimate` and
`GET /weather/forecast` contracts stable: real routing via OpenRouteService
(`ROUTE_PROVIDER=ors`) and real weather via OpenWeatherMap
(`WEATHER_PROVIDER=openweathermap`), each opt-in with mock as the default and the
fail-open fallback. Exchange-rate v1 exposes deterministic mock
`GET /exchange-rates/latest` and `GET /exchange-rates/convert` endpoints used by
Trip Service budget summaries; real exchange-rate provider names are reserved
for future adapters. Attraction/ticket price v1 exposes an internal deterministic
mock `POST /prices/estimate` endpoint used by Trip Service price enrichment; real
price provider fields are placeholders for future adapters. Results are cached in
a small in-memory TTL cache, and provider API keys never reach the Web App. The
Web App calls this service when
attaching optional place metadata to itinerary items, estimating per-day routes
via `POST /routes/estimate`, and showing trip weather via
`GET /weather/forecast`; Trip Service calls it for budget conversion and
generated-item ticket estimates. Route, weather, exchange-rate, and price data
are read-only; attached places can also carry optional local `openingHours`
intervals (`dayOfWeek` 1 Monday through 7 Sunday, `HH:mm` local time). The Web
App shows advisory closed-place warnings when hours are available and handles
missing real provider fields gracefully. No real Google Places provider, real
opening-hours provider, Google Maps routing, historical exchange rates, real
ticket booking/checkout provider, or trading-grade rate accuracy is enabled yet.
Calendar Sync v1 is implemented inside External Integrations Service rather
than a separate Calendar Service. Users can connect one Google Calendar account
through server-side OAuth, tokens are encrypted at rest, and Trip Service can
one-way sync timed itinerary items as Google Calendar events. Sync is per user
and per private trip; owners and editors can sync their own calendars, viewers
and public share viewers cannot. v1 uses the primary calendar only, the
`calendar.events` scope, and no two-way sync, webhooks, recurring events, or
Apple/Outlook providers.
Budget Tracking v1 gives each trip an optional structured budget
(`{ amount, currency }`) and each itinerary item an optional structured
`estimatedCost` (`{ amount, currency, category, confidence, source, note }`,
AI-estimated or manually edited). Trip Service computes a budget summary on
demand via `GET /trips/{id}/budget-summary` (estimated total, remaining/over,
daily and category breakdowns) and owners/editors set the budget via
`PUT /trips/{id}/budget` without touching the itinerary revision. Manual item
cost edits reuse the existing revision-checked `PUT /trips/{id}/itinerary` flow,
so conflict detection and version history apply unchanged. Multi-currency
Support v1 converts item and accommodation estimates into the trip budget
currency with External Integrations Service exchange rates, preserves original
currency totals, and reports conversion warnings when a cost cannot be
converted. The Web App shows a `BudgetPanel`, item cost badges, approximate
converted totals, and budget-aware quality warnings (over-budget trip/day,
expensive items, missing estimates, conversion gaps, and provider ticket-price
review hints). AI generation/regeneration prefers the trip/preferred currency but
may use local currencies when natural; the backend conversion is the source of
truth for budget totals. Trip Service can enrich likely ticketed attractions with
deterministic provider `estimatedCost` values after generation while preserving
manual costs by default. v1 has no real ticket booking/checkout provider,
historical rates, crypto rates, or financial accuracy guarantees, and never
exposes the private trip budget or provider review metadata on the public share
page.
Advanced AI Budget Optimization v1 lets owners/editors request a day-level
cheaper-itinerary proposal from AI without automatically overwriting the trip.
Trip Service queues a `budget_optimization_day` job, sends the selected day plus
budget, accommodation, weather, route, and preference context to AI Planning
Service `/optimize-budget/day`, stores the validated proposal in
`budget_optimization_proposals`, and shows it in the Web App for explicit review.
Applying a proposal uses `expectedItineraryRevision`, replaces only that day,
increments the revision once, and creates version/activity records; discarding
does not change the itinerary. Accepted viewers can read private proposals but
cannot create/apply/discard them, and public shares expose none of this UI or
API surface.
Accommodation Planning v1 adds one private structured stay location per trip.
Owners/editors can add, edit, or remove an accommodation with name/type/address,
optional attached place coordinates, check-in/check-out dates, notes, and an
optional accommodation `estimatedCost`; accepted viewers can read it on private
trips. Trip Service stores it as JSONB on `trips`, includes its cost in the
budget summary, forwards it to AI generation/regeneration, and records sanitized
activity events. The Web App shows an `Accommodation` panel, uses the stay as a
daily route start/end anchor when coordinates exist, includes it in private PDF
exports, and omits structured accommodation from public share pages/exports.
v1 has no hotel search provider, booking flow, multi-stay support, real booking
price feed, or check-in/out calendar events.

Web App v1 supports register/login/logout and stores tokens in `localStorage`
for development. Secure httpOnly cookies should replace localStorage token
storage before production.

Web App v1 lives in `apps/web`. The local full stack runs from
`infra/docker-compose.yml`, and the browser URL is `http://localhost:3000`.
Detailed full-stack instructions are in [infra/README.md](infra/README.md).

## Local Development

Local application infrastructure is defined in `infra/docker-compose.yml`; the main
compose file intentionally lives under `infra/`, not the project root.

Start with:

```bash
cp infra/.env.example infra/.env
./scripts/dev-setup.sh
```

Run the full app smoke test with:

```bash
./scripts/smoke-test.sh
```

The smoke test registers/logs in a unique user, checks profile/preferences
defaults and updates, creates and generates a trip with
`Authorization: Bearer <accessToken>`, exercises personalized generation,
registers a second user and verifies collaborator invite/accept/viewer/editor
permissions/removal,
searches mock places, checks mock route and weather endpoints, saves attached
place metadata with opening hours through Trip Service, verifies public trip
sharing create/status/password unlock/clear/disable behavior, verifies itinerary
version history and restore behavior, exercises itinerary comments
(create/list/counts/update/soft-delete, owner-deletes-any, collaborator-cannot-
delete-others, comments require auth, and public shares expose no comments),
verifies the activity feed records major actions and that
owner/accepted-collaborator can read it while pending/removed/non-collaborators,
unauthenticated requests, and the public share endpoint cannot,
checks presence state/snapshot access for owners, collaborators, removed
collaborators, and non-collaborators,
checks notification preferences can suppress and re-enable future comment
notifications, verifies itinerary revision conflict detection rejects stale
manual edits and day regeneration attempts,
exercises accommodation add/read/delete, budget inclusion, viewer read-only
permissions, activity feed events, and public-share redaction,
confirms only that user can access the trip and versions, and logs out.

See `infra/README.md` for direct Docker Compose commands, Ollama model pulls,
knowledge indexing, useful URLs, and troubleshooting. The full app can be
started with:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env up --build
```
