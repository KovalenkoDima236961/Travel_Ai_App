# Travel Assistant Copilot v1

Copilot is a current-trip help panel, not an autonomous agent. It helps an
authenticated user understand existing readiness signals and navigate to the
screen where a human can review or make a change.

## What it can do

- Explain Trip Health, Budget Confidence, Group Readiness, route readiness,
  checklist/reminder state, expense totals, approval status, and policy warnings.
- Answer “where do I find…” and feature-help questions.
- Suggest existing deep links from the allowlisted action catalog.
- Explain uncertainty and partial context failures.
- In Travel Day Mode, explain a compact current-day summary and offer a deep
  link back to the authenticated travel-day screen.

## What it cannot do

Copilot cannot edit itineraries or budgets, apply repair, restore a version,
delete a trip, manage sharing/collaborators, book/pay, send nudges, upload files,
or run any other mutation. It never claims it performed one of those actions.

## Privacy and permissions

The Trip Service authenticates and resolves trip access before building context.
Only owner, accepted editor/viewer, or permitted workspace access can use the
private endpoint. Pending/removed collaborators and public-share users are
denied in v1.

Safe context is rebuilt from current summaries. It excludes tokens, API keys,
share passwords/tokens, raw receipt OCR, private expense notes, raw calendar
events/free-busy details, comments, provider secrets, file paths, and raw AI
prompts. The browser sends only a message and optional UI focus hint; it never
supplies trusted trip facts.

Available deep links are filtered against the caller’s role. A viewer receives
view links and a permission note; edit-only links are not returned. AI output is
validated server-side against that same allowlist before it reaches the browser.

## AI safety

The AI Planning Service prompt limits answers to the provided safe context,
treats chat text as untrusted, rejects prompt-injection requests, and forbids
action execution or claims of booking/payment/legal/safety guarantees. Mock mode
is deterministic; local Ollama mode uses strict JSON and falls back to mock when
configured. No hosted model is required.

## Observability and limitations

Copilot creates privacy-safe `copilot_response` traces with intent, message hash,
context sections, mode, timing, and validation result. It does not persist raw
chat content by default. Metrics use only low-cardinality labels.

Copilot is advisory and summary-based. Users must independently verify
schedules, prices, bookings, policies, local safety conditions, and official
requirements.

## Adding a source or action safely

1. Add a sanitized summary to `internal/copilot/context_builder.go`; do not pass
   raw domain payloads through.
2. Add a source type and its approved tab target in `response_validator.go`.
3. Add navigation-only or review-only actions to `action_catalog.go` with the
   concrete required permission. Never add an executable mutation.
4. Update the AI schema and mock responder, then add permission and response
   validation tests.
