# 6. Add Multi-Destination & Multi-Modal Travel Planning v1: support trips with multiple stops, transfer legs between destinations, transport modes such as car/train/bus/flight/boat/bike/hiking, route builder UI, AI generation for transfer days, budget/route estimates, and map display

You are a senior full-stack engineer and product-minded AI architect. Continue building the web-based AI travel planning application.

Your task:
Implement Multi-Destination & Multi-Modal Travel Planning v1: support trips with multiple stops, transfer legs between destinations, transport modes such as car/train/bus/flight/boat/bike/hiking, route builder UI, AI generation for transfer days, budget/route estimates, and map display.

Context:
We already have a microservices-based AI travel planning app.

Existing services:

- Auth Service:
  - Go microservice
  - issues JWT access tokens and refresh tokens
- User/Profile Service:
  - Go microservice
  - stores user profile/preferences
  - stores preferredLanguage
  - owns workspace membership and workspace roles
- Trip Service:
  - Go microservice
  - owns trips, workspace trips, trip creation, itinerary generation jobs, AI trip discovery, AI template adaptation jobs, AI repair jobs, budgets, workspace budgets, cost analytics, cost splitting, approval workflow, workspace policies, approval risk scoring, comments, activity, version history, templates, conflict detection, notifications integration, and permissions
  - supports personal trips and workspace trips
  - calls AI Planning Service for generation/regeneration/optimization/adaptation/repair/discovery
- Worker Service:
  - Go microservice
  - processes RabbitMQ-backed jobs
- Notification Service:
  - Go microservice
  - owns in-app/email/web-push/SSE notifications and notification preferences
- External Integrations Service:
  - Go microservice
  - owns places, routes, weather, calendar, exchange rates, prices, availability provider adapters, quota/rate limits
- AI Planning Service:
  - Python FastAPI service
  - supports itinerary generation, partial regeneration, budget optimization, template adaptation, policy-aware repair, trip discovery, destination context/RAG, validation/repair, multilingual output, and Ollama/mock modes
- Web App:
  - Next.js app under apps/web
  - supports auth, trips, workspace switcher, workspace pages, trip discovery, templates, AI template adaptation, budgets, workspace budgets, cost analytics, cost splitting, approval workflow, workspace policies, approval risk scoring, AI repair proposals, availability cards, exports, offline mode, PWA install, notifications, internationalization, etc.
- Infra:
  - Postgres
  - RabbitMQ
  - Prometheus/Grafana

Current behavior:

- Trips are mostly modeled as one destination/city.
- Itinerary days contain activities inside one city.
- Route/walking estimates mostly assume walking inside a single destination.
- Create trip flow assumes the user provides one destination.
- AI Trip Discovery can suggest destinations but should now be able to suggest route-style trips too.

Problem:
Real travel is often multi-destination and multi-modal:

- Bratislava → Vienna → Salzburg → Hallstatt
- Barcelona → Valencia → Madrid
- Paris → Brussels → Amsterdam
- Tokyo → Kyoto → Osaka
- Road trips with several towns
- Train/backpacking routes
- Camping/hiking trips
- Island hopping by ferry/boat

The app needs to support:

- multiple stops/towns/cities in one trip
- transfer days between stops
- transport modes beyond walking
- car/train/bus/flight/boat/bike/hiking/public transport
- camping/hiking/adventure trip styles
- route-level budget and timing estimates

Goal:
Add Multi-Destination & Multi-Modal Travel Planning v1:

- A trip can have one or more route stops.
- Backward compatibility: old/single-city trips still work.
- Multi-destination trips include transfer legs between stops.
- Users can build/reorder route stops in the create trip page.
- Users can specify transport preferences and trip style.
- AI generation understands route stops and transfer days.
- Itinerary can contain transfer items/legs.
- Budget summary includes transfer costs.
- Map view shows route stops and transfer lines.
- Route validation warns about unrealistic routes.
- AI Trip Discovery can suggest multi-city routes, not only single destinations.

Do NOT add:

