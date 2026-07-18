#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"

usage() {
  echo "Usage: scripts/release/prepare-release.sh VERSION [--skip-contract-check]" >&2
}

[[ $# -ge 1 ]] || { usage; exit 2; }
NEW_VERSION="$1"; shift
skip_contract=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-contract-check) skip_contract=true; shift ;;
    *) usage; exit 2 ;;
  esac
done
[[ "${NEW_VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$ ]] || { echo "Invalid semantic version: ${NEW_VERSION}" >&2; exit 1; }

printf '%s\n' "${NEW_VERSION}" > "${PROJECT_ROOT}/VERSION"
date_utc="$(date -u +%F)"
changelog="${PROJECT_ROOT}/CHANGELOG.md"
if ! grep -q "^## \[${NEW_VERSION}\]" "${changelog}"; then
  tmp="$(mktemp)"
  awk -v version="${NEW_VERSION}" -v date="${date_utc}" '
    /^## \[Unreleased\]/ { print; print ""; next }
    /^### Added$/ && !inserted { print "## [" version "] - " date; print ""; inserted=1 }
    { print }
  ' "${changelog}" > "${tmp}"
  mv "${tmp}" "${changelog}"
fi

"${SCRIPT_DIR}/check-changelog.sh"
if [[ "${skip_contract}" == false ]]; then
  "${PROJECT_ROOT}/scripts/contracts/validate-openapi.sh"
  "${PROJECT_ROOT}/scripts/contracts/check-generated.sh"
fi
VERSION="${NEW_VERSION}" "${SCRIPT_DIR}/generate-release-notes.sh"

echo "Prepared ${NEW_VERSION}. Next: review CHANGELOG.md, run ./scripts/release/check-release.sh ci, then ./scripts/release/tag-release.sh ${NEW_VERSION}."
