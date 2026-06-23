# Auth Service & Web Auth Integration

> 83 nodes · cohesion 0.03

## Key Concepts

- **auth.go** (61 connections) — `services/auth-service/internal/infrastructure/repository/postgres/dto/auth.go`
- **itinerary_version.go** (16 connections) — `services/trip-service/internal/domain/entity/itinerary_version.go`
- **postgres.go** (16 connections) — `services/auth-service/pkg/storage/postgres/postgres.go`
- **errs.go** (15 connections) — `services/auth-service/internal/domain/errs/errs.go`
- **dto.go** (12 connections) — `services/external-integrations-service/internal/http-server/handler/dto.go`
- **DB** (9 connections) — `services/auth-service/pkg/storage/postgres/postgres.go`
- **authFetch()** (7 connections) — `apps/web/src/lib/api/auth.ts`
- **preferences.go** (6 connections) — `services/user-service/internal/domain/entity/preferences.go`
- **repository.go** (5 connections) — `services/user-service/internal/infrastructure/repository/postgres/repository.go`
- **itineraryVersionFromScannedValues()** (4 connections) — `services/trip-service/internal/infrastructure/repository/postgres/dto/itinerary_version.go`
- **New()** (3 connections) — `services/auth-service/pkg/storage/postgres/postgres.go`
- **AuthApiError** (2 connections) — `apps/web/src/lib/api/auth.ts`
- **buildAuthUrl()** (2 connections) — `apps/web/src/lib/api/auth.ts`
- **login()** (2 connections) — `apps/web/src/lib/api/auth.ts`
- **logout()** (2 connections) — `apps/web/src/lib/api/auth.ts`
- **me()** (2 connections) — `apps/web/src/lib/api/auth.ts`
- **NewAuth()** (2 connections) — `services/auth-service/internal/http-server/dto/response/auth.go`
- **NewUser()** (2 connections) — `services/auth-service/internal/http-server/dto/response/auth.go`
- **refresh()** (2 connections) — `apps/web/src/lib/api/auth.ts`
- **DependencyError** (2 connections) — `services/trip-service/internal/application/errs/errs.go`
- **InvalidInputError** (2 connections) — `services/auth-service/internal/application/errs/errs.go`
- **ItineraryVersionInsertValues()** (2 connections) — `services/trip-service/internal/infrastructure/repository/postgres/dto/itinerary_version.go`
- **marshalMetadata()** (2 connections) — `services/trip-service/internal/infrastructure/repository/postgres/dto/itinerary_version.go`
- **ScanItineraryVersion()** (2 connections) — `services/trip-service/internal/infrastructure/repository/postgres/dto/itinerary_version.go`
- **ScanItineraryVersionSummary()** (2 connections) — `services/trip-service/internal/infrastructure/repository/postgres/dto/itinerary_version.go`
- *... and 58 more nodes in this community*

## Relationships

- No strong cross-community connections detected

## Source Files

- `apps/web/src/lib/api/auth.ts`
- `services/auth-service/internal/application/dto/auth.go`
- `services/auth-service/internal/application/errs/errs.go`
- `services/auth-service/internal/domain/errs/errs.go`
- `services/auth-service/internal/http-server/dto/request/auth.go`
- `services/auth-service/internal/http-server/dto/response/auth.go`
- `services/auth-service/internal/http-server/handler/auth.go`
- `services/auth-service/internal/infrastructure/repository/postgres/auth.go`
- `services/auth-service/internal/infrastructure/repository/postgres/dto/auth.go`
- `services/auth-service/pkg/storage/postgres/postgres.go`
- `services/external-integrations-service/internal/http-server/handler/dto.go`
- `services/trip-service/internal/application/errs/errs.go`
- `services/trip-service/internal/domain/entity/itinerary_version.go`
- `services/trip-service/internal/infrastructure/repository/postgres/dto/itinerary_version.go`
- `services/trip-service/internal/usercontext/dto.go`
- `services/user-service/internal/domain/entity/preferences.go`
- `services/user-service/internal/infrastructure/repository/postgres/dto/preferences.go`
- `services/user-service/internal/infrastructure/repository/postgres/repository.go`

## Audit Trail

- EXTRACTED: 245 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*