- real ticket booking
- train ticket purchase
- live flight search/prices
- hotel booking
- car rental checkout
- boat rental checkout
- camping permit booking
- advanced GPS hiking route generation
- turn-by-turn navigation
- offline navigation
- complex route optimization across countries
- legal/visa guarantees
- new backend service
- Kubernetes

For v1:

- Support structured route stops and transfer legs.
- Support planned transport modes.
- Use mock/estimated transfer durations and costs if real route provider does not support mode.
- Keep real provider integration optional/fail-open.
- Use AI to plan realistic transfer days.
- Keep user in control.
- Existing trips must not break.
- Single-destination trips should be represented as route with one stop internally if practical.
- Multi-destination trips should work with budgets, exports, map, approval/risk/policy as much as practical.

Important codebase consistency requirement:
Before implementing, inspect existing services and follow the same patterns exactly:

- services/trip-service
- services/ai-planning-service
- services/external-integrations-service
- services/worker-service
- services/user-service
- services/notification-service
- apps/web

Do not invent a different architecture if the repository already has conventions.

Match existing patterns for:

- migrations
- sqlc
- pgxpool
- Go modules
- Uber Fx
- Zap logging
- config/env
- HTTP handlers
- middleware
- JWT/trip/workspace permission checks
- response/error helpers
- itinerary schema validation
- version history
- activity events
- budget summary
- map view
- route estimate client
- AI request building
- FastAPI schemas/routes
- Ollama/mock mode
- i18n
- frontend API clients/hooks
- TanStack Query
- forms
- UI components
- tests
- smoke scripts
- docs

Part 1: Core domain model

1. Add route model to trips.

A trip should support:

{
"route": {
"origin": {
"name": "Bratislava",
"country": "Slovakia",
"coordinates": {
"lat": 48.1486,
"lng": 17.1077
}
},
"returnToOrigin": false,
"stops": [
{
"id": "stop_1",
"destination": "Vienna",
"city": "Vienna",
"country": "Austria",
"arrivalDate": "2026-09-10",
"departureDate": "2026-09-12",
"nights": 2,
"coordinates": {
"lat": 48.2082,
"lng": 16.3738
},
"accommodationHint": "hotel",
"notes": null
},
{
"id": "stop_2",
"destination": "Salzburg",
"city": "Salzburg",
"country": "Austria",
"arrivalDate": "2026-09-12",
"departureDate": "2026-09-14",
"nights": 2,
"coordinates": {
"lat": 47.8095,
"lng": 13.0550
},
"accommodationHint": "guesthouse",
"notes": null
}
],
"legs": [
{
"id": "leg_1",
"fromStopId": "origin",
"toStopId": "stop_1",
"fromName": "Bratislava",
"toName": "Vienna",
"mode": "train",
"departureDate": "2026-09-10",
"estimatedDurationMinutes": 70,
"estimatedDistanceKm": 80,
"estimatedCost": {
"amount": 18,
"currency": "EUR",
"category": "transport",
"confidence": "medium",
"source": "ai"
},
"notes": "Direct regional train recommended.",
"providerMetadata": null
}
],
"preferences": {
"preferredModes": ["train", "public_transport"],
"avoidModes": ["flight"],
"carAvailable": false,
"maxTransferHoursPerDay": 4,
"tripStyles": ["train_trip", "city_break"]
}
}
}

2. Backward compatibility.

Existing single-city trips:

- continue working with existing destination field.
- route can be null.
- OR route is auto-derived as one stop.
  Recommended:
- Add nullable route_json JSONB column to trips.
- Keep existing destination field.
- New multi-destination trips set route_json.
- Existing APIs include route if present.
- UI treats route null as single-destination.

3. Migration.

Trip Service migration:
ALTER TABLE trips ADD COLUMN route_json JSONB NULL;

Optional:
ALTER TABLE trips ADD COLUMN trip_type TEXT NOT NULL DEFAULT 'single_destination';

Allowed trip_type:

- single_destination
- multi_destination

If adding trip_type:

- existing trips default single_destination.
- multi-stop trips set multi_destination.

4. Route validation.

Validate:

