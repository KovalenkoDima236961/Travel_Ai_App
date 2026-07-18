# Local Service Ports

| Component | Port | Endpoint / note |
| --- | ---: | --- |
| Web App | 3000 | `http://localhost:3000` |
| Trip Service | 8080 | `/health`, `/ready`, `/metrics` |
| Auth Service | 8082 | `/health`, `/ready`, `/metrics` |
| User Service | 8083 | `/health`, `/ready`, `/metrics` |
| External Integrations | 8084 | `/health`, `/ready`, `/metrics` |
| Notification Service | 8086 | `/health`, `/ready`, `/metrics` |
| Worker Service | 8090 | `/health`, `/ready`, `/metrics` |
| AI Planning Service | 8000 | `ai` / `rag` profiles only |
| Postgres | 5432 | `core` profile |
| RabbitMQ AMQP | 5672 | `core` profile |
| RabbitMQ management | 15672 | local-only management UI |
| RabbitMQ metrics | 15692 | Prometheus scrape target |
| Ollama | 11434 | `ai` / `rag` profiles only |
| Prometheus | 9090 | `observability` profile only |
| Grafana | 3030 | `observability` profile only |
| Adminer | 8081 | `dev-tools` profile only |

Override published ports through the matching `*_SERVICE_PORT`, `WEB_APP_PORT`,
or `AI_HTTP_PORT` variables in `infra/.env`. Keep only local development ports
exposed; the production Compose file binds backend ports to loopback by default.
