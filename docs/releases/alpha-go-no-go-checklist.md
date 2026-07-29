# Alpha go/no-go checklist

This checklist gathers evidence. It does not automatically produce a go decision.

## Scope

- [ ] `docs/releases/alpha-scope.md` reviewed.
- [ ] `config/feature-flags/alpha.json` reviewed.
- [ ] `scripts/release/validate-alpha-scope.sh` passes against the target env.
- [ ] Disabled features are enforced by backend flags.
- [ ] Known limitations are published.

## Quality

- [ ] Alpha Playwright suite passed.
- [ ] Alpha smoke report passed.
- [ ] Failure-recovery tests passed or opt-in fixture gaps are recorded.
- [ ] Mobile 375/390/tablet critical pages passed.
- [ ] No unresolved data corruption issue.
- [ ] Browser support decision recorded.

## Security

- [ ] No unresolved critical/high blocker.
- [ ] Public share sanitization passed.
- [ ] Ops access and audit behavior reviewed.
- [ ] OpenAI key privacy checks passed.
- [ ] CORS, headers, rate limits, and internal tokens reviewed.
- [ ] ZAP baseline findings triaged.

## Reliability

- [ ] `/health`, `/ready`, and `/version` passed for every service.
- [ ] Web `/api/health`, `/api/ready`, and `/api/version` passed.
- [ ] Worker retry/idempotency evidence captured.
- [ ] Fallback and circuit-breaker evidence captured.
- [ ] Feature kill switches tested.
- [ ] DLQ recovery path tested.

## Data

- [ ] Fresh migration check passed.
- [ ] Migration status recorded after rehearsal.
- [ ] Backup created.
- [ ] Backup verified.
- [ ] Restore into separate validation DB completed or explicitly risk-accepted.
- [ ] Non-DB storage expectations documented.

## Cost

- [ ] Daily/monthly spend caps active.
- [ ] Request/concurrency limits active.
- [ ] Token usage visible.
- [ ] Spend checker or ledger evidence attached.
- [ ] Alerts loaded in Prometheus.

## Operations

- [ ] Alpha Overview reviewed.
- [ ] Ops job retry/cancel/mark-failed tested.
- [ ] Feature flag disable actions tested with audit reason.
- [ ] Rollback rehearsal completed.
- [ ] Incident owner assigned.
- [ ] Rollback owner assigned.
- [ ] Support error catalog ready.

## Documentation

- [ ] Alpha scope published.
- [ ] OpenAI checklist complete.
- [ ] Telemetry/capacity documented.
- [ ] Known issues published.
- [ ] Alpha runbooks available.
- [ ] Release notes prepared.
- [ ] `docs/releases/alpha-launch-decision.md` signed by reviewers.
