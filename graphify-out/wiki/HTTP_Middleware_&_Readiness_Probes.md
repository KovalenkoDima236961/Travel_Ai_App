# HTTP Middleware & Readiness Probes

> 20 nodes · cohesion 0.15

## Key Concepts

- **routes.go** (19 connections) — `services/auth-service/internal/http-server/routes.go`
- **middleware.go** (6 connections) — `services/user-service/internal/auth/middleware.go`
- **_parse_partial_request()** (4 connections) — `services/ai-planning-service/app/api/routes.py`
- **Middleware()** (3 connections) — `services/user-service/internal/auth/middleware.go`
- **_check_chroma()** (3 connections) — `services/ai-planning-service/app/api/routes.py`
- **ready()** (3 connections) — `services/ai-planning-service/app/api/routes.py`
- **bearerToken()** (2 connections) — `services/user-service/internal/auth/middleware.go`
- **writeUnauthorized()** (2 connections) — `services/user-service/internal/auth/middleware.go`
- **_check_ollama()** (2 connections) — `services/ai-planning-service/app/api/routes.py`
- **NewRouter()** (2 connections) — `services/auth-service/internal/http-server/routes.go`
- **regenerate_day()** (2 connections) — `services/ai-planning-service/app/api/routes.py`
- **regenerate_item()** (2 connections) — `services/ai-planning-service/app/api/routes.py`
- **requestLogger()** (2 connections) — `services/auth-service/internal/http-server/routes.go`
- **_resolve_service_path()** (2 connections) — `services/ai-planning-service/app/api/routes.py`
- **_validation_error_message()** (2 connections) — `services/ai-planning-service/app/api/routes.py`
- **MiddlewareConfig** (1 connections) — `services/user-service/internal/auth/middleware.go`
- **generate_itinerary()** (1 connections) — `services/ai-planning-service/app/api/routes.py`
- **get_configured_itinerary_generator()** (1 connections) — `services/ai-planning-service/app/api/routes.py`
- **health()** (1 connections) — `services/ai-planning-service/app/api/routes.py`
- **healthHandler()** (1 connections) — `services/auth-service/internal/http-server/routes.go`

## Relationships

- No strong cross-community connections detected

## Source Files

- `services/ai-planning-service/app/api/routes.py`
- `services/auth-service/internal/http-server/routes.go`
- `services/user-service/internal/auth/middleware.go`

## Audit Trail

- EXTRACTED: 61 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*