- stops length 1–20.
- stop destination required.
- arrivalDate/departureDate valid if present.
- departureDate >= arrivalDate.
- nights >= 0.
- legs connect valid stop IDs or origin.
- supported transport mode.
- maxTransferHoursPerDay reasonable, 1–24.
- coordinates optional but if present lat/lng valid.
- tripStyles supported.
- route dates should not contradict trip start/end/duration if those exist.

5. Transport mode enum.

Supported v1 modes:

- walk
- car
- rental_car
- train
- bus
- flight
- boat
- ferry
- bike
- public_transport
- hiking
- other

6. Trip style enum.

Supported v1 trip styles:

- city_break
- road_trip
- train_trip
- backpacking
- camping
- hiking
- island_hopping
- nature
- beach
- food
- culture
- adventure
- family
- romantic
- low_budget
- luxury
- hidden_gem

7. Accommodation hint enum.

Supported:

- hotel
- hostel
- apartment
- guesthouse
- campsite
- cabin
- campervan
- home
- other
- unknown

Part 2: Itinerary schema updates

8. Support transfer itinerary items.

Add or extend itinerary item type:

- transfer

Transfer item shape:

{
"type": "transfer",
"name": "Train from Vienna to Salzburg",
"description": "Travel from Vienna to Salzburg by train, then check in and take a relaxed evening walk.",
"startTime": "09:30",
"endTime": "12:00",
"transfer": {
"legId": "leg_2",
"from": "Vienna",
"to": "Salzburg",
"mode": "train",
"estimatedDurationMinutes": 150,
"estimatedDistanceKm": 295,
"estimatedCost": {
"amount": 35,
"currency": "EUR",
"category": "transport",
"confidence": "medium",
"source": "ai"
},
"bookingRequired": false,
"notes": "Check train times before travel."
},
"estimatedCost": {
"amount": 35,
"currency": "EUR",
"category": "transport",
"confidence": "medium",
"source": "ai"
}
}

9. Day location.

Each itinerary day should optionally include:

- primaryStopId
- locationName
- transferDay boolean

Example:

{
"dayNumber": 3,
"date": "2026-09-12",
"title": "Transfer to Salzburg",
"primaryStopId": "stop_2",
"locationName": "Salzburg",
"transferDay": true,
"items": [...]
}

10. Existing itinerary validation.

Update validators to accept:

- transfer item type
- transfer object
- transport modes
- estimatedCost category transport
- day.primaryStopId/locationName/transferDay if present

Do not break old itinerary items.

Part 3: Trip Service API

11. Update trip create/update DTOs.

Trip creation should accept either:

- destination for single-city trip
- route for multi-destination trip

Request example:

{
"tripType": "multi_destination",
"title": "Austria train route",
"destination": "Austria",
"startDate": "2026-09-10",
"days": 5,
"budget": {
"amount": 900,
"currency": "EUR"
},
"travelers": 2,
"route": {...}
}

Rules:

- If tripType=single_destination, destination required.
- If tripType=multi_destination, route.stops required.
- destination can be derived as “Austria route” or first/primary stop.
- trips list should display route summary for multi-destination trips.

12. Add route endpoints.

Add endpoints:

GET /trips/{tripId}/route
PUT /trips/{tripId}/route

PUT request:
{
"expectedItineraryRevision": 12,
"route": {...}
}

Permissions:

- owner/editor can update route.
- viewer read-only.
- public share read-only if route included in public trip.

Behavior:

- Updating route is material if itinerary exists.
- Should increment itineraryRevision only if route is stored as part of itinerary? Recommended:
  - route update should increment routeRevision or update trip metadata but also mark itinerary potentially stale.
  - Simpler v1: route update increments itineraryRevision only if itinerary depends on route.
- If approval pending/approved, reset to draft if route materially changes.
- Create activity event route_updated.
- Expire stale repair proposals if needed.

13. Public share behavior.

Public trip share may include sanitized route:

- origin name
- stops
- legs
- transport modes
- durations/costs if itinerary exposes them

Do not expose:

- private notes
- internal provider metadata
- user IDs
- workspace policy metadata

14. Version history.

Itinerary versions should include route snapshot if route affects itinerary.
If existing version only stores itinerary JSON, add metadata routeSnapshot if useful.

