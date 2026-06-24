# Travel AI App

AI travel planning project with Go Auth Service, Go Trip Service, Go User
Service, Go External Integrations Service, Python/FastAPI AI Planning Service,
and a Next.js web app.

Auth Service v1 lives in `services/auth-service` and supports email/password
registration, login, refresh token rotation, logout, and JWT-backed `/auth/me`.
Trip Service validates those JWT access tokens locally with the shared
`JWT_ACCESS_SECRET` and scopes `/trips` data by the authenticated `sub` user ID.
Trip Service also records itinerary version snapshots after generation, manual
edits, partial regeneration, and restores; users can preview older versions and
restore them without deleting history. Authenticated trip owners can also create
one public read-only share link per trip. Public share links use opaque random
tokens, expose only sanitized trip/itinerary data at `/share/{shareToken}`, and
can be disabled by the owner.
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
estimation, and weather forecast provider boundaries. Place search/details use
the deterministic mock provider by default and can optionally use Foursquare via
`PLACE_PROVIDER=foursquare`; mock remains the local no-key default. The Web App
calls this service when attaching optional place metadata to itinerary items,
estimating per-day walking routes via `POST /routes/estimate` (mock provider:
Haversine × 1.25 at 5 km/h), and showing mock trip weather via
`GET /weather/forecast`. Route and weather data are read-only and approximate;
attached places can also carry optional local `openingHours` intervals
(`dayOfWeek` 1 Monday through 7 Sunday, `HH:mm` local time). The Web App shows
advisory closed-place warnings when hours are available and handles missing real
provider fields gracefully. No real Google Places provider, real opening-hours
provider, real weather provider, or real turn-by-turn routing is enabled yet.

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
searches mock places, checks mock route and weather endpoints, saves attached
place metadata with opening hours through Trip Service, verifies public trip
sharing create/view/disable behavior, verifies itinerary version history and
restore behavior, confirms only that user can access the trip and versions, and
logs out.

See `infra/README.md` for direct Docker Compose commands, Ollama model pulls,
knowledge indexing, useful URLs, and troubleshooting. The full app can be
started with:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env up --build
```
