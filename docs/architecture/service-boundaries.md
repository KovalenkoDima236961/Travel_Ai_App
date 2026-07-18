# Service boundaries

Use a service's public HTTP API or an authenticated `/internal/*` lookup route;
never bypass its rules by reading or writing its database.

| Service | Owns | Does not own | Depends on / called by | Main endpoints and gotchas |
| --- | --- | --- | --- | --- |
| Auth | `users`, `refresh_tokens`; password and token lifecycle | Profiles, trips, workspace roles | Postgres; called by Web and internal user lookup callers | `/auth/*`, `/internal/users/*`. JWT secret must match private services; never log tokens. |
| User | `user_profiles`, `user_preferences`, workspaces, memberships, account exports | Auth credentials, trip records | Auth JWT; Auth/Notification/Trip internal calls; called by Web/Trip | `/users/me/*`, `/workspaces/*`, `/internal/workspaces/*`. Derive user from JWT, never body `userId`. |
| Trip | Trips, revisions, shares, collaborators, jobs, budgets, receipts, policies and feature state | Identity/profile canonical data, notification delivery, provider credentials | User, Notification, External, AI, RabbitMQ; called by Web/Worker | `/trips/*`, `/trip-templates/*`, public shares and ops. Revision-sensitive writes need `expectedItineraryRevision`; enforce trip/workspace permissions. |
| Notification | Notifications, preferences, push subscriptions, digest/mute/dedupe data | Trip authorization, user profile truth | Auth/User lookups; called by Trip/Worker and Web | `/notifications/*`, `/internal/notifications/batch`. Never notify actor/self or include sensitive payloads. |
| Worker | Job execution/consumer state and operational queue handling | Authoritative trip, notification, or provider data | RabbitMQ, Trip, Notification; ops users | `/ops/worker/*`, `/ops/queues/*`, `/ops/dlq/*`. Idempotency, retry limits and DLQ handling are required. |
| External Integrations | Calendar connection/OAuth state and provider quota usage | Trip persistence, web credentials | Provider APIs, Postgres; called by Trip/Web calendar proxy | `/places`, `/routes`, `/weather`, `/calendar`, `/ops/providers/*`. Default to mock; tokens are encrypted and provider outputs are estimates. |
| AI Planning | Prompt builders, schemas, generation/repair/discovery/RAG behavior | Trip persistence, permissions, user identity | Ollama/Chroma; called by Trip | `/generate-itinerary`, `/repair-*`, `/suggest-*`. Return strict validated JSON only; do not log raw prompts. |
| Web App | Browser UI, cached companion data, local pending mutations, i18n | Server authority, secrets, authorization decisions | All public APIs/Next proxies | `src/app`, `src/lib/api`, hooks. UI permission gates are advisory; backend remains authoritative. |

## Database rule

Each service owns the tables created in its own `migrations/` directory. The
only valid cross-service data access is through a documented public or internal
endpoint. A reference such as `user_id` is an identity reference, not a foreign
key permission to write Auth/User data.

## Related docs

- [Data ownership](data-ownership.md)
- [Endpoint inventory](../api/endpoint-inventory.md)
- [Adding a backend endpoint](../development/playbooks/add-backend-endpoint.md)