Part 4: Transfer estimates / External Integrations

15. Extend route estimate endpoint.

Existing:
POST /routes/estimate

Extend request to support modes:

- car
- train
- bus
- flight
- boat/ferry
- bike
- hiking
- public_transport

Request:
{
"from": {
"name": "Vienna",
"lat": 48.2082,
"lng": 16.3738
},
"to": {
"name": "Salzburg",
"lat": 47.8095,
"lng": 13.0550
},
"mode": "train",
"date": "2026-09-12",
"currency": "EUR"
}

Response:
{
"mode": "train",
"estimatedDistanceKm": 295,
"estimatedDurationMinutes": 150,
"estimatedCost": {
"amount": 35,
"currency": "EUR",
"category": "transport",
"confidence": "low",
"source": "mock"
},
"provider": "mock",
"fallbackUsed": true,
"warnings": [
"This is an estimate, not a live schedule."
]
}

16. Mock estimator.

Implement deterministic estimates:

- walk: distance / 5 km/h
- bike: distance / 15 km/h
- hiking: distance / 3.5 km/h plus terrain warning
- car/rental_car: distance / 80 km/h plus 20 min buffer
- bus: distance / 60 km/h plus 30 min buffer
- train: distance / 100 km/h plus 20 min buffer
- flight: fixed airport overhead 180 min + flight distance / 700 km/h
- ferry/boat: distance / 35 km/h plus 30 min buffer
- public_transport: distance / 35 km/h plus 30 min buffer
- other: distance / 50 km/h

Cost estimates:

- walk: 0
- bike: low/0 if user-owned, else estimate
- hiking: 0
- car/rental_car: fuel estimate distance \* 0.18 EUR/km, rental_car add optional daily estimate if existing
- bus: distance \* 0.08 EUR/km
- train: distance \* 0.12 EUR/km
- flight: max(50, distance \* 0.15 EUR/km)
- ferry/boat: distance \* 0.20 EUR/km
- public_transport: distance \* 0.10 EUR/km
- boat rental should be an activity/accommodation-style estimate, not ordinary transfer booking in v1.

17. Real provider support.

If existing OpenRouteService supports only walking/driving/cycling:

- map car/rental_car to driving.
- bike to cycling.
- walk/hiking to walking if safe.
- train/bus/flight/ferry should use mock estimate in v1 with warning.
- Do not pretend live schedules exist.

18. Trip Service route estimation client.

Trip Service should be able to:

- estimate all route legs
- update route_json legs with estimates
- fail open with warnings if provider fails

Config:

- MULTI_DESTINATION_ENABLED=true
- ROUTE_LEG_ESTIMATION_ENABLED=true
- ROUTE_LEG_ESTIMATION_FAIL_OPEN=true
- ROUTE_LEG_MAX_STOPS=20
- ROUTE_LEG_TIMEOUT_SECONDS=8

Part 5: AI Planning Service updates

19. Update generation schemas.

Add route context to generation requests:

- route
- transport preferences
- trip styles
- max transfer time
- camping/hiking preferences

Request excerpt:
{
"route": {...},
"transportPreferences": {
"preferredModes": ["train"],
"avoidModes": ["flight"],
"carAvailable": false,
"maxTransferHoursPerDay": 4
},
"tripStyles": ["train_trip", "city_break"]
}

20. AI prompt rules.

Prompt should instruct:

- Plan across all route stops.
- Respect arrival/departure dates and nights per stop.
- Include transfer items on transfer days.
- Do not schedule dense sightseeing before/after long transfers.
- Use selected transport modes.
- Avoid disallowed transport modes.
- Add realistic rest after long travel.
- For camping trips, include campsite/accommodation-style notes but do not claim reservations.
- For hiking trips, include conservative day planning and safety notes, but do not generate technical GPS routes.
- For boat/ferry/island hopping, include transfer estimates as approximate and warn to verify schedules.
- Keep costs as estimates.
- Do not claim tickets/bookings are confirmed.
- Output in requested language.

21. Mock generation behavior.

For route with multiple stops:

