# Travel AI Planner Web

Next.js Web App v1 for registering/logging in, managing profile and travel
preferences, creating trip requests, listing trips, opening trip details,
generating itineraries, viewing generated plans, and editing completed
itineraries. Completed trips with itineraries also show version history with
read-only preview and restore actions.

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
- `PUT /trips/{id}/itinerary`
- `POST /trips/{id}/itinerary/days/{dayNumber}/regenerate`
- `POST /trips/{id}/itinerary/days/{dayNumber}/items/{itemIndex}/regenerate`
- `GET /trips/{id}/itinerary/versions`
- `GET /trips/{id}/itinerary/versions/{versionId}`
- `POST /trips/{id}/itinerary/versions/{versionId}/restore`

The frontend calls External Integrations Service v1 directly for mock place
search while editing itinerary items:

- `GET /places/search?query=Colosseum&destination=Rome`
- `GET /places/{placeId}`

The Web App does not call third-party place APIs directly.

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
              "website": "https://example.com/colosseum"
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

To attach a real-place-shaped mock place, open a completed trip, click
`Edit itinerary`, click `Attach real place` on an item, search, select a result,
then click `Save`. Existing itinerary items without `place` metadata continue to
render normally. v1 intentionally has no opening hours, route optimization,
real route distance, flights, hotels, weather, or real Google Places provider.
See Distance / Walking Estimate below for the approximate straight-line distance
the Web App does calculate.

## Map View

Completed trips with itinerary items that have attached places and valid
coordinates show a read-only Map View on the trip detail page. The map uses
Leaflet with OpenStreetMap tiles and renders markers only for itinerary items
with numeric latitude and longitude values. Use the day filter to view all
mapped places or only the mapped places for a single day.

Attach places in itinerary edit mode first, then save or leave edit mode to see
them on the map. Map View v1 does not support route optimization, marker
dragging, or editing places from the map.

## Distance / Walking Estimate

Completed trips with itinerary items that have attached place coordinates show a
read-only Distance estimate panel below the Map View on the trip detail page. It
is fully frontend-only — no routing APIs, backend endpoints, or map provider
keys are involved.

- Uses the latitude/longitude on attached place metadata.
- Calculates approximate straight-line distance between consecutive mapped stops
  per day using the Haversine formula (Earth radius 6371 km).
- Estimates walking time using a flat 5 km/h pace.
- Compares each day total with `maxWalkingKmPerDay` from the User/Profile
  Service and shows a warning badge for days above the preference.
- It is **not** real route distance. Real walking distance may be higher, and
  the UI labels the figures as approximate.

Only itinerary items with a numeric, in-range `latitude` and `longitude` are
counted, so existing trips without place coordinates keep working. A day needs at
least two mapped stops before it contributes a distance. Preferences are fetched
non-blocking: if the request fails the estimates still render, just without the
preference comparison. Distance estimates are hidden during itinerary edit mode
and reappear after saving or leaving edit mode. The estimate logic lives in
`src/lib/itinerary/distance-utils.ts` as pure functions covered by
`distance-utils.test.ts`.

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
