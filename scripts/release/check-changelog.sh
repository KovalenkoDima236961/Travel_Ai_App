#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
CHANGELOG="${PROJECT_ROOT}/CHANGELOG.md"
VERSION="$(tr -d '[:space:]' < "${PROJECT_ROOT}/VERSION")"

[[ -f "${CHANGELOG}" ]] || { echo "CHANGELOG.md is required." >&2; exit 1; }
grep -q '^## \[Unreleased\]' "${CHANGELOG}" || { echo "CHANGELOG.md must contain a [Unreleased] section." >&2; exit 1; }

for section in Added Changed Deprecated Removed Fixed Security "Migration Notes" "API Contract Changes" "Known Issues"; do
  grep -q "^### ${section}$" "${CHANGELOG}" || { echo "CHANGELOG.md Unreleased is missing '${section}'." >&2; exit 1; }
done

if git -C "${PROJECT_ROOT}" diff --quiet HEAD -- VERSION 2>/dev/null && ! grep -q "^## \[${VERSION}\]" "${CHANGELOG}"; then
  echo "WARN VERSION ${VERSION} has no dated changelog section yet; prepare-release will create one." >&2
else
  echo "PASS changelog structure is valid for ${VERSION}."
fi
