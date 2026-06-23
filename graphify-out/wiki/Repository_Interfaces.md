# Repository Interfaces

> 26 nodes · cohesion 0.09

## Key Concepts

- **Repository** (27 connections) — `services/auth-service/internal/infrastructure/repository/postgres/auth.go`
- **.UpdateItineraryByUserIDAndCreateVersion()** (5 connections) — `services/trip-service/internal/infrastructure/repository/postgres/trip.go`
- **.nextItineraryVersionNumber()** (3 connections) — `services/trip-service/internal/infrastructure/repository/postgres/trip.go`
- **.CreateItineraryVersion()** (2 connections) — `services/trip-service/internal/infrastructure/repository/postgres/trip.go`
- **.GetNextItineraryVersionNumber()** (2 connections) — `services/trip-service/internal/infrastructure/repository/postgres/trip.go`
- **.UpdateItineraryByUserID()** (2 connections) — `services/trip-service/internal/infrastructure/repository/postgres/trip.go`
- **New()** (2 connections) — `services/trip-service/internal/infrastructure/repository/postgres/trip.go`
- **.Create()** (1 connections) — `services/trip-service/internal/infrastructure/repository/postgres/trip.go`
- **.CreateDefaultPreferences()** (1 connections) — `services/user-service/internal/infrastructure/repository/postgres/repository.go`
- **.CreateDefaultProfile()** (1 connections) — `services/user-service/internal/infrastructure/repository/postgres/repository.go`
- **.CreateRefreshToken()** (1 connections) — `services/auth-service/internal/infrastructure/repository/postgres/auth.go`
- **.CreateUser()** (1 connections) — `services/auth-service/internal/infrastructure/repository/postgres/auth.go`
- **.GetByIDAndUserID()** (1 connections) — `services/trip-service/internal/infrastructure/repository/postgres/trip.go`
- **.GetItineraryVersionByIDTripAndUser()** (1 connections) — `services/trip-service/internal/infrastructure/repository/postgres/trip.go`
- **.GetPreferencesByUserID()** (1 connections) — `services/user-service/internal/infrastructure/repository/postgres/repository.go`
- **.GetProfileByUserID()** (1 connections) — `services/user-service/internal/infrastructure/repository/postgres/repository.go`
- **.GetRefreshTokenByHash()** (1 connections) — `services/auth-service/internal/infrastructure/repository/postgres/auth.go`
- **.GetUserByEmail()** (1 connections) — `services/auth-service/internal/infrastructure/repository/postgres/auth.go`
- **.GetUserByID()** (1 connections) — `services/auth-service/internal/infrastructure/repository/postgres/auth.go`
- **.ListByUser()** (1 connections) — `services/trip-service/internal/infrastructure/repository/postgres/trip.go`
- **.ListItineraryVersionsByTripAndUser()** (1 connections) — `services/trip-service/internal/infrastructure/repository/postgres/trip.go`
- **.RevokeRefreshTokenByHash()** (1 connections) — `services/auth-service/internal/infrastructure/repository/postgres/auth.go`
- **.RotateRefreshToken()** (1 connections) — `services/auth-service/internal/infrastructure/repository/postgres/auth.go`
- **.UpdateStatusByUserID()** (1 connections) — `services/trip-service/internal/infrastructure/repository/postgres/trip.go`
- **.UpsertPreferences()** (1 connections) — `services/user-service/internal/infrastructure/repository/postgres/repository.go`
- *... and 1 more nodes in this community*

## Relationships

- No strong cross-community connections detected

## Source Files

- `services/auth-service/internal/infrastructure/repository/postgres/auth.go`
- `services/trip-service/internal/infrastructure/repository/postgres/trip.go`
- `services/user-service/internal/infrastructure/repository/postgres/repository.go`

## Audit Trail

- EXTRACTED: 62 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*