- Generate days assigned to stops.
- Insert transfer item when day changes stop.
- Use route leg mode in transfer item.
- Produce deterministic simple itinerary.

Example:
Day 1: Arrival in Vienna
Day 2: Explore Vienna
Day 3: Transfer Vienna → Salzburg by train + Salzburg evening
Day 4: Explore Salzburg
Day 5: Return / relaxed final day

22. Template adaptation / repair / discovery.

Update AI Planning Service schemas/prompts:

- Template adaptation can adapt templates into route-style trips if route provided.
- Policy-aware repair can repair route-related issues.
- Trip discovery can suggest route suggestions, not only single destinations.

Part 6: AI Trip Discovery integration

23. Support route suggestions.

Extend destination suggestion response with optional route:

{
"suggestionType": "single_destination" | "route",
"destination": "Austria train route",
"route": {
"origin": {...},
"stops": [...],
"legs": [...],
"preferences": {...}
}
}

24. Discovery cards.

For route suggestions, cards should show:

- route title
- stops sequence
- transport mode
- estimated total route duration
- estimated transfer cost
- why it fits
- downsides
- “Use this route”

25. Create trip from route suggestion.

When user selects route suggestion:

- create multi_destination trip
- store route_json
- optionally auto-generate itinerary

Part 7: Budget integration

26. Transport cost category.

Ensure estimatedCost supports:

- category: transport

Transfer legs should contribute to:

- trip budget summary
- cost analytics
- workspace budget summary
- cost splitting if travelers > 1

27. Avoid double counting.

If a transfer leg has estimatedCost and itinerary transfer item also has estimatedCost:

- decide one source of truth.
  Recommended:
- Itinerary transfer item estimatedCost is included in budget summary.
- Route leg cost is used to prefill transfer item and route display.
- Avoid counting route leg separately if equivalent transfer item exists.

28. Accommodation/camping.

Camping as accommodation style:

- campsite accommodation cost may be stored in accommodation model or itinerary estimatedCost.
- Do not implement full campsite booking.
- Budget can include campsite cost as accommodation if user adds it.

Part 8: Policy/risk/repair integration

29. Workspace policy additions.

Extend workspace policy rules optionally:

- maxTransferHoursPerDay
- disallowedTransportModes
- preferredTransportModes already exists
- requireCarAvailableForRoadTrip optional
- maxTransportBudget optional

If too much for v1:

- add only:
  - maxTransferHoursPerDay
  - disallowedTransportModes

30. Policy evaluator.

Evaluate:

- transfer legs exceeding maxTransferHoursPerDay.
- disallowed transport modes used.
- flight used when disallowed.
- transport budget over threshold if implemented.

31. Approval risk scoring.

Add risk factors:

- too_many_stops_for_duration
- long_transfer_day
- disallowed_transport_mode
- route_estimate_missing
- high_transport_cost
- hiking_day_too_dense
- camping_accommodation_missing

32. AI repair.

Repair can suggest:

- remove a stop
- change transport mode
- add rest after transfer
- reduce route complexity
- replace flight with train if feasible
- split long transfer across days

Do not auto-apply.

Part 9: Web App route builder UI

33. Create trip page.

Update /trips/new.

Modes:

- Single destination
- Multi-destination route
- Help me choose

Multi-destination route builder:

- origin input
- stops list
- add stop
- remove stop
- reorder stops
- nights per stop
- arrival/departure date optional
- transport mode per leg
- route preferences

34. Components.

Create:

apps/web/components/routes/TripRouteBuilder.tsx
apps/web/components/routes/RouteStopCard.tsx
apps/web/components/routes/RouteLegCard.tsx
apps/web/components/routes/TransportModeSelector.tsx
apps/web/components/routes/TripStyleSelector.tsx
apps/web/components/routes/RouteSummaryCard.tsx
apps/web/components/routes/RouteValidationWarnings.tsx

35. Transport mode selector.

Show modes:

- Walking
- Car
- Rental car
- Train
- Bus
- Flight
- Ferry/boat
- Bike
- Hiking
- Public transport
- Other

Use icons if existing icon system supports it.

36. Trip style selector.

Chips:

