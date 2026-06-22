# Travel AI App

Backend-only AI travel planning project.

## Local Development

Local backend infrastructure is defined in `infra/docker-compose.yml`; the main
compose file intentionally lives under `infra/`, not the project root.

Start with:

```bash
cp infra/.env.example infra/.env
./scripts/dev-setup.sh
```

Run the full backend smoke test with:

```bash
./scripts/smoke-test.sh
```

See `infra/README.md` for direct Docker Compose commands, Ollama model pulls,
knowledge indexing, useful URLs, and troubleshooting.
