# Feature flags and runtime controls

## Purpose and ownership

Feature flags separate deployment from release: a reviewed capability can ship
disabled, be enabled once its dependencies are ready, and be turned off as an
operational kill switch without a redeploy. Trip Service owns v1 because it
already owns the shared operations routes, PostgreSQL migration lifecycle, and
ops audit pattern. This is intentionally a small internal control plane, not
an experimentation system.

Flags are stable `snake_case` registry entries in
`services/trip-service/internal/featureflags`. Only registered flags may be
changed outside local development. V1 supports global database overrides; the
schema reserves workspace and user scopes but they are not evaluated yet.

## Sources and evaluation

The evaluator resolves a flag deterministically:

1. reviewed hardcoded registry default;
2. deployment environment default (`FEATURE_<FLAG_NAME_UPPER>`);
3. matching database global override for `APP_ENV` (with a generic override
   available for a future migration path);
4. future workspace and user scopes, when explicitly implemented.

Values are typed (`boolean`, `string`, or `int`); the current registry uses
booleans. Feature flags never hold credentials, tokens, URLs containing
credentials, or arbitrary JSON. The in-process cache is concurrency-safe,
defaults to 30 seconds, and is invalidated immediately after an ops change.

`APP_ENV` is `local`, `test`, `staging`, or `production`. A failed runtime
lookup is fail-closed for backend-enforced flags in staging/production and
returns safe local defaults in local/test. `FEATURE_FLAGS_FAIL_CLOSED=true`
can opt into the conservative behavior elsewhere.

## Enforcement and frontend use

The frontend fetches only the safe boolean projection from
`GET /feature-flags/public` using TanStack Query. It uses conservative fallback
values when that request fails; browser flags are strictly a UX aid and are not
authorization.

Trip Service enforces risky routes before side effects and returns:

```json
{"error":{"code":"feature_disabled","message":"This feature is currently disabled.","details":{"feature":"copilot_enabled"},"requestId":"..."}}
```

Current enforcement covers generation/regeneration, repair and policy repair,
Copilot, route alternatives, exports, sharing creation and update, calendar
sync/import, transport search, receipt OCR extraction, template adaptation,
and workspace approval actions. Disabling public sharing prevents new shares
or changes but deliberately leaves an already-issued public read link usable;
disable the link itself when it must be revoked.

Other services retain their own static safety configuration for real providers,
calendar OAuth, email, and push. Before turning on their corresponding runtime
flag, ensure that service’s configuration and credentials are present; the
environment validator checks that dependency relationship in strict
environments.

## Operations and auditing

Allowlisted ops admins use:

- `GET /ops/feature-flags`
- `GET/PATCH /ops/feature-flags/{key}`
- `POST /ops/feature-flags/{key}/reset`
- `GET /ops/feature-flags/{key}/audit`

Ops routes require the normal JWT, `OPS_ADMIN_EMAILS`, static
`OPS_DASHBOARD_ENABLED`, and the runtime `ops_dashboard_enabled` flag. Every
change is transactional with an audit event containing actor, old/new safe
value, reason, environment, scope, request ID, action, and timestamp. Reasons
are required in staging and production. The Web Ops panel uses these endpoints
and confirms changes to public sharing, exports, providers, and calendar sync.

## Adding a flag

1. Add a typed, documented registry definition with conservative production
   and useful local defaults.
2. Add the `FEATURE_*` example values and strict dependency validation when
   the capability needs external configuration.
3. Gate the owning backend route or service before any side effect; add tests
   for enabled, disabled, and no-side-effect behavior.
4. Mark it frontend-safe only if a boolean reveals no sensitive information;
   use the provider/gate to hide the associated UI.
5. Update OpenAPI, endpoint/error docs, release notes, and this page.

Do not use flags for authentication, authorization, entitlement, secrets,
schema compatibility, permanent user settings, or A/B experiments. Those have
separate ownership and audit requirements.

## Environment behavior

Local and test use mock-friendly defaults and can fall back to registry values
without a running database. Staging is fail-closed so operators can verify
rollout behavior. Production defaults keep real providers, calendar sync,
receipt OCR, web push, public sharing, policy repair, and the Ops Dashboard
off until configuration and a rollout decision explicitly enable them. Release
notes must name any new flag, its production default, dependencies, owner, and
rollback action. The first rollback action for a faulty flagged capability is
to disable its database override, then investigate before redeploying.