- Road trip
- Train trip
- Backpacking
- Camping
- Hiking
- Island hopping
- Nature
- Beach
- Food
- Culture
- Adventure
- Family
- Romantic
- Low budget
- Hidden gem

37. Route validation UI.

Warn:

- “5 stops in 3 days may feel rushed.”
- “This transfer is longer than your max transfer time.”
- “Flight is selected but flights are in your avoid list.”
- “Camping selected, but no campsite/accommodation stop is configured.”
- “Hiking selected; routes are approximate and should be checked with local maps.”

38. Create trip dialog/submit.

When submitting:

- validate route.
- create trip with tripType=multi_destination.
- optionally auto-generate itinerary.

Part 10: Trip detail UI

39. Route overview panel.

Add route overview on trip detail:

- route sequence
- stops with dates/nights
- legs with mode/duration/cost
- warnings
- edit route button for editors

40. Itinerary UI.

Transfer items should render differently:

- transport icon
- from → to
- mode
- estimated duration
- estimated cost
- warning “Verify schedule before travel”

41. Map view.

Update map to show:

- markers for stops
- numbered stop order
- lines between stops
- existing activity markers if coordinates exist

If exact route geometry unavailable:

- draw straight lines with dashed/approx style.
- label as approximate.

42. Route editing.

Editors can update route.
If itinerary exists:

- show warning:
  “Changing the route may make the current itinerary outdated. Regenerate affected days after saving.”

V1 can update route without automatically rewriting itinerary.

Part 11: Frontend API/types/hooks

43. Types.

Create/update:

apps/web/types/route.ts

Types:

- TripRoute
- RouteStop
- RouteLeg
- TransportMode
- TripStyle
- AccommodationHint
- RouteValidationWarning

Update trip types to include:

- tripType
- route

44. API client.

Create/update:
apps/web/lib/api/trip-routes.ts

Functions:

- getTripRoute(tripId)
- updateTripRoute(tripId, input)

Update createTrip API to accept tripType/route.

45. Hooks.

Create:

- useTripRoute
- useUpdateTripRoute

Update:

- useCreateTrip
- useGenerateTrip
- useTrip
- useTripList if route summary displayed

Part 12: Exports/calendar/offline

46. PDF export.

Include:

- route overview
- stops
- transfer legs
- transport warnings

47. CSV export.

Include transfer items and transport mode/cost columns.

48. ICS export.

Transfer items should become calendar events if timed.
Title:
“Transfer: Vienna → Salzburg”
Description includes mode/duration/warnings.

49. Offline mode.

Cached trip should include route_json.
Offline editing route can be disabled in v1 unless existing offline mutation architecture supports it.

Part 13: Internationalization

50. Add translation keys.

Namespaces:

- routes
- transportModes
- tripStyles

Translate to:

- en
- es
- uk
- fr

Keys:

- Multi-destination route
- Add stop
- Remove stop
- Reorder
- Origin
- Stop
- Transfer
- Transport mode
- Road trip
- Train trip
- Camping
- Hiking
- Ferry/boat
- Estimated duration
- Estimated cost
- Verify schedules before travel
- Route may be unrealistic

Part 14: Backend tests

51. Trip Service tests.

Test:

- create single destination trip still works.
- create multi-destination trip with route works.
- invalid route rejected.
- unsupported transport mode rejected.
- get/update route permissions.
- viewer cannot update route.
- public share route sanitized.
- route update resets approval if needed.
- route update creates activity.
- generation request includes route.
- existing trip APIs do not break when route null.

52. Itinerary validation tests.

Test:

- transfer item accepted.
- invalid transfer mode rejected.
- transfer estimatedCost category transport accepted.
- old itinerary items still accepted.

53. Budget tests.

Test:

- transfer item costs included in budget.
- route leg cost not double-counted if matching transfer item exists.
- transport category appears in analytics.

54. Policy/risk tests.

Test:

- disallowed transport mode violation.
- max transfer hours violation.
- risk factor for long transfer.
- personal trip behavior unchanged.

55. External Integration tests.

Test:

