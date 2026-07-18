# Environment configuration

## Environment files

| File | Use |
| --- | --- |
| `infra/.env.example` | Canonical local starting point; copy to ignored `infra/.env`. |
| `infra/.env.local.example` | Local override/reference values. |
| `infra/.env.test.example` | Isolated, mock-only test Compose environment. |
| `infra/.env.staging.example` | Staging template. |
| `infra/.env.production.example` | Production template and safety baseline. |
| `infra/.env.ci.example` | CI-specific environment. |

`infra/.env.test.example` exists; do not substitute a test stack for local
development. `infra/.env.staging.example` is the staging template (there is no
separate `infra/.env.local` checked in).

Validate before starting:

```bash
./scripts/validate-env.sh local --env-file infra/.env
./scripts/compose-validate.sh --env-file infra/.env
```

## Variables by concern

| Concern | Key variables |
| --- | --- |
| Shared trust | `JWT_ACCESS_SECRET`, `INTERNAL_SERVICE_TOKEN`, `INTERNAL_SERVICE_TOKENS`, `CORS_ALLOWED_ORIGINS` |
| Postgres/RabbitMQ | `POSTGRES_*`, per-service `*_POSTGRES_DB`, `RABBITMQ_*`, `RABBITMQ_URL` |
| Browser endpoints | `WEB_APP_PORT`, `NEXT_PUBLIC_*_SERVICE_URL`, `*_INTERNAL_URL` |
| AI | `TRIP_ITINERARY_GENERATOR_MODE`, `AI_ITINERARY_GENERATOR_MODE`, `OLLAMA_*`, `RAG_*`, `AI_PROMPT_LOGGING_*` |
| Providers | `*_PROVIDER`, `*_API_KEY`, `*_FALLBACK_TO_MOCK`, cache/timeout/quota variables |
| Delivery | `EMAIL_*`, `SMTP_*`, `WEB_PUSH_*`, `NOTIFICATION_*` |
| Calendar and files | `GOOGLE_*`, `CALENDAR_TOKEN_ENCRYPTION_KEY`, `RECEIPT_STORAGE_*`, `*_EXPORT_*` |

The complete, current list is the annotated [`.env.example`](../../infra/.env.example),
which is the source of truth for defaults.

## Safe defaults and production checks

- Local `core` runs mock providers and Trip Service mock generation. Add `ai`
  only when you intentionally need FastAPI/Ollama; use `rag` for RAG.
- Keep API keys blank in mock mode. CI must never call real providers or send
  real email/push.
- Production/staging require strong unique JWT/internal secrets, restrictive
  CORS, non-default database/RabbitMQ credentials, and configured provider and
  calendar encryption secrets. Run `validate-env.sh staging|production`.
- Never commit `.env`, credentials, private keys, tokens, dumps, or receipt
  data. Use a secret manager for deployed values.

## Common mistakes

- Changing `NEXT_PUBLIC_*` requires rebuilding the web image because those
  values are build arguments.
- Docker service URLs use names such as `http://trip-service:8080`; browsers
  use `http://localhost:8080`.
- Every private Go service must validate JWTs with the same local
  `JWT_ACCESS_SECRET`. Internal callers and receivers must share a valid
  internal token.
- A changed RabbitMQ username/password may conflict with its existing named
  volume. Reset only local data after confirming the target.

## Related docs

- [Getting started](getting-started.md)
- [Security configuration](../security/config-hardening.md)
- [Troubleshooting](troubleshooting.md)
