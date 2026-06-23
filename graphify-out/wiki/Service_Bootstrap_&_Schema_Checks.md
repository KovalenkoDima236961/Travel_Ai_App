# Service Bootstrap & Schema Checks

> 42 nodes · cohesion 0.06

## Key Concepts

- **errors.py** (17 connections) — `services/ai-planning-service/app/core/errors.py`
- **closer.go** (9 connections) — `services/auth-service/pkg/closer/closer.go`
- **readiness_test.go** (9 connections) — `services/trip-service/internal/http-server/readiness_test.go`
- **app.go** (8 connections) — `services/auth-service/internal/app/app.go`
- **checks.go** (7 connections) — `services/auth-service/pkg/storage/postgres/checks.go`
- **validator.go** (6 connections) — `services/user-service/pkg/validation/validator.go`
- **decodeReadinessBody()** (4 connections) — `services/trip-service/internal/http-server/readiness_test.go`
- **closer** (3 connections) — `services/trip-service/pkg/closer/closer.go`
- **main.go** (3 connections) — `services/auth-service/cmd/server/main.go`
- **ValidationError** (3 connections) — `services/user-service/pkg/validation/validator.go`
- **App** (2 connections) — `services/auth-service/internal/app/app.go`
- **New()** (2 connections) — `services/auth-service/internal/app/app.go`
- **warnWeakDevelopmentSecret()** (2 connections) — `services/auth-service/internal/app/app.go`
- **Add()** (2 connections) — `services/auth-service/pkg/closer/closer.go`
- **CloseAll()** (2 connections) — `services/auth-service/pkg/closer/closer.go`
- **.add()** (2 connections) — `services/trip-service/pkg/closer/closer.go`
- **.closeAll()** (2 connections) — `services/trip-service/pkg/closer/closer.go`
- **fakeReadinessDB** (2 connections) — `services/trip-service/internal/http-server/readiness_test.go`
- **TestReadinessHandlerChecksAIPlanningServiceInHTTPMode()** (2 connections) — `services/trip-service/internal/http-server/readiness_test.go`
- **TestReadinessHandlerReadyInMockMode()** (2 connections) — `services/trip-service/internal/http-server/readiness_test.go`
- **TestReadinessHandlerReturnsUnavailableOnPostgresFailure()** (2 connections) — `services/trip-service/internal/http-server/readiness_test.go`
- **Validation** (2 connections) — `services/user-service/pkg/validation/validator.go`
- **.Validate()** (2 connections) — `services/user-service/pkg/validation/validator.go`
- **fieldErrorMessage()** (2 connections) — `services/user-service/pkg/validation/validator.go`
- **.Run()** (1 connections) — `services/auth-service/internal/app/app.go`
- *... and 17 more nodes in this community*

## Relationships

- No strong cross-community connections detected

## Source Files

- `services/ai-planning-service/app/core/errors.py`
- `services/ai-planning-service/app/main.py`
- `services/auth-service/cmd/server/main.go`
- `services/auth-service/internal/app/app.go`
- `services/auth-service/pkg/closer/closer.go`
- `services/auth-service/pkg/storage/postgres/checks.go`
- `services/trip-service/internal/http-server/readiness_test.go`
- `services/trip-service/pkg/closer/closer.go`
- `services/trip-service/pkg/storage/postgres/checks.go`
- `services/user-service/pkg/validation/validator.go`

## Audit Trail

- EXTRACTED: 115 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*