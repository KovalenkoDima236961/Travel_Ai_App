# Web API Client

> 24 nodes · cohesion 0.12

## Key Concepts

- **client.go** (21 connections) — `services/trip-service/pkg/cache/redis/client.go`
- **Client** (5 connections) — `services/trip-service/internal/usercontext/client.go`
- **.get()** (5 connections) — `services/trip-service/internal/usercontext/client.go`
- **apiFetchInternal()** (4 connections) — `apps/web/src/lib/api/client.ts`
- **RedisClient** (4 connections) — `services/trip-service/pkg/cache/redis/redis.go`
- **.GetUserContext()** (4 connections) — `services/trip-service/internal/usercontext/client.go`
- **.GetMyPreferences()** (3 connections) — `services/trip-service/internal/usercontext/client.go`
- **.GetMyProfile()** (3 connections) — `services/trip-service/internal/usercontext/client.go`
- **ApiError** (2 connections) — `apps/web/src/lib/api/client.ts`
- **apiFetch()** (2 connections) — `apps/web/src/lib/api/client.ts`
- **buildApiUrl()** (2 connections) — `apps/web/src/lib/api/client.ts`
- **isMissing()** (2 connections) — `services/trip-service/internal/usercontext/client.go`
- **NewClient()** (2 connections) — `services/trip-service/pkg/cache/redis/client.go`
- **normalizeBaseURL()** (2 connections) — `services/trip-service/internal/usercontext/client.go`
- **notifySessionExpired()** (2 connections) — `apps/web/src/lib/api/client.ts`
- **readErrorBody()** (2 connections) — `services/trip-service/internal/usercontext/client.go`
- **Error** (2 connections) — `services/trip-service/internal/usercontext/client.go`
- **.Error()** (2 connections) — `services/trip-service/internal/usercontext/client.go`
- **.constructor()** (1 connections) — `apps/web/src/lib/api/client.ts`
- **readJson()** (1 connections) — `apps/web/src/lib/api/client.ts`
- **.Close()** (1 connections) — `services/trip-service/pkg/cache/redis/redis.go`
- **.HealthCheck()** (1 connections) — `services/trip-service/pkg/cache/redis/redis.go`
- **.Unwrap()** (1 connections) — `services/trip-service/pkg/cache/redis/redis.go`
- **ErrorType** (1 connections) — `services/trip-service/internal/usercontext/client.go`

## Relationships

- No strong cross-community connections detected

## Source Files

- `apps/web/src/lib/api/client.ts`
- `services/trip-service/internal/usercontext/client.go`
- `services/trip-service/pkg/cache/redis/client.go`
- `services/trip-service/pkg/cache/redis/redis.go`

## Audit Trail

- EXTRACTED: 75 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*