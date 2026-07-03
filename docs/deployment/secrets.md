# Secrets

Never commit filled `.env` files. Keep production secrets in the host
environment, a restricted env file, or your platform's secret store.

## Required Secrets

- `POSTGRES_PASSWORD`
- `RABBITMQ_PASSWORD`
- `RABBITMQ_MANAGEMENT_PASSWORD`
- `JWT_ACCESS_SECRET`
- `JWT_REFRESH_SECRET`
- `INTERNAL_SERVICE_TOKEN`
- `NOTIFICATION_SERVICE_TOKEN`
- `OPS_INTERNAL_SERVICE_TOKEN`
- `PUBLIC_SHARE_ACCESS_SECRET`
- `CALENDAR_TOKEN_ENCRYPTION_KEY` when calendar sync is enabled
- `WEB_PUSH_VAPID_PRIVATE_KEY` when web push is enabled
- `SMTP_PASSWORD` when `EMAIL_PROVIDER=smtp`
- Provider API keys when their real providers are enabled

## Generate Strong Values

```sh
openssl rand -base64 32
```

Use a different value for each secret. Do not reuse JWT secrets as internal
service tokens.

## VAPID Keys

```sh
npx web-push generate-vapid-keys
```

Expose only `WEB_PUSH_VAPID_PUBLIC_KEY` to the browser. Keep
`WEB_PUSH_VAPID_PRIVATE_KEY` server-side.

## Calendar Encryption Key

`CALENDAR_TOKEN_ENCRYPTION_KEY` must be exactly 16, 24, or 32 bytes because the
service uses AES-GCM. Prefer 32 bytes. Rotate by deploying code that can decrypt
old and new values before replacing stored tokens.

## Rotation

Rotate one secret class at a time:

1. Add the new secret.
2. Deploy services that can use it.
3. Expire old sessions or tokens if the rotated value signs user credentials.
4. Remove the old secret after verification.
