# Travel AI Planner Web

Next.js Web App v1 for creating trip requests, listing trips, opening trip details, generating itineraries, and viewing generated plans.

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

The app expects the Trip Service URL in:

```bash
NEXT_PUBLIC_TRIP_SERVICE_URL=http://localhost:8080
TRIP_SERVICE_INTERNAL_URL=http://localhost:8080
```

## Backend

Start the repository backend services first, then run the web app. The frontend calls the Trip Service endpoints:

- `POST /trips`
- `GET /trips?limit=20&offset=0`
- `GET /trips/{id}`
- `POST /trips/{id}/generate`

Browser requests go through the Next.js route proxy at `/api/trip-service/*`,
which forwards to `TRIP_SERVICE_INTERNAL_URL` when set, then falls back to
`NEXT_PUBLIC_TRIP_SERVICE_URL`. In Docker Compose, the browser-facing URL stays
`http://localhost:8080` while server-side proxy calls use
`http://trip-service:8080`.

Trip Service also enables CORS for `http://localhost:3000`, so direct browser
calls to `NEXT_PUBLIC_TRIP_SERVICE_URL` remain possible during local
development.

The current Trip Service validates the most active pace as `packed`; the UI labels that option as `Intensive`.

## Commands

```bash
npm run dev
npm run build
npm run start
npm run typecheck
```
