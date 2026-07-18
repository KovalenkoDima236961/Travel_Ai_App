# Travel AI App documentation

This is the operational documentation for the repository. It is deliberately
kept beside the code and Compose configuration it describes.

## Start here

- [Getting started](development/getting-started.md) — first local run.
- [Common commands](development/common-commands.md) — copy-paste command map.
- [Architecture overview](architecture/overview.md) — services and dependencies.
- [Troubleshooting](development/troubleshooting.md) — local failures and recovery.

## Browse by topic

| Area | Documents |
| --- | --- |
| Architecture | [Overview](architecture/overview.md), [service boundaries](architecture/service-boundaries.md), [data ownership](architecture/data-ownership.md), [key flows](architecture/key-flows.md) |
| Development | [Getting started](development/getting-started.md), [environment](development/environment.md), [ports](development/ports.md), [migrations](development/migrations.md), [playbooks](development/playbooks.md) |
| API | [Overview](api/overview.md), [endpoint inventory](api/endpoint-inventory.md), [errors](api/errors.md) |
| Testing | [Strategy](testing/strategy.md), [running tests](testing/running-tests.md), [CI](testing/ci.md) |
| Security | [Audit](security/audit.md), [tools](security/tools.md), [threat model](security/threat-model.md) |
| Performance | [Backend](backend/performance.md), [frontend](frontend/performance.md), [audit](performance/performance-audit.md) |
| Operations | [Runbooks](operations/runbooks.md), [deployment](deployment/production.md), [backups](deployment/backups.md) |
| Releases | [Release process](releases/release-process.md), [hotfix process](releases/hotfix-process.md), [rollback](releases/rollback.md), [migration safety](releases/migration-safety.md), [troubleshooting](releases/troubleshooting.md) |
| Features | [Trips](features/trips.md), [AI generation](features/ai-generation.md), [offline/PWA](features/offline-pwa.md), [receipts and expenses](features/receipts-expenses.md), [notifications](features/notifications.md), [workspaces](features/workspaces-approvals-policies.md) |
| Platform | [Feature flags and runtime controls](platform/feature-flags.md) |

## Documentation ownership

Documentation is part of the definition of done. Update the relevant page in
the same change when APIs, environment variables, commands, migrations,
service boundaries, or CI workflows change. If a behavior cannot be verified
from code or a test, label it as a TODO rather than presenting it as fact.

## Related docs

- [Repository README](../README.md)
- [API endpoint inventory](api/endpoint-inventory.md)
- [Operational runbooks](operations/runbooks.md)
