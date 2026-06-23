# Travel AI App

AI travel planning project with Go Auth Service, Go Trip Service, Go User
Service, Python/FastAPI AI Planning Service, and a Next.js web app.

Auth Service v1 lives in `services/auth-service` and supports email/password
registration, login, refresh token rotation, logout, and JWT-backed `/auth/me`.
Trip Service validates those JWT access tokens locally with the shared
`JWT_ACCESS_SECRET` and scopes `/trips` data by the authenticated `sub` user ID.
User/Profile Service v1 lives in `services/user-service` and owns travel
profiles/preferences for authenticated users, also scoped by the JWT `sub`.
AI Planning Service owns itinerary generation and local travel knowledge.

Web App v1 supports register/login/logout and stores tokens in `localStorage`
for development. Secure httpOnly cookies should replace localStorage token
storage before production.

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

The smoke test registers/logs in a unique user, checks profile/preferences
defaults and updates, creates and generates a trip with
`Authorization: Bearer <accessToken>`, verifies only that user can access it,
and logs out.

See `infra/README.md` for direct Docker Compose commands, Ollama model pulls,
knowledge indexing, useful URLs, and troubleshooting. The full app can be
started with:

```bash
docker compose -f infra/docker-compose.yml --env-file infra/.env up --build
```
