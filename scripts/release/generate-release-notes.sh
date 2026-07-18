#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
# shellcheck source=version-info.sh
source "${SCRIPT_DIR}/version-info.sh"

output_dir="${PROJECT_ROOT}/dist/release-notes"
output="${output_dir}/${VERSION}.md"
mkdir -p "${output_dir}"

unreleased="$(awk '/^## \[Unreleased\]/{capture=1; next} /^## \[/{capture=0} capture {print}' "${PROJECT_ROOT}/CHANGELOG.md")"
last_tag="$(git -C "${PROJECT_ROOT}" describe --tags --abbrev=0 2>/dev/null || true)"
commits=""
if [[ -n "${last_tag}" ]]; then
  commits="$(git -C "${PROJECT_ROOT}" log --format='- %s (%h)' "${last_tag}..HEAD")"
else
  commits="$(git -C "${PROJECT_ROOT}" log --format='- %s (%h)' -20)"
fi
if [[ -n "${last_tag}" ]]; then
  migrations="$(git -C "${PROJECT_ROOT}" diff --name-only "${last_tag}..HEAD" -- 'services/*/migrations/*.sql' 2>/dev/null || true)"
else
  migrations="$(git -C "${PROJECT_ROOT}" ls-files 'services/*/migrations/*.sql')"
fi

{
  printf '# Travel AI %s\n\n## Summary\n\nRelease candidate built from `%s` at %s.\n\n## Highlights\n\n' "${VERSION}" "${GIT_SHA}" "${BUILD_TIME}"
  printf '%s\n\n## Changes\n\n%s\n\n' "${unreleased:-_No Unreleased notes were recorded._}" "${commits:-_No commits found since the previous tag._}"
  cat <<'EOF'
## Security

See the **Security** section in CHANGELOG.md and the CI security-scan artifact.

## API Contract Changes

See [docs/api/contract-changelog.md](../../docs/api/contract-changelog.md). Published OpenAPI specifications are release artifacts.

## Database Migrations

EOF
  if [[ -n "${migrations}" ]]; then
    printf '```text\n%s\n```\n\n' "${migrations}"
  else
    printf '_No migration files changed since the previous tag._\n\n'
  fi
  cat <<EOF
## Operational Notes

- Images: \`<registry>/<service>:${IMAGE_TAG_VERSION}\` and \`<registry>/<service>:${IMAGE_TAG_SHA}\`.
- Use \`scripts/release/up-staging-like.sh\` for the Docker Compose verification stack.

## Verification Checklist

- [ ] Required release checks passed.
- [ ] Migration safety and rollback notes reviewed.
- [ ] Version metadata matches this release.

## Rollback Notes

See [docs/releases/rollback.md](../../docs/releases/rollback.md). Database migrations are not automatically rolled back.

## Known Issues

See the **Known Issues** section in CHANGELOG.md.
EOF
} > "${output}"

echo "Wrote ${output#"${PROJECT_ROOT}/"}."
