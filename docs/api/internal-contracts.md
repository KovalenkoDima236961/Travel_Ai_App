# Internal API contracts

Internal routes are private service-network contracts, not Web App APIs. They
require `X-Internal-Service-Token`; callers must propagate request IDs and
return only the minimum data required for the operation.

| Endpoint | Service | Known caller | Contract and failure behavior |
| --- | --- | --- | --- |
| `GET /internal/users/by-email`, `POST /internal/users/batch` | Auth | Trip/Notification services | Sanitized registered-user lookup; missing/invalid token is `401/403`, unknown users are omitted/not found without secrets. |
| `/internal/workspaces/*` | User | Trip service | Workspace/member authorization lookup; token required and workspace privacy is preserved. |
| `POST /internal/notifications/batch`, `/process-digests` | Notification | Trip/Worker | Bounded notification fan-out/digest work; invalid payload is rejected without delivering partial secrets. |
| `POST /internal/calendar/google/events/sync`, `/delete` | External integrations | Trip | Calendar event lifecycle; tokens and provider OAuth material never appear in responses. |
| `POST /internal/reminders/process-due` | Trip | Worker | Scheduled reminder processing; request is trusted but bounded and idempotent at job level. |
| `POST /internal/data-exports/account-package` | Trip | User | Private account export package; only authorized service caller can obtain metadata. |

Before adding an internal endpoint, record the caller, authentication, request,
response, retry/idempotency and privacy rules here. Add a focused test proving
that a missing internal token is denied and that response DTOs omit secrets.
