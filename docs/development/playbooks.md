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

## Related docs

- [Architecture overview](../architecture/overview.md)
- [API overview](../api/overview.md)
- [Testing strategy](../testing/strategy.md)
