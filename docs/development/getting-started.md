# Getting started

The fastest supported local path is mock-first. It needs Docker Desktop and
does not need real provider keys, SMTP, push credentials, or Ollama.

## Prerequisites

- Docker Desktop with Compose v2
- Git, Bash, `curl`, and `jq` (required by smoke scripts)
- Node.js 22/npm for browser-only work; Go/Python only when running service
  suites outside containers

## First run

```bash
git clone <repository-url>
cd Travel_Ai_App
cp infra/.env.example infra/.env
./scripts/validate-env.sh local --env-file infra/.env
./scripts/compose-validate.sh --env-file infra/.env
./scripts/dev-setup.sh --build
./scripts/smoke-test.sh --core
```

Open http://localhost:3000. The setup script starts Postgres first, applies
migrations with the Compose migration runner, starts the `core` profile, and
waits for readiness.

## Profiles

| Need | Command |
| --- | --- |
| Core mock-first stack | `./scripts/dev-setup.sh` |
| Rebuild images | `./scripts/dev-setup.sh --build` |
| AI/Ollama | `./scripts/dev-setup.sh --ai` |
| AI plus RAG | `./scripts/dev-setup.sh --rag` |
| Metrics dashboards | `./scripts/dev-setup.sh --observability` |
| Prepare env only | `./scripts/dev-setup.sh --prepare-only` |

To start Compose manually, always pass the checked local environment file:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env --profile core up -d
./scripts/run-migrations.sh
./scripts/wait-for-ready.sh core
```

## Daily workflows

- Run all local quality checks with `./scripts/test-all.sh`, or use the faster
  service-specific commands in [common commands](common-commands.md).
- Tail a service with `docker compose -f infra/docker-compose.yml --env-file infra/.env logs -f trip-service`.
- Check every Compose health state with `docker compose -f infra/docker-compose.yml --env-file infra/.env --profile core ps`.
- Browser-only development: `cd apps/web && npm run dev`; the app uses the
  `NEXT_PUBLIC_*` API URLs from its environment.

## Stop or reset

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env --profile core down
# Destructive: removes local Compose volumes. Optional backup first.
./scripts/dev-reset.sh --yes --backup
```

Logs stay in container output; persistent Postgres, RabbitMQ, Ollama, exports,
and receipts are named volumes. `down -v` and `dev-reset` remove local volume
data, so back up data you care about first.

## Related docs

- [Environment](environment.md)
- [Ports](ports.md)
- [Troubleshooting](troubleshooting.md)
- [Common commands](common-commands.md)
