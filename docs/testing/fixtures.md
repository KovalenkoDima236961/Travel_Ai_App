# Deterministic test fixtures

## Canonical identities

Frontend fixtures use reserved UUIDs and `.test` email addresses:

| Persona | ID suffix | Email | Product role |
| --- | --- | --- | --- |
| Owner | `...0001` | `owner@example.test` | Personal trip owner/workspace owner |
| Editor | `...0002` | `editor@example.test` | Accepted trip editor/workspace admin |
| Viewer | `...0003` | `viewer@example.test` | Accepted read-only collaborator/workspace member |
| Outsider | `...0004` | `outsider@example.test` | No trip access/workspace viewer fixture |

Backends should create equivalent records through package-local helpers instead of importing a giant cross-module fixture package. Each service owns its database schema and can evolve its setup independently. Use fixed UUIDs inside serial tests or a deterministic test-ID prefix when tests run in parallel.

The standard E2E password is `TestPassword1`; it is test-only. E2E email addresses include a run/worker/retry suffix so retries do not collide. `registerOrLogin` makes setup idempotent inside a retained test stack.

## Canonical trip data

- Destination: Vienna, Austria.
- Start date: `2027-04-10` for browser flows; unit fixtures may use `2026-04-10` because no current-date validation applies.
- Duration: 2 days; travelers: 2; pace: balanced.
- Budget: EUR 600.
- Interests: food and culture.
- Itinerary revision: 3.
- Day 1: Ringstrasse and Naschmarkt; Day 2: Belvedere.
- Route origin: Bratislava; transport: train; fixed distance/duration/cost.
- Expenses: EUR 24 lunch with deterministic receipt metadata.
- Public shares are enabled without password unless the test explicitly owns password/expiry behavior.

Frontend TypeScript sources are under `apps/web/test/fixtures`: users/workspace roles, trips, itinerary, budget/expenses, route, and notifications. Keep these objects aligned with backend DTO fields and use `tests/src/contracts/critical-dtos.test.ts` when a critical response changes.

## Checklist, reminders, and notifications

Package-local backend fixtures should use:

- checklist item `Pack travel documents`, due before departure, initially open;
- reminder `Check train platform`, due at a fixed UTC timestamp, initially pending;
- unread itinerary-updated notification and read itinerary-generated notification;
- preferences that allow in-app trip updates while email and push remain disabled.

Never place real emails, phone numbers, receipt OCR, calendar event text, API keys, or production identifiers in a fixture.

## File fixtures

`apps/web/e2e/fixtures/receipt.pdf` is a tiny synthetic receipt PDF. `unsupported-receipt.txt` is intentionally rejected. Existing safe PWA PNGs under `apps/web/public/icons` may be used when a valid image upload is required; tests must copy/upload the smallest icon rather than a personal image. File tests should assert detected MIME, extension consistency, size limits, authorization, and sanitized metadata—not OCR prose.

## Provider fixtures

Provider fixtures use the built-in mock providers. Expected results must remain deterministic: fixed coordinates, distances, costs, currencies, weather dates, availability options, and fallback flags. Real provider adapters use local HTTP servers with controlled JSON and timeouts. Do not record or replay real customer/provider traffic.

## Fixture maintenance rules

- Keep the smallest object that still represents the contract.
- Use explicit ISO-8601 UTC timestamps and currency codes.
- Clone mutable fixtures before changing them in a test.
- Add fields when they are regression-sensitive; do not mirror every optional DTO field.
- Update frontend fixtures, backend response tests, docs, and MSW handlers together when a critical DTO changes.
