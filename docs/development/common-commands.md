# Common commands

Run these from the repository root (`Travel_Ai_App`). Commands use the existing
scripts and `infra/docker-compose.yml`.

| Task | Command |
| --- | --- |
| Prepare/start core | `./scripts/dev-setup.sh` |
| Start core manually | `docker compose -f infra/docker-compose.yml --env-file infra/.env --profile core up -d` |
| Start full local AI/RAG/observability | `./scripts/dev-setup.sh --rag --observability` |
| Run all migrations / one service | `./scripts/run-migrations.sh` / `./scripts/run-migrations.sh trip-service` |
| Inspect migrations | `./scripts/migration-status.sh` |
| Wait for readiness | `./scripts/wait-for-ready.sh core` (or `core ai`) |
| Core smoke | `./scripts/smoke-test.sh --core` |
| Security scan | `./scripts/security-scan.sh` (`--audit`, optional `--zap`) |
| Frontend lint/type/test/build | `./scripts/test-frontend.sh` |
| Browser E2E | `./scripts/test-frontend-e2e.sh` |
| All Go modules | `./scripts/test-go.sh` |
| Python/FastAPI | `./scripts/test-python.sh` |
| Backend integration | `./scripts/test-backend-integration.sh` |
| Full test pyramid | `./scripts/test-all.sh` |
| Start/stop isolated test stack | `./scripts/test-stack-up.sh` / `./scripts/test-stack-down.sh` |
| Validate environment/Compose | `./scripts/validate-env.sh local --env-file infra/.env` / `./scripts/compose-validate.sh --env-file infra/.env` |
| Backup Postgres | `./scripts/backup-postgres.sh --output ./backups --gzip` |
| Verify backup | `./scripts/verify-backup.sh <backup-file-or-directory>` |
| Restore local backup | `./scripts/restore-postgres.sh <backup-file-or-directory> --yes` |
| Reset local stack (destructive) | `./scripts/dev-reset.sh --yes --backup` |

For a direct web loop, use `cd apps/web && npm run dev`; for direct service
checks use the service `Makefile` (`make fmt`, `make vet`, `make test`,
`make build`) where provided. Python commands are described in the AI service
README and `scripts/test-python.sh`.

## Related docs

- [Running tests](../testing/running-tests.md)
- [Deployment backups](../deployment/backups.md)
- [Operations runbooks](../operations/runbooks.md)
