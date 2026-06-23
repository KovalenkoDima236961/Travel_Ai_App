# Service Configuration

> 51 nodes · cohesion 0.05

## Key Concepts

- **config.go** (43 connections) — `services/auth-service/pkg/storage/postgres/config.go`
- **server.go** (8 connections) — `services/auth-service/internal/http-server/server.go`
- **Config** (7 connections) — `services/auth-service/internal/config/config.go`
- **get_settings()** (6 connections) — `services/ai-planning-service/app/config.py`
- **cors_test.go** (6 connections) — `services/auth-service/internal/http-server/cors_test.go`
- **route.ts** (6 connections) — `apps/web/src/app/api/trip-service/[...path]/route.ts`
- **Load()** (4 connections) — `services/auth-service/internal/config/config.go`
- **cors.go** (4 connections) — `services/auth-service/internal/http-server/cors.go`
- **proxyTripServiceRequest()** (4 connections) — `apps/web/src/app/api/trip-service/[...path]/route.ts`
- **.IsProduction()** (3 connections) — `services/auth-service/internal/config/config.go`
- **.validateJWTSecret()** (3 connections) — `services/auth-service/internal/config/config.go`
- **getTripServiceInternalUrl()** (3 connections) — `apps/web/src/lib/config.ts`
- **Server** (3 connections) — `services/auth-service/internal/http-server/server.go`
- **copyHeader()** (3 connections) — `apps/web/src/app/api/trip-service/[...path]/route.ts`
- **GET()** (3 connections) — `apps/web/src/app/api/trip-service/[...path]/route.ts`
- **.applyDefaults()** (2 connections) — `services/auth-service/internal/config/config.go`
- **.UsesDefaultDevelopmentJWTSecret()** (2 connections) — `services/user-service/internal/config/config.go`
- **_env_bool()** (2 connections) — `services/ai-planning-service/app/config.py`
- **_env_float()** (2 connections) — `services/ai-planning-service/app/config.py`
- **_env_int()** (2 connections) — `services/ai-planning-service/app/config.py`
- **_env_string()** (2 connections) — `services/ai-planning-service/app/config.py`
- **getTripApiBaseUrl()** (2 connections) — `apps/web/src/lib/config.ts`
- **getTripServiceUrl()** (2 connections) — `apps/web/src/lib/config.ts`
- **MustLoad()** (2 connections) — `services/auth-service/internal/config/config.go`
- **corsMiddleware()** (2 connections) — `services/auth-service/internal/http-server/cors.go`
- *... and 26 more nodes in this community*

## Relationships

- No strong cross-community connections detected

## Source Files

- `apps/web/src/app/api/trip-service/[...path]/route.ts`
- `apps/web/src/lib/config.ts`
- `services/ai-planning-service/app/config.py`
- `services/auth-service/internal/config/config.go`
- `services/auth-service/internal/http-server/cors.go`
- `services/auth-service/internal/http-server/cors_test.go`
- `services/auth-service/internal/http-server/server.go`
- `services/auth-service/pkg/storage/postgres/config.go`
- `services/external-integrations-service/internal/config/config.go`
- `services/trip-service/internal/config/config.go`
- `services/trip-service/internal/http-server/cors_test.go`
- `services/trip-service/pkg/cache/redis/config.go`
- `services/user-service/internal/config/config.go`

## Audit Trail

- EXTRACTED: 154 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*