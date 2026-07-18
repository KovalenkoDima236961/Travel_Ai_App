# Local development troubleshooting

Start every investigation with the affected service logs and readiness:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env --profile core ps
docker compose -f infra/docker-compose.yml --env-file infra/.env logs --tail=200 <service>
./scripts/wait-for-ready.sh core
```

| Symptom | Likely cause and diagnosis | Fix |
| --- | --- | --- |
| Docker/Compose is unavailable | `docker version`; `docker compose version` | Start Docker Desktop; install Compose v2. |
| Port already in use | `lsof -nP -iTCP:3000 -sTCP:LISTEN` (replace port) | Stop the process or change the matching published-port variable in `infra/.env`. |
| Postgres unhealthy | `docker compose ... logs postgres`; `docker compose ... exec postgres pg_isready -U postgres` | Correct `POSTGRES_*`; for disposable local data use `./scripts/dev-reset.sh --yes --backup`. |
| Migrations failed/schema mismatch | `./scripts/migration-status.sh`; migration-runner logs | Fix the first error and rerun `./scripts/run-migrations.sh`; do not edit applied migrations. See [migration runbook](../operations/runbooks/migrations-failed.md). |
| RabbitMQ auth or connection failure | RabbitMQ logs and `http://localhost:15672`; compare `RABBITMQ_URL`, user, password | Keep all values aligned. If credentials changed after initial boot, reset only confirmed local RabbitMQ data. |
| AI cannot reach Ollama | `docker compose ... --profile ai ps`; `curl -fsS http://localhost:8000/ready` | Run `./scripts/dev-setup.sh --ai`; use Trip mock mode while offline. |
| Chroma/RAG not ready | AI logs show embedding/Chroma error; verify `RAG_ENABLED` | Start `--rag`, pull embedding model via setup, then `./scripts/index-knowledge.sh`. |
| Web cannot reach backend/CORS failure | Browser Network tab; verify `NEXT_PUBLIC_*`, CORS logs | Browser URLs must be localhost, Docker internal URLs use service names; rebuild web after changing `NEXT_PUBLIC_*`. |
| JWT or internal token mismatch | Service logs show 401/403; compare secrets in `.env` | Align `JWT_ACCESS_SECRET` for private services and valid `INTERNAL_SERVICE_TOKEN` for `/internal/*`. |
| Provider key missing/quota | External service logs; `GET /ops/providers/status` only if ops access enabled | Use mock provider locally, or set the provider key and validate env. Do not put real provider calls in CI. |
| Email/push appears disabled | Notification logs; inspect `EMAIL_PROVIDER`, `WEB_PUSH_ENABLED` | Mock email is local default; configure SMTP/VAPID only in approved environments. |
| Playwright cannot connect | Run `./scripts/test-stack-up.sh`; inspect test project logs | Install Chromium (`cd apps/web && npx playwright install chromium`), free test ports, or run [Playwright runbook](../operations/runbooks/playwright-failures.md). |
| Test stack is stale | Test data/ports persisted; inspect `docker compose -p travel-ai-test ... ps` | `./scripts/test-stack-reset.sh` only deletes the isolated test project. |
| Worker jobs stuck | Worker `/ready`, RabbitMQ UI and Trip ops jobs | Follow the [RabbitMQ runbook](../operations/runbooks/rabbitmq-jobs-stuck.md); do not discard DLQ messages before recording cause. |
| Public share fails | Check share status/unlock request and expiry/password | Verify token, expiry, unlock cookie/token flow; use browser-safe public endpoint, never a private trip route. |
| Receipt upload path error | Trip logs; check `RECEIPT_STORAGE_PROVIDER` and volume | Use supported type/size; verify storage directory/volume permissions; do not hand-edit receipt paths. |
| Service is unhealthy | `/health` works but `/ready` fails | Inspect dependency readiness and migrations. `/health` is liveness; `/ready` checks required dependencies. |
| Grafana has no data | `http://localhost:9090/targets`; service `/metrics` | Start `--observability`, repair failed scrape targets, then use Grafana at `:3030`. |

Receipts and exports use named volumes and are removed by `down -v` or
`dev-reset`. Back up data before destructive recovery.

## Related docs

- [Getting started](getting-started.md)
- [Operations runbooks](../operations/runbooks.md)
- [Ports](ports.md)
