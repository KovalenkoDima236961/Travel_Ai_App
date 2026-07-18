# Release process

## Goals and versioning

Releases must be reproducible, inspectable, and reversible. v1 uses one semantic project version from the repository-root [`VERSION`](../../VERSION) file; services do not receive independent versions. A release `0.2.0` produces `service:0.2.0` and `service:0.2.0-<shortSha>` images.

- **MAJOR**: a breaking public API, data, configuration, or deployment change.
- **MINOR**: backward-compatible features.
- **PATCH**: backward-compatible bug, security, or performance fixes.
- Pre-releases use SemVer identifiers such as `0.2.0-rc.1` or `0.2.0-beta.1`.
- Build metadata is published separately as the full Git SHA and UTC build time. It does not alter the release version.

Every service exposes public non-sensitive `GET /version`: service, version, git SHA, build time, environment, and API contract version. The Web App exposes the same data at `GET /api/version` and a subtle Settings build line.

## Branches and tags

Develop and merge normal changes through the repository’s reviewed branch flow. Cut a short-lived `release/<version>` branch only when stabilization requires it; otherwise release from the reviewed commit on `main`. The immutable release reference is an annotated Git tag such as `v0.2.0`; do not retag or move published tags.

## Checklist and release gates

1. Select the SemVer bump and update the Unreleased entries in [`CHANGELOG.md`](../../CHANGELOG.md), including Security, Migration Notes, API Contract Changes, and Known Issues.
2. If an OpenAPI document changed, update [`docs/api/contract-changelog.md`](../api/contract-changelog.md), regenerate Web types, and typecheck the Web App.
3. Review migrations using [migration safety](migration-safety.md). Take and verify a production backup before any production schema change.
4. Run `./scripts/release/prepare-release.sh <version>` and review the generated notes.
5. Run `./scripts/release/check-release.sh ci`. Required gates are tests, contracts, fresh migrations, security scans, Docker builds, and smoke/E2E checks. Local exploratory runs may explicitly pass a skip flag; CI release mode may not.
6. Build version/SHA images with `REGISTRY=<registry> ./scripts/release/build-images.sh`; push only with an explicit `--push` or `push-images.sh`.
7. Run staging-like verification below, review [rollback](rollback.md), then create the annotated tag with `./scripts/release/tag-release.sh <version>`.

When a release adds a runtime flag, list its key, production default, owner,
required dependency configuration, rollout owner, and rollback action in the
release notes. Ship risky code disabled when practical; disabling its audited
database override is the first rollback control before a redeploy.

## Artifacts

Release CI retains release notes, image-tag list, OpenAPI specifications, generated client output when present, test reports, security summaries, and migration-check output. Never upload environment files, credentials, exports, receipts, or sensitive logs.

## Staging-like verification

Build images first, then use Compose production definitions with the no-bind-mount override:

```bash
cp infra/.env.staging.example infra/.env.staging
# replace all placeholder secrets, then build matching images
REGISTRY=travel-ai ./scripts/release/build-images.sh
./scripts/release/up-staging-like.sh
./scripts/release/down-staging-like.sh
```

The stack runs the immutable `<version>-<shortSha>` image tag with mock providers by default, isolated named volumes, readiness checks, migrations, `GET /version`, and the existing authenticated smoke flow. It does not download Ollama models or require real provider keys.

## Production/manual deployment

Production deployment is intentionally manual in v1. An operator selects the immutable `<version>-<shortSha>` tag, supplies managed secrets, verifies backup health, runs migrations once, deploys each service according to the existing Compose/hosting process, then runs `check-versions.sh` and `smoke-release.sh`. Record the deployed image digests and operator in the change record.

## Rollback decision guide

Roll back quickly for a widespread request failure, security regression introduced by the release, incompatible API deployment, worker backlog growth, or user-facing corruption with a known-good prior image. First disable an affected integration or stop a worker when that narrows impact. Do not automatically roll back a database after migrations: schema/data compatibility determines whether an app rollback is safe. Follow the scenario-specific [rollback playbook](rollback.md) and use a forward fix where rollback would violate migration compatibility.

For urgent releases see the [hotfix process](hotfix-process.md), and for common failures see [troubleshooting](troubleshooting.md).
# Data lifecycle release check

For every new table, uploaded/generated file, provider cache, job, or proposal, define its retention category, owner, cleanup method, and environment configuration in `docs/data/retention-policy.md` before release.
