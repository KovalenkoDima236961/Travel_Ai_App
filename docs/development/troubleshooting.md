# Local Development Troubleshooting

| Symptom | Check / recovery |
| --- | --- |
| A port is already in use | Change the matching port variable in `infra/.env`, or stop the conflicting process. |
| Postgres never becomes ready | Run `docker compose -f infra/docker-compose.yml --env-file infra/.env logs postgres`. A stale schema can be reset only with `./scripts/dev-reset.sh --yes`. |
| Migration runner fails | Run `./scripts/migration-status.sh`; correct the first failure, then rerun `./scripts/run-migrations.sh`. Back up before schema repair. |
| RabbitMQ authentication fails | Keep `RABBITMQ_URL`, `RABBITMQ_USER`, and `RABBITMQ_PASSWORD` aligned, then recreate only local data if credentials changed after first boot. |
| AI Planning Service is absent | This is expected for `core`. Start `--profile ai` and run the smoke test with `--ai`. |
| Ollama is not ready | Start the `ai` profile and pull the configured model with `./scripts/dev-setup.sh --ai`; use `TRIP_ITINERARY_GENERATOR_MODE=mock` while offline. |
| RAG returns no results | Enable `rag`, pull the embedding model, then run `./scripts/index-knowledge.sh`. |
| Browser cannot reach an API | Check the `NEXT_PUBLIC_*` browser URLs, the `*_INTERNAL_URL` Docker URLs, and `CORS_ALLOWED_ORIGINS`; rebuild the Web App after changing public values. |
| JWT or internal calls fail | Ensure all services use the same local `JWT_ACCESS_SECRET` and `INTERNAL_SERVICE_TOKEN`. |
| A container is unhealthy | Use `./scripts/wait-for-ready.sh core` to identify the failing service and inspect its logs. `/health` is process liveness; `/ready` checks required dependencies. |
| Provider key is reported missing | Keep the provider in `mock` mode locally, or configure the corresponding key and rerun `./scripts/validate-env.sh local`. |
| Grafana has no data | Start `--profile observability`; inspect `http://localhost:9090/targets`, then verify the service `/metrics` endpoint. |
| Local files disappeared | Receipts and exports live in named volumes and are removed by `down -v` / `dev-reset`. Back up Postgres separately and copy private file volumes when needed. |

For a clean local rebuild, first consider `./scripts/backup-postgres.sh`, then run
`./scripts/dev-reset.sh --yes`. The reset command deliberately refuses to run
without the explicit confirmation flag.
