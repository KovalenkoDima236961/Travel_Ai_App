# Operational runbooks

Runbooks are safe first-response procedures for local and staging operations.
They do not authorize production destructive actions; production restores,
queue discards, credential rotation, and schema repair require the deployment
owner's approved change process.

- [Service not starting](runbooks/service-not-starting.md)
- [Migrations failed](runbooks/migrations-failed.md)
- [RabbitMQ jobs stuck](runbooks/rabbitmq-jobs-stuck.md)
- [AI generation failing](runbooks/ai-generation-failing.md)
- [AI model rollout regression](runbooks/ai-model-rollout-regression.md)
- [Provider quota exceeded](runbooks/provider-quota-exceeded.md)
- [Notifications not sending](runbooks/notifications-not-sending.md)
- [Playwright failures](runbooks/playwright-failures.md)
- [Security scan failures](runbooks/security-scan-failures.md)
- [Restore database backup](runbooks/restore-database-backup.md)
- [Alpha generation failure](runbooks/alpha-generation-failure.md)
- [Alpha worker queue stuck](runbooks/alpha-worker-queue-stuck.md)
- [Alpha OpenAI outage](runbooks/alpha-openai-outage.md)
- [Alpha spend limit reached](runbooks/alpha-spend-limit-reached.md)
- [Alpha public share incident](runbooks/alpha-public-share-incident.md)
- [Alpha rollback](runbooks/alpha-rollback.md)

## First-response commands

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env --profile core ps
docker compose -f infra/docker-compose.yml --env-file infra/.env logs --tail=200 <service>
./scripts/wait-for-ready.sh core
./scripts/migration-status.sh
```

## Related docs

- [Troubleshooting](../development/troubleshooting.md)
- [Ports](../development/ports.md)
- [Deployment checklist](../deployment/checklist.md)
- [Closed alpha scope](../releases/alpha-scope.md)
- [Alpha go/no-go checklist](../releases/alpha-go-no-go-checklist.md)

## Data lifecycle

See [cleanup failures](runbooks/cleanup-failed.md) and [storage growth](runbooks/storage-growth.md). Run cleanup dry-runs before destructive changes, and do not use lifecycle cleanup to remove protected receipts, audit events, or production backups.
