# Closed alpha scope

This document freezes `closed-alpha-v1`. It is a controlled invited-user release, not production readiness and not a beta launch.

## Purpose

Validate the core AI trip-planning journey with real invited users while keeping recovery, privacy, cost, and rollback controls stronger than the product surface.

## Audience

- Invited users who understand this is an alpha.
- Allowlisted operators listed in `OPS_ADMIN_EMAILS`.
- Engineers and support reviewers using the alpha runbooks.

Do not invite general public traffic, paid customers, or users whose trips require booking/payment guarantees.

## Included features

- Register, login, logout, session refresh.
- Profile and travel preferences.
- Create/list/read trips.
- OpenAI primary itinerary generation through the existing provider-neutral AI path.
- Deterministic fallback when configured.
- AI validation, one bounded generation-output repair attempt, and quality warnings.
- Generation jobs, polling, retry/cancel/mark-failed Ops recovery.
- Itinerary review, editing, revision conflict prevention, and version history.
- Route basics and distance/transport summaries from already stable paths.
- Budget basics and budget summary.
- Place review, grounding confidence, opening-hours and quality warnings.
- In-app/email notifications and digest controls.
- Public read-only sharing with expiry/password support.
- Data exports only while `data_exports_enabled` remains in the alpha profile and smoke tests pass.
- Settings, data/privacy controls, alpha label, feedback link.
- Ops Dashboard jobs, queues, DLQ, provider health, quotas, feature flags, AI traces, and Alpha Overview.

## Disabled by default

These are disabled by `config/feature-flags/alpha.json` and checked by `scripts/release/validate-alpha-scope.sh`:

- Fine-tuning experiment controls and adapter inference/staging.
- Shadow model evaluation for normal users.
- Candidate model internal, allowlist, percentage, and user opt-in rollout.
- Real travel providers unless a blocker is explicitly reviewed.
- Calendar sync, availability search, transport search, and booking-style workflows.
- Receipt OCR.
- Workspace approvals and policy repair.
- Web push.
- Offline mutation mode.
- Template adaptation and route-alternative generation.

Disabling must be enforced by backend feature flags, not navigation hiding only.

## Known limitations

- Alpha runtime uses `APP_ENV=staging`; there is no new service enum named `alpha`.
- Real OpenAI calls are allowed only with explicit approval, strict caps, and no normal CI usage.
- Cost telemetry depends on OpenAI usage tracking plus the spend-limit checker; Prometheus spend-ratio alerts require an exported spend ratio metric.
- Public sharing is read-only and must remain sanitized.
- Exports are private and authenticated; receipt file backup/restore is documented but not automatically restored by alpha rollback scripts.
- Browser release candidate coverage is Chromium by default; WebKit/Firefox are release-candidate evidence when practical.
- No booking, payment, calendar production sync, or live provider availability guarantee.

## Unsupported workflows

- Production fine-tuning or candidate rollout.
- Manual database edits as normal recovery.
- Manual down migrations during rollback.
- Real provider booking, ticket purchase, payment, or travel-safety guarantees.
- Public-share write actions or private collaboration data on public routes.
- Raw prompt/response logging, raw itinerary telemetry, raw OCR telemetry, or prompt artifacts in CI.

## Data and privacy expectations

- Alpha telemetry uses low-cardinality aggregates only.
- Do not store or upload raw trip titles, exact destinations in metrics, comments, emails, names, receipts, OCR text, private notes, tokens, raw prompts, or raw AI responses.
- AI trace pages expose safe summaries and redacted snapshots only when observability is enabled.
- Feedback stores structured chips with safe metadata: generation job ID as entity ID, trip/workspace IDs in private authenticated storage, and low-cardinality metadata keys only.
- Public share responses must exclude private budgets, expenses, comments, collaborators, activity, provider internals, AI traces, and tokens.

## Support expectations

- User-visible failures must show a safe error code and request/support reference where the route supports it.
- Operators use the Alpha Overview, Feature Flags, Jobs, DLQ, AI Generations, and provider quota panels before SQL access.
- Incident response follows the alpha runbooks and `docs/security/incident-response.md`.
- Every accepted risk must be recorded in `docs/releases/alpha-launch-decision.md`.

## Validation commands

```bash
cp infra/.env.alpha.example infra/.env.alpha
# replace every change_me value with managed secrets before validating
./scripts/validate-env.sh alpha --env-file infra/.env.alpha
./scripts/release/validate-alpha-scope.sh --env-file infra/.env.alpha
REGISTRY=travel-ai ./scripts/release/build-images.sh
./scripts/release/rehearse-alpha-release.sh --env-file infra/.env.alpha --mock-openai
./scripts/release/rehearse-alpha-rollback.sh --env-file infra/.env.alpha --previous-image-tag <prior-tag>
```

## Beta exit criteria

- Two consecutive release candidates pass alpha readiness CI, alpha smoke, rollback rehearsal, backup verification, security scans, and the Playwright alpha suite.
- No unresolved data corruption, critical/high security issue, or unbounded OpenAI cost risk.
- Generation success, fallback, repair, queue delay, and public-share sanitization stay within documented telemetry targets.
- Support has a reviewed error catalog, incident owner, rollback owner, and known issues document.
- Candidate model rollout and real providers remain separately reviewed before beta.
