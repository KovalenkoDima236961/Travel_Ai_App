# Data Export, Portability & Account Cleanup v1

Travel AI creates private, authenticated, short-lived export packages. A ZIP is
stored on a service-local private volume with restrictive permissions, and is
available only to the authenticated person who created it. There are no public
links, signed public URLs, or object-storage URLs in this version.

## API surface

| Service | Route | Purpose |
| --- | --- | --- |
| User | `POST /users/me/export` | Create an account-service export job. |
| User | `GET /users/me/export/{exportId}` | Read job state. |
| User | `GET /users/me/export/{exportId}/download` | Download a completed ZIP. |
| User | `POST /users/me/account-cleanup/request-deletion` | Record a reviewable cleanup request; it does not delete data. |
| Trip | `POST /trips/{tripId}/export/archive` | Create a full private trip archive. |
| Trip | `GET /trips/{tripId}/export/{exportId}` | Read job state. |
| Trip | `GET /trips/{tripId}/export/{exportId}/download` | Download a completed archive. |
| Trip | `GET /trips/{tripId}/expenses/export.csv` | Download portable expense CSV. |
| Trip | `GET /trips/{tripId}/settlements/export.csv` | Download portable settlement CSV. |
| Trip | `GET /trips/{tripId}/budget/export.csv` | Download portable budget CSV. |
| Trip | `GET /trips/{tripId}/expenses/receipts/export-metadata.csv` | Download receipt metadata only. |
| Notification | `POST /notifications/cleanup` | Permanently remove selected old notifications; unread entries remain protected by default. |

Trip archive and CSV exports require an owner or editor. Each download route
checks the requesting user and trip again; a job identifier alone is never a
capability. Download responses are `private, no-store`, carry a safe attachment
filename, and do not log file contents, receipt bytes, OCR raw text, access
tokens, or export download URLs.

## Contents and limits

Trip archives contain a manifest and README, trip structure, itinerary, route,
accommodation, budget/expense/settlement CSVs, receipt metadata, checklists,
reminders, and available recap/verification summaries. Receipt files are
**off by default** and require `includeReceiptFiles: true`; raw OCR text is
excluded. Account exports contain profile and preferences plus a nested
`trip-data.zip` handoff from Trip Service. That package contains the authorized
trip archives for the account owner/editor; optional workspace data follows the
same owner/editor rule, while read-only shared trips are recorded as skipped.

Defaults are a 24-hour retention period, 100 MB per trip archive, and 250 MB
per account-service archive. Cleanup loops remove only generated package files
and mark their jobs expired; they never delete trips, receipts, preferences, or
other source data. Configure the limits with `DATA_EXPORT_*` variables in the
service and infrastructure environment examples.

## Cleanup semantics

Offline cleanup is browser-local and scoped to the signed-in user’s IndexedDB
records. It can remove cached trip data, pending changes, receipt drafts, or
all local app cache, with a confirmation that states the scope. Clearing
pending changes intentionally discards changes that have not reached the
server. Notification cleanup hard-deletes only the requested old records and
keeps unread entries unless `onlyRead` is explicitly set to `false`.

An account-cleanup request is an auditable placeholder for a reviewed support
or legal workflow. It never triggers automatic account deletion in v1.
