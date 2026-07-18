#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"

usage() { echo "Usage: scripts/release/tag-release.sh VERSION [--push]" >&2; }
[[ $# -ge 1 ]] || { usage; exit 2; }
VERSION="$1"; shift
push=false
while [[ $# -gt 0 ]]; do
  case "$1" in --push) push=true; shift ;; *) usage; exit 2 ;; esac
done

[[ "$(tr -d '[:space:]' < "${PROJECT_ROOT}/VERSION")" == "${VERSION}" ]] || { echo "VERSION does not match ${VERSION}." >&2; exit 1; }
git -C "${PROJECT_ROOT}" diff --quiet || { echo "Working tree has unstaged changes; commit or stash them before tagging." >&2; exit 1; }
git -C "${PROJECT_ROOT}" diff --cached --quiet || { echo "Working tree has staged changes; commit them before tagging." >&2; exit 1; }
git -C "${PROJECT_ROOT}" rev-parse "v${VERSION}" >/dev/null 2>&1 && { echo "Tag v${VERSION} already exists." >&2; exit 1; }

notes="dist/release-notes/${VERSION}.md"
[[ -f "${PROJECT_ROOT}/${notes}" ]] || VERSION="${VERSION}" "${SCRIPT_DIR}/generate-release-notes.sh"
git -C "${PROJECT_ROOT}" tag -a "v${VERSION}" -m "Release v${VERSION}; notes: ${notes}"
echo "Created annotated tag v${VERSION}."
if [[ "${push}" == true ]]; then
  git -C "${PROJECT_ROOT}" push origin "v${VERSION}"
  echo "Pushed v${VERSION}."
else
  echo "Tag was not pushed. Use --push after review."
fi
