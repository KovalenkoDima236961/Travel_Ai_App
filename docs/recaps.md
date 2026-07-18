# Post-Trip Recap & Learning Loop v1

Trip Recap is a private, editable record for a completed trip. Trip Service owns
the recap and feedback rows; it can call the existing AI Planning Service for a
strict `trip_recap_v1` JSON proposal and falls back to deterministic generation
when configured. It does not add a service, public share surface, or automatic
profile update.

## Access and lifecycle

Owners, accepted editors, accepted viewers, and permitted workspace members can
read a recap if they can already view its trip. Owners and editors can generate,
edit, finalize, archive, and create a template. Learning is per user: only an
explicitly approved candidate is mapped into the existing personalization
feedback flow. Viewers cannot modify the shared recap.

The active lifecycle is `draft`, `generated`, `edited`, `finalized`, then
`archived`. Generation is normally available after the trip end date; an editor
may explicitly confirm early generation through `generateEarly`.

## Routes

- `GET /trips/{id}/recap/status`
- `GET|PATCH|DELETE /trips/{id}/recap`
- `POST /trips/{id}/recap/generate`
- `POST /trips/{id}/recap/finalize`
- `POST /trips/{id}/recap/feedback`
- `POST /trips/{id}/recap/apply-learning`
- `POST /trips/{id}/recap/create-template`

The web route is `/trips/{id}/recap`, linked from Trip detail, Command Center,
post-trip Travel Day Mode, and the global command palette.

## Privacy boundary

The AI request contains bounded, sanitized aggregate data only: itinerary
planned/done/skipped/delayed outcomes, tracked expense totals and category
totals, receipt coverage counts, route/transport summaries, verification issue
counts, and checklist/reminder completion counts. It never contains receipt
files, raw OCR, receipt or expense notes, comments, calendar event data, share
credentials, access tokens, provider secrets, raw prompts, or user feedback
metadata. The AI response is schema-validated and sensitive text is rejected
before persistence.

Templates use the existing sanitization path and preserve only safe itinerary
structure plus selected lesson snippets. Recaps remain excluded from public
sharing and public exports. Copilot receives only a short safe recap summary;
editable notes, feedback metadata, and source artifacts are excluded.

## Configuration and operations

Trip Service: `TRIP_RECAP_ENABLED`, `TRIP_RECAP_AI_ENABLED`,
`TRIP_RECAP_FAIL_OPEN_WITH_DETERMINISTIC`, `TRIP_RECAP_TIMEOUT_SECONDS`, and
`TRIP_RECAP_MAX_SOURCE_CHARS`.

AI Planning Service: `TRIP_RECAP_ENABLED`, `TRIP_RECAP_AI_MODE=mock|ollama`,
`TRIP_RECAP_TIMEOUT_SECONDS`, and `TRIP_RECAP_FALLBACK_ENABLED`.

Apply migration `000036_create_trip_recaps` before enabling production traffic.
Metrics use the `trip_recap_*` prefix and avoid trip/user identifiers and free
text labels.

Run `RECAP_SMOKE_ACCESS_TOKEN=... RECAP_SMOKE_TRIP_ID=... ./scripts/smoke-recap.sh`
against a disposable owner/editor trip to verify generation, edit, explicit
learning, sanitized template creation, and finalization. Set
`RECAP_SMOKE_ARCHIVE=true` only when the disposable recap should be archived.
