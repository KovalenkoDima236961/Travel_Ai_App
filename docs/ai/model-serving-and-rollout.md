# AI Model Serving and Rollout

V1 keeps the grounded baseline as the production default. Candidate adapters can
be registered, shadowed, and exposed to internal or allowlisted users only after
explicit backend configuration. No candidate is promoted automatically.

## Model Definitions

The baseline model variant is `grounded_baseline`: the current AI Planning
Service path with Trip Service grounding, validation, repair, and provider
checks. It must remain loadable while any candidate is evaluated.

The candidate model variant is `fine_tuned_candidate`: a checksum-verified
LoRA/QLoRA adapter tied to an experiment and dataset version. Candidate
adapters are disabled by default and cannot be selected by public clients.

## Deployment States

Supported states are:

- `registered`: deployment record exists but receives no traffic.
- `candidate`: candidate is available for configuration, not traffic.
- `shadow`: eligible for sampled asynchronous comparison only.
- `internal`: user-visible only for ops/internal users.
- `allowlist`: user-visible only for allowlisted users or workspaces.
- `staged_rollout`: deterministic opted-in percentage rollout.
- `active`: default active deployment for an environment/task.
- `paused`: receives no new assignments.
- `rejected`: cannot serve.
- `retired`: cannot serve.

Only one `active`/`active` deployment can exist per environment and task type.
Multiple shadow candidates may be registered, but V1 executes at most one
candidate per request.

## Shadow Inference Flow

Trip Service builds sanitized planning constraints and grounding once, resolves
the routing decision, executes the baseline as the user-visible primary, saves
the primary result normally, and then enqueues `ai_shadow_generation_evaluation`
when the request is sampled and capacity allows it.

The shadow job contains safe immutable references only: assignment ID, trip ID,
generation job ID, deployment IDs, snapshot references, prompt version,
validator version, and expiry. Shadow output never creates itinerary versions,
changes trip state, triggers user notifications, or appears in public shares.

## Deterministic Assignment

Assignments use a stable bucket `0..9999` derived from:

`assignmentSalt + userId`, then workspace ID, then trip ID, then request key.

Changing the salt is an explicit rollout reset and must be audited. Frontend
controls may expose opt-in UI, but model assignment is computed server-side.

## Rollout Modes

Internal rollout is for ops users and staging. Allowlist rollout supports user
and workspace IDs. Percentage rollout is only for eligible users who opted in to
experimental AI and are bucket-selected. Opt-in is separate from training-data
consent.

User-visible labels are `Standard AI` and `Experimental AI`. Technical model
paths, adapter paths, and private experiment details are ops-only.

## Guardrails and Rollback

Guardrails evaluate quality, parse/failure rates, repair rate, latency,
grounded-place rate, hallucination/destination mismatch regressions, and
language regressions over bounded windows. Critical guardrail failures in
user-visible candidate modes pause candidate traffic, set rollout to zero,
route new requests to the grounded baseline, record `guardrail_paused`, and
notify ops.

Rollback is configuration/database state, not an application redeploy. Existing
itineraries are not mutated.

## Privacy Rules

Do not store raw prompts, complete generated itineraries, user notes, comments,
receipts, calendar data, profile identity fields, private addresses, provider
secrets, adapter paths, or high-cardinality IDs in analytics labels. Comparison
rows store normalized metrics and safe references only.

Encrypted shadow input snapshots, when needed, are private, short-lived, not
API-exposed, and cleaned up after expiry. Production candidate raw output
retention is disabled by default.

## Capacity Limits

Default capacity controls:

- `AI_SHADOW_ENABLED=false`
- `AI_SHADOW_SAMPLE_PERCENT=0`
- `AI_SHADOW_MAX_CONCURRENT=1`
- `AI_SHADOW_QUEUE_NAME=ai.shadow.evaluations`
- `AI_SHADOW_TIMEOUT_SECONDS=180`
- `AI_SHADOW_MAX_QUEUE_AGE_SECONDS=900`
- `AI_SHADOW_SKIP_WHEN_QUEUE_DEPTH_ABOVE=100`
- `AI_SHADOW_FAIL_OPEN=true`

Primary requests take priority. Shadow is skipped under load, when the
candidate is unavailable, when snapshots are invalid/expired, or when a request
is unsupported or sensitive.

## Promotion Workflow

1. Register baseline and candidate deployments.
2. Enable a small shadow sample in local/staging.
3. Review online comparison metrics, guardrails, and skipped-shadow counts.
4. Enable internal or allowlist rollout after human approval.
5. Increase percentage only with opt-in, guardrails, and rollback target ready.
6. Make a manual promotion decision. Training loss alone is never sufficient.

## Known Limitations

V1 uses the existing AI Planning Service and Worker flow. It does not introduce
Kubernetes, a service mesh, or an external model-serving platform. Trip Service
can register deployments, route from database deployment state with a static
baseline fallback, and persist request assignments. Full asynchronous shadow
execution still requires queue consumer wiring and adapter load approval before
candidate comparisons appear in online metrics.
