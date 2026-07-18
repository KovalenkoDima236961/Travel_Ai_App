# Developer playbooks

Use these checklists while making a focused change. They point to current
repository conventions rather than replacing code review.

- [Add a backend endpoint](playbooks/add-backend-endpoint.md)
- [Add a database migration](playbooks/add-database-migration.md)
- [Add a frontend page](playbooks/add-frontend-page.md)
- [Add a TanStack Query hook](playbooks/add-query-hook.md)
- [Add a notification type](playbooks/add-notification-type.md)
- [Add an AI generation job](playbooks/add-generation-job.md)
- [Add an external provider](playbooks/add-external-provider.md)
- [Add a security-sensitive feature](playbooks/add-security-sensitive-feature.md)
- [Add a Playwright test](playbooks/add-playwright-test.md)

## Common commands

```bash
./scripts/test-frontend.sh
./scripts/test-go.sh
./scripts/test-python.sh
./scripts/test-backend-integration.sh
./scripts/security-scan.sh
```

## Release implications

- A backend endpoint or changed response may require an OpenAPI update, generated Web client update, and an entry in [the API contract changelog](../api/contract-changelog.md).
- A migration requires the migration safety checklist and a `CHANGELOG.md` Migration Notes entry.
- A new or changed environment variable requires the relevant non-secret environment template and release notes when operators must act.
- Security fixes require a `CHANGELOG.md` Security entry and the hotfix process when they cannot wait for a normal release.

## Related docs

- [Architecture overview](../architecture/overview.md)
- [API overview](../api/overview.md)
- [Testing strategy](../testing/strategy.md)
