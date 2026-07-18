# API contracts

## Why contracts exist

API contracts make the browser-facing boundary explicit. They give backend
owners a reviewable description of a change and let the Web App compile against
the same request and response types that are published by the services. They
do not replace handler tests: handlers and their DTOs remain the executable
implementation.

## Source of truth

Each service owns its OpenAPI document under `docs/api/openapi/`:

- Go service documents are maintained with the public handler and DTO changes.
- The AI Planning document is exported from FastAPI/Pydantic by
  `scripts/contracts/export-ai-openapi.sh`.
- `apps/web/src/lib/api/generated/` is derived output. It must never be edited
  manually.

This is deliberately one-way: backend-owned schemas produce frontend types.
Existing raw response bodies are documented as they are; the v1 envelope is a
forward convention for new or intentionally migrated endpoints, not a
big-bang breaking change.

## Generated and hand-written boundaries

`apps/web/src/lib/api/generated/*/schema.ts` contains only generated OpenAPI
types. Hand-written modules in `apps/web/src/lib/api/` own transport details
(base URL, bearer refresh, multipart/download handling and UI adapters). The
small `contracts.ts` adapter exports stable aliases for generated schemas. UI
form values, query keys, component props and derived view models remain local.

## Maintenance workflow

1. Change the public handler and its DTO/test.
2. Update that service's OpenAPI document (or export the FastAPI document).
3. Run `./scripts/contracts/generate-web-client.sh`.
4. Update affected Web App wrapper, MSW fixture and focused handler test.
5. Record breaking changes in `contract-changelog.md` and update versioning
   notes when necessary.

`./scripts/contracts/validate-openapi.sh` lints every published document;
`./scripts/contracts/check-generated.sh` regenerates and fails on a diff.
The `api-contracts` CI job runs both before the frontend typecheck.

## Versioning and scope

This is contract version `v1`. URL paths are not versioned in v1. Additive
fields and endpoints are preferred. Removing/renaming a public field, changing
its meaning, tightening accepted input, or changing authorization is breaking
and requires a migration plan, a contract-changelog entry and frontend update.

The initial coverage is intentionally frontend-facing: core auth, profile,
trip/itinerary/jobs/budget/expenses/public sharing, notifications, provider
lookups and AI planning. The endpoint inventory records the wider surface; its
remaining domains are scheduled as incremental additions rather than being
guessed into generated browser clients.

## Public and private DTOs

`TripPrivateResponse`/`Trip` is an authenticated DTO. `PublicTripResponse` is
a separately named, read-only schema; it must never reuse the private response.
Public DTOs exclude identity/contact data, collaborators, comments, activity,
expenses, receipts, private budget data, internal metadata, share password
hashes, tokens and AI prompts/traces. Export and ops DTOs are likewise separate
from browser DTOs.

## Internal endpoints

`/internal/*` and ops-only routes are documented in
[internal contracts](internal-contracts.md), use `X-Internal-Service-Token`,
and are excluded from generated browser clients. A new internal route needs an
identified caller, bounded request/response schema, failure behavior and a
test that denies a missing token.

## v1 follow-up inventory

The first pass does not generate every trip domain: templates, approvals,
policies, archives/library, travel-day, exports, receipts OCR and command
center subresources continue to be described by the endpoint inventory until
their handlers are migrated one domain at a time. Their paths remain stable.
