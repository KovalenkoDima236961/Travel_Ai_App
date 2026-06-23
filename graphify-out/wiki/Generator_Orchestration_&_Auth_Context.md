# Generator Orchestration & Auth Context

> 97 nodes · cohesion 0.03

## Key Concepts

- **http.go** (37 connections) — `services/trip-service/internal/infrastructure/generator/http.go`
- **context.go** (35 connections) — `services/user-service/internal/auth/context.go`
- **zap.go** (28 connections) — `services/auth-service/pkg/logger/zap.go`
- **http_test.go** (16 connections) — `services/trip-service/internal/infrastructure/generator/http_test.go`
- **auth_test.go** (11 connections) — `services/auth-service/internal/http-server/handler/auth_test.go`
- **client_test.go** (11 connections) — `services/trip-service/internal/usercontext/client_test.go`
- **di.go** (11 connections) — `services/auth-service/internal/app/di.go`
- **provider_test.go** (11 connections) — `services/external-integrations-service/internal/infrastructure/provider/places/provider_test.go`
- **newTestHTTPGenerator()** (10 connections) — `services/trip-service/internal/infrastructure/generator/http_test.go`
- **newTestClient()** (8 connections) — `services/trip-service/internal/usercontext/client_test.go`
- **validTrip()** (8 connections) — `services/trip-service/internal/infrastructure/generator/http_test.go`
- **generator.go** (6 connections) — `services/trip-service/internal/application/generator.go`
- **stubAuthService** (6 connections) — `services/auth-service/internal/http-server/handler/auth_test.go`
- **readiness.go** (6 connections) — `services/auth-service/internal/http-server/readiness.go`
- **tls.go** (6 connections) — `services/trip-service/pkg/tls/tls.go`
- **newTestRouter()** (5 connections) — `services/auth-service/internal/http-server/handler/auth_test.go`
- **assertUserContextError()** (5 connections) — `services/trip-service/internal/usercontext/client_test.go`
- **AIPlanningHTTPGenerator** (5 connections) — `services/trip-service/internal/infrastructure/generator/http.go`
- **TestAIPlanningHTTPGeneratorRegenerateDay_SendsCorrectPayload()** (4 connections) — `services/trip-service/internal/infrastructure/generator/http_test.go`
- **TestAIPlanningHTTPGeneratorRegenerateItem_SendsCorrectPayload()** (4 connections) — `services/trip-service/internal/infrastructure/generator/http_test.go`
- **TestClientGetMyProfile_DoesNotAcceptMissingToken()** (3 connections) — `services/trip-service/internal/usercontext/client_test.go`
- **TestClientGetUserContext_AuthFailureIsTypedError()** (3 connections) — `services/trip-service/internal/usercontext/client_test.go`
- **TestClientGetUserContext_MalformedJSONIsTypedError()** (3 connections) — `services/trip-service/internal/usercontext/client_test.go`
- **TestClientGetUserContext_ServiceFailureIsTypedError()** (3 connections) — `services/trip-service/internal/usercontext/client_test.go`
- **UserFromContext()** (3 connections) — `services/user-service/internal/auth/context.go`
- *... and 72 more nodes in this community*

## Relationships

- No strong cross-community connections detected

## Source Files

- `services/auth-service/internal/app/di.go`
- `services/auth-service/internal/http-server/handler/auth_test.go`
- `services/auth-service/internal/http-server/readiness.go`
- `services/auth-service/pkg/logger/zap.go`
- `services/external-integrations-service/internal/infrastructure/provider/places/provider_test.go`
- `services/trip-service/internal/application/generator.go`
- `services/trip-service/internal/auth/context.go`
- `services/trip-service/internal/http-server/readiness.go`
- `services/trip-service/internal/infrastructure/generator/http.go`
- `services/trip-service/internal/infrastructure/generator/http_test.go`
- `services/trip-service/internal/infrastructure/generator/provider_test.go`
- `services/trip-service/internal/usercontext/client_test.go`
- `services/trip-service/internal/usercontext/module.go`
- `services/trip-service/pkg/cache/redis/redis.go`
- `services/trip-service/pkg/tls/tls.go`
- `services/user-service/internal/auth/context.go`

## Audit Trail

- EXTRACTED: 363 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*