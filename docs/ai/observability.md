# AI generation observability

AI generation traces make background and synchronous planning failures diagnosable without retaining raw prompts or private trip data. Trip Service stores traces in Postgres; the existing Ops allowlist protects the read API and dashboard.

## Stored data

Each trace records generation type, job/trip correlation, provider/mode/model metadata, prompt and validator versions, timing, queue wait, final status, quality status, and safe summaries for planning constraints, RAG, validation, repair, and saved output. Timeline events record the high-level stages only.

Summaries contain counts, boolean flags, category/severity totals, and issue IDs. They do not retain itinerary text, request instructions, user email, phone, receipt OCR, expense notes, calendar details, comments, tokens, secrets, API keys, share credentials, or RAG chunk text.

## Prompt privacy

Raw prompts are neither persisted nor logged by default. `AI_OBSERVABILITY_STORE_REDACTED_PROMPTS=false` is the production default. When a local diagnostic environment enables it, the privacy guard redacts sensitive patterns, truncates content to `AI_OBSERVABILITY_MAX_PROMPT_SNAPSHOT_CHARS`, stores a hash, and exposes it only to allowlisted ops users with an audit event. Production configuration rejects unsafe prompt logging and snapshot settings.

AI Planning Service returns safe response metadata (`promptVersion`, provider/model/mode, duration, and token estimate) but never returns an internal prompt to normal clients.

## Ops usage and retention

Allowlisted admins use `/ops/ai-generations` to filter traces and `/ops/ai-generations/{traceId}` to inspect the timeline and safe summaries. Normal users and public-share clients have no trace route.

`AI_OBSERVABILITY_RETENTION_DAYS` defaults to 30. Trip Service runs cleanup daily; deleting a trace cascades to events and optional redacted snapshots. Trace persistence is fail-open by default (`AI_OBSERVABILITY_FAIL_OPEN=true`) so an observability database failure cannot fail a generation job.

## Limitations

V1 is a debugging history, not a prompt experiment, billing, benchmark, or distributed tracing platform. AI-call timing is measured around the worker generation operation; detailed provider token accounting is only available when a provider reports it.
