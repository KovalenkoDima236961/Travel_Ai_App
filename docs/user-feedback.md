# User Feedback

Closed-alpha feedback is collected through an authenticated global feedback
dialog and stored in Trip Service.

## User Submission

Users can submit:

- bug report
- feature request
- AI issue
- performance issue
- UI issue
- accessibility issue
- security issue
- other

The browser automatically includes app version, browser family, OS family,
device type, request/correlation identifiers when supplied, provider/model/prompt
metadata when supplied, and feature flags when supplied.

The feedback dialog tells users not to include private notes, prompts, receipts,
or secrets.

## Attachments

Feedback supports optional screenshot attachment metadata. The backend validates:

- maximum 3 attachments
- MIME type must be `image/png`, `image/jpeg`, or `image/webp`
- size must be greater than 0 and no more than 5 MB
- SHA-256 digest must be valid when supplied

The v1 handler records attachment metadata only; it does not persist binary
files. If binary upload storage is added later, malware scanning and retention
controls must be added before storing files.

## Ops Workflow

Ops routes:

| Method/path | Purpose |
| --- | --- |
| `GET /ops/alpha/feedback` | List feedback, optionally by status/category |
| `GET /ops/alpha/feedback/{feedbackId}` | Read feedback detail and attachment metadata |
| `PATCH /ops/alpha/feedback/{feedbackId}` | Update status, priority, owner, or internal notes |

Statuses are `open`, `triaged`, `in_progress`, `resolved`, `closed`, and
`duplicate`. Priorities are `low`, `normal`, `high`, and `urgent`.

Internal notes remain Ops-only and should not include secrets, raw prompts,
receipts, or copied user private content.

## Categorization

Category values are normalized to:

- `ai`
- `ui`
- `performance`
- `bug`
- `security`
- `accessibility`
- `feature_request`
- `other`

Submitting feedback also increments alpha feedback metrics. Bug reports and
feature requests have dedicated counters for alerting.
