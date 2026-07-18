#!/usr/bin/env bash
# Source this file to export release metadata, or execute it to print the values.
set -euo pipefail

RELEASE_SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${RELEASE_SCRIPT_DIR}/../.." && pwd)"
VERSION_FILE="${VERSION_FILE:-${PROJECT_ROOT}/VERSION}"

[[ -f "${VERSION_FILE}" ]] || { echo "VERSION file not found: ${VERSION_FILE}" >&2; return 1 2>/dev/null || exit 1; }

VERSION="${VERSION:-$(tr -d '[:space:]' < "${VERSION_FILE}")}"
SEMVER_REGEX='^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$'
[[ "${VERSION}" =~ ${SEMVER_REGEX} ]] || { echo "VERSION must be a semantic version, got: ${VERSION}" >&2; return 1 2>/dev/null || exit 1; }

if [[ -z "${GIT_SHA:-}" ]]; then
  if git -C "${PROJECT_ROOT}" rev-parse --verify HEAD >/dev/null 2>&1; then
    GIT_SHA="$(git -C "${PROJECT_ROOT}" rev-parse HEAD)"
  else
    GIT_SHA="unknown"
  fi
fi
SHORT_SHA="${SHORT_SHA:-${GIT_SHA:0:12}}"
[[ -n "${SHORT_SHA}" ]] || SHORT_SHA="unknown"
BUILD_TIME="${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
IMAGE_TAG_VERSION="${VERSION}"
IMAGE_TAG_SHA="${VERSION}-${SHORT_SHA}"

export VERSION GIT_SHA SHORT_SHA BUILD_TIME IMAGE_TAG_VERSION IMAGE_TAG_SHA

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  printf 'VERSION=%s\nGIT_SHA=%s\nSHORT_SHA=%s\nBUILD_TIME=%s\nIMAGE_TAG_VERSION=%s\nIMAGE_TAG_SHA=%s\n' \
    "${VERSION}" "${GIT_SHA}" "${SHORT_SHA}" "${BUILD_TIME}" "${IMAGE_TAG_VERSION}" "${IMAGE_TAG_SHA}"
fi
