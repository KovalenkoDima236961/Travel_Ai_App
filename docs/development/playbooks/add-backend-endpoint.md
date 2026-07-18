# Playbook: add a backend endpoint

1. Choose the owning service from [service boundaries](../../architecture/service-boundaries.md); do not add a cross-service table shortcut.
2. Add request/response wire DTOs beside the service HTTP DTOs and keep private/public DTOs separate.
3. Put business logic in the application/domain service; keep handlers responsible for parsing, validation, auth context, status mapping, and JSON.
4. Register the route in the service router. Use bearer middleware for private routes and internal-token middleware only for `/internal/*` callers.
5. Enforce trip/workspace/resource permission in the service, not only the route or UI. Validate IDs, limits, body fields, and optimistic revision when the change edits an itinerary.
6. Add repository/sqlc/squirrel query code only in the owning service. Add a migration if persistence changes.
7. Propagate request/correlation IDs, use structured/sanitized logging, and add bounded metrics. Never log credentials, tokens, raw OCR, or full prompts.
8. Test success, validation, unauthenticated, forbidden, not-found, and conflict paths with `httptest` plus focused domain/repository tests.
9. Update [endpoint inventory](../../api/endpoint-inventory.md), error documentation, service README, and frontend client/hook if browser-facing.

Run `make fmt && make vet && make test` in the service and relevant root tests.
