# Data ownership and migrations

## Database ownership

| Database / service | Tables or table families |
| --- | --- |
| Auth Service | `users`, `refresh_tokens` |
| User Service | `user_profiles`, `user_preferences`, `workspaces`, `workspace_members`, `workspace_invitations`, account cleanup/export jobs |
| Trip Service | `trips`, itinerary versions, shares, collaborators, comments/activity, generation jobs/traces, routes, budgets, templates, polls, checklists/reminders, expenses/receipts/settlements, approvals/policies, recaps, exports and personalization feedback |
| Notification Service | notifications, preferences/settings/mutes, push subscriptions, digest batches/items and dedupe records |
| External Integrations | calendar connections/OAuth states, provider daily usage/totals |
| Worker / AI / Web | No independent application tables in v1. AI returns data and Web uses browser storage. Worker has an intentional v1 legacy direct-write integration for Trip Service job state; do not extend it—new cross-service work uses authenticated APIs. |

The Postgres container hosts separate service databases; it is not a shared
application schema. No cross-service table writes are permitted apart from the
documented Worker-to-Trip job-state compatibility path above. For lookup needs
use existing internal endpoints, for example Auth user batch lookup and User
workspace access/list endpoints, with the internal token header.

## Migrations

- A database-owning Go service stores paired, ordered SQL migrations under
  `services/<service>/migrations/`.
- The Compose `migration-runner` applies Auth, User, Trip, Notification, then
  External Integrations migrations. Run `./scripts/run-migrations.sh` and
  inspect with `./scripts/migration-status.sh`.
- Add a forward `*_up.sql`; add `*_down.sql` only when rollback is genuinely
  safe. Production recovery usually means a compensating forward migration or
  restoring a verified backup, not automatic down migration.

## Test databases

`./scripts/test-stack-up.sh` creates an isolated `travel-ai-test` Compose
project with mock modes and test-only ports. `test-stack-reset.sh` refuses to
touch non-test projects. Integration tests must use this isolated stack; do not
aim tests at a developer or production database.

## Related docs

- [Migration playbook](../development/playbooks/add-database-migration.md)
- [Migrations runbook](../operations/runbooks/migrations-failed.md)
- [Backups](../deployment/backups.md)
