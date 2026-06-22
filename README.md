# Travel AI App

AI travel planning project with Go Trip Service, Python/FastAPI AI Planning
Service, and a Next.js web app.

Web App v1 lives in `apps/web`. The local full stack runs from
`infra/docker-compose.yml`, and the browser URL is `http://localhost:3000`.
Detailed full-stack instructions are in [infra/README.md](infra/README.md).

## Local Development

Local application infrastructure is defined in `infra/docker-compose.yml`; the main
compose file intentionally lives under `infra/`, not the project root.

Start with:

```bash
cp infra/.env.example infra/.env
./scripts/dev-setup.sh
```

Run the full app smoke test with:

```bash
./scripts/smoke-test.sh
```

See `infra/README.md` for direct Docker Compose commands, Ollama model pulls,
knowledge indexing, useful URLs, and troubleshooting. The full app can be
started with:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env up --build
```
