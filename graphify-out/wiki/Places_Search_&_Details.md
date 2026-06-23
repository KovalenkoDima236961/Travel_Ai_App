# Places Search & Details

> 31 nodes · cohesion 0.12

## Key Concepts

- **places.go** (20 connections) — `services/external-integrations-service/internal/application/service/places.go`
- **routes_test.go** (18 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **newTestRouter()** (12 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **performRequest()** (10 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **PlacesHandler** (4 connections) — `services/external-integrations-service/internal/http-server/handler/places.go`
- **writeError()** (4 connections) — `services/external-integrations-service/internal/http-server/handler/places.go`
- **writeJSON()** (4 connections) — `services/external-integrations-service/internal/http-server/handler/places.go`
- **.Details()** (3 connections) — `services/external-integrations-service/internal/http-server/handler/places.go`
- **.Search()** (3 connections) — `services/external-integrations-service/internal/http-server/handler/places.go`
- **placeFetch()** (3 connections) — `apps/web/src/lib/api/places.ts`
- **TestGetDetailsReturnsPlace()** (3 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **TestGetDetailsUnknownIDReturnsNotFound()** (3 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **TestHealthReturnsOK()** (3 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **TestReadyReturnsOK()** (3 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **TestSearchColosseumRomeReturnsColosseum()** (3 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **TestSearchDestinationFiltersResults()** (3 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **TestSearchIsCaseInsensitive()** (3 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **TestSearchMissingQueryReturnsBadRequest()** (3 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **TestSearchUnknownQueryReturnsCityFallback()** (3 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **PlacesService** (3 connections) — `services/external-integrations-service/internal/application/service/places.go`
- **getPlaceDetails()** (2 connections) — `apps/web/src/lib/api/places.ts`
- **searchPlaces()** (2 connections) — `apps/web/src/lib/api/places.ts`
- **testConfig()** (2 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **TestCORSPreflightWorks()** (2 connections) — `services/external-integrations-service/internal/http-server/routes_test.go`
- **.RegisterRoutes()** (1 connections) — `services/external-integrations-service/internal/http-server/handler/places.go`
- *... and 6 more nodes in this community*

## Relationships

- No strong cross-community connections detected

## Source Files

- `apps/web/src/lib/api/places.ts`
- `services/external-integrations-service/internal/application/service/places.go`
- `services/external-integrations-service/internal/http-server/handler/places.go`
- `services/external-integrations-service/internal/http-server/routes_test.go`

## Audit Trail

- EXTRACTED: 126 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*