# Local service ports

| Component | URL / port | Health | Ready | Notes |
| --- | --- | --- | --- | --- |
| Web App | `http://localhost:3000` | — | `/api/ready` | `core`; `WEB_APP_PORT` |
| Trip Service | `http://localhost:8080` | `/health` | `/ready` | `core`; `/metrics` |
| Auth Service | `http://localhost:8082` | `/health` | `/ready` | `core`; `/metrics` |
| User Service | `http://localhost:8083` | `/health` | `/ready` | `core`; `/metrics` |
| External Integrations | `http://localhost:8084` | `/health` | `/ready` | `core`; `/metrics` |
| Notification Service | `http://localhost:8086` | `/health` | `/ready` | `core`; `/metrics` |
| Worker Service | `http://localhost:8090` | `/health` | `/ready` | `core`; operational endpoints are protected |
| AI Planning | `http://localhost:8000` | `/health` | `/ready` | `ai`/`rag`; test stack publishes `18000` by default |
| Postgres | `localhost:5432` | — | `pg_isready` | `core`; `POSTGRES_PUBLISHED_PORT` |
| RabbitMQ AMQP | `localhost:5672` | — | diagnostics ping | `core`; `RABBITMQ_PUBLISHED_PORT` |
| RabbitMQ management | `http://localhost:15672` | — | — | `guest`/configured local credentials |
| RabbitMQ metrics | `http://localhost:15692` | — | — | Prometheus scrape endpoint |
| Ollama | `http://localhost:11434` | `api/tags` | Compose `ollama list` | `ai`/`rag` |
| Prometheus | `http://localhost:9090` | `/-/healthy` | `/-/ready` | `observability` |
| Grafana | `http://localhost:3030` | `/api/health` | — | `observability` |
| Adminer | `http://localhost:8081` | — | — | `dev-tools` |

`/health` is process liveness; `/ready` includes required dependencies. Override
published values through matching `*_SERVICE_PORT`, `WEB_APP_PORT`,
`POSTGRES_PUBLISHED_PORT`, or RabbitMQ variables in `infra/.env`. See the
[Compose file](../../infra/docker-compose.yml) for the authoritative mapping.
