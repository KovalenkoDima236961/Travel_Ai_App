# Runbook: service not starting

1. Identify the exact failed unit and exit state:

   ```bash
   docker compose -f infra/docker-compose.yml --env-file infra/.env --profile core ps
   docker compose -f infra/docker-compose.yml --env-file infra/.env logs --tail=200 <service>
   ```

2. Check `GET /health` and `GET /ready` where the process starts. A healthy
   process may be unready because Postgres/RabbitMQ/Ollama is unavailable.
3. Validate configuration: `./scripts/validate-env.sh local --env-file infra/.env`
   and `./scripts/compose-validate.sh --env-file infra/.env`. Check host port
   collisions with `lsof -nP -iTCP:<port> -sTCP:LISTEN`.
4. Check dependencies: Postgres health, RabbitMQ UI/credentials, required
   migrations (`./scripts/migration-status.sh`), and `ai` profile for AI use.
5. Correct the first cause, then recreate only the affected service:

   ```bash
   docker compose -f infra/docker-compose.yml --env-file infra/.env --profile core up -d --build <service>
   ./scripts/wait-for-ready.sh core
   ```

Do not delete volumes to solve a startup failure until logs/configuration and a
backup have been reviewed. Escalate staging/production failures with logs,
request IDs, image version, config diff (without secrets), and dependency state.
