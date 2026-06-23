# Travel AI Planner Web

Next.js Web App v1 for registering/logging in, managing profile and travel
preferences, creating trip requests, listing trips, opening trip details,
generating itineraries, and viewing generated plans.

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
TRIP_SERVICE_INTERNAL_URL=http://localhost:8080
```

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
- `GET /trips/{id}`
- `POST /trips/{id}/generate`

The frontend calls the protected User Service endpoints from `/settings`:

- `GET /users/me/profile`
- `PUT /users/me/profile`
- `GET /users/me/preferences`
- `PATCH /users/me/preferences`

Browser requests go through the Next.js route proxy at `/api/trip-service/*`,
which forwards to `TRIP_SERVICE_INTERNAL_URL` when set, then falls back to
`NEXT_PUBLIC_TRIP_SERVICE_URL`. In Docker Compose, the browser-facing URL stays
`http://localhost:8080` while server-side proxy calls use
`http://trip-service:8080`. The proxy forwards the `Authorization` header.

Auth Service, Trip Service, and User Service enable CORS for
`http://localhost:3000`, so direct browser calls to
`NEXT_PUBLIC_AUTH_SERVICE_URL`, `NEXT_PUBLIC_TRIP_SERVICE_URL`, and
`NEXT_PUBLIC_USER_SERVICE_URL` remain possible during local development.

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
```