- route estimate for car/train/bus/flight/ferry returns deterministic mock.
- unsupported mode rejected.
- ORS provider maps car/bike/walk where available.
- train/flight/ferry fallback to mock with warning.

56. AI Planning Service tests.

Test:

- mock generation with route produces transfer days.
- route prompt includes stops/legs/modes.
- no booking claims.
- multilingual output still works.
- old single-destination generation still works.

Part 15: Frontend tests

57. Route builder tests.

Test:

- add/remove/reorder stop.
- select transport mode.
- select trip style.
- route validation warnings.
- submit multi-destination create request.

58. Trip detail tests.

Test:

- route overview renders stops/legs.
- transfer item card renders mode/from/to/duration/cost.
- map receives route stops/lines.
- edit route button hidden for viewer.

59. Discovery tests.

Test:

- route suggestion card renders stop sequence.
- create trip from route suggestion sends route.

60. i18n tests.

Test route/transport/trip style labels in all four languages.

Part 16: Smoke tests

61. Update scripts/smoke-test.sh.

API smoke:

1. Login user.
2. Create multi-destination trip:
   - origin Bratislava
   - stops Vienna and Salzburg
   - train leg
3. Assert trip route exists.
4. Trigger itinerary generation in mock mode.
5. Assert itinerary has transfer item.
6. Assert budget summary includes transport cost.
7. Update route with car leg.
8. Assert activity created and route updated.
9. Create single-destination trip and assert old flow still works.
10. Request route estimate for train/bus/flight/ferry and assert response/warnings.

11. Update scripts/web-smoke-test.md.

Manual test:

1. Open /trips/new.
2. Select Multi-destination route.
3. Add origin and 3 stops.
4. Select train/car transport modes.
5. Select trip styles: train trip + hiking.
6. Create trip and auto-generate itinerary.
7. Confirm route overview appears.
8. Confirm transfer day appears.
9. Confirm map shows stop markers and route lines.
10. Confirm budget includes transfer cost.
11. Edit route and confirm warning about outdated itinerary.
12. Confirm single-destination create still works.

Part 17: Documentation

63. Update Trip Service README.

Document:

- route_json schema
- trip_type
- route endpoints
- transfer itinerary items
- budget behavior
- policy/risk integration
- public share sanitization
- limitations

64. Update External Integrations README.

Document:

- route estimate modes
- mock estimator rules
- provider fallback
- warnings for non-live schedules

65. Update AI Planning Service README.

Document:

- route context in generation requests
- transfer day behavior
- camping/hiking constraints
- no booking/live schedule claims

66. Update Web App README.

Document:

- multi-destination route builder
- transport mode selector
- trip style selector
- route overview/map
- limitations

67. Update root README.md.

Mention:

- Multi-Destination & Multi-Modal Travel Planning v1.

68. User-facing limitations.

Document:

- transport durations/costs are estimates.
- no live train/bus/flight/ferry schedules in v1.
- no booking or ticket purchase.
- hiking/camping suggestions require user verification.
- route lines on map may be approximate.
- changing route does not automatically rewrite the whole itinerary unless user regenerates.

Part 18: Security and quality requirements

- Existing single-destination trips must not break.
- Route update must enforce trip edit permissions.
- Public share must sanitize route metadata.
- Do not expose internal provider metadata.
- Do not claim live schedules/prices unless real provider explicitly supports them.
- Do not book transport/accommodation.
- Do not generate technical hiking navigation or safety guarantees.
- AI output must be validated before saving.
- Transport cost must avoid double counting.
- Approval/risk/policy behavior must remain consistent.
- Keep code consistent with existing service patterns.
- Keep tests and docs updated.

Expected output:
Trips can be single-destination or multi-destination.
Multi-destination trips store route stops, transfer legs, transport modes, and trip styles.
Create Trip page includes a route builder with stops, transport modes, and route validation warnings.
AI generation can create transfer days and multi-stop itineraries.
External Integrations can estimate transfer durations/costs for supported modes with mock/fallback behavior.
Trip detail shows route overview, transfer items, and route map.
Budget, policy, approval risk, exports, discovery, and public share support route data where practical.
Existing single-city trips continue working.
Docs, tests, and smoke tests are updated.
