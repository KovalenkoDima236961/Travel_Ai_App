#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
PROFILE_FILE="${PROJECT_ROOT}/config/feature-flags/alpha.json"
ENV_FILE="${PROJECT_ROOT}/infra/.env.alpha"
RUNTIME_URL=""
ACCESS_TOKEN="${ALPHA_SCOPE_ACCESS_TOKEN:-}"

usage() {
  cat <<'USAGE'
Usage: scripts/release/validate-alpha-scope.sh [--profile PATH] [--env-file PATH] [--runtime-url URL --access-token TOKEN]

Compares the closed-alpha feature flag profile with deployment defaults. When a
runtime URL and bearer token are supplied, it also checks the allowlisted Ops
feature flag endpoint. Secrets are never printed.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; PROFILE_FILE="$2"; shift 2 ;;
    --env-file) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; ENV_FILE="$2"; shift 2 ;;
    --runtime-url) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; RUNTIME_URL="$2"; shift 2 ;;
    --access-token) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; ACCESS_TOKEN="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
done

[[ -f "${PROFILE_FILE}" ]] || { echo "Alpha feature profile not found: ${PROFILE_FILE}" >&2; exit 1; }
[[ -f "${ENV_FILE}" ]] || { echo "Alpha env file not found: ${ENV_FILE}" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "jq is required." >&2; exit 1; }

load_env_file() {
  local line
  while IFS= read -r line || [[ -n "${line}" ]]; do
    case "${line}" in
      ""|\#*) continue ;;
      *=*) export "${line}" ;;
    esac
  done < "$1"
}

load_env_file "${ENV_FILE}"

declare -a ISSUES=()
issue() { ISSUES+=("$1"); }

env_var_for_flag() {
  local key="$1"
  printf 'FEATURE_%s' "$(tr '[:lower:]' '[:upper:]' <<<"${key}")"
}

env_value_for_flag() {
  local key="$1" name
  name="$(env_var_for_flag "${key}")"
  printf '%s' "${!name:-}"
}

expect_env_value() {
  local key="$1" expected="$2" section="$3" owner reason actual
  owner="$(jq -r --arg key "${key}" ".${section}[\$key].owner // empty" "${PROFILE_FILE}")"
  reason="$(jq -r --arg key "${key}" ".${section}[\$key].reason // empty" "${PROFILE_FILE}")"
  actual="$(env_value_for_flag "${key}")"
  if [[ "${actual}" != "${expected}" ]]; then
    issue "$(env_var_for_flag "${key}") expected ${expected}, got '${actual:-unset}' (${owner:-unknown owner}: ${reason:-no reason})"
  fi
}

while IFS= read -r key; do
  expect_env_value "${key}" "true" "requiredEnabled"
done < <(jq -r '.requiredEnabled | keys[]' "${PROFILE_FILE}")

while IFS= read -r key; do
  expect_env_value "${key}" "false" "requiredDisabled"
done < <(jq -r '.requiredDisabled | keys[]' "${PROFILE_FILE}")

while IFS= read -r key; do
  if [[ "$(env_value_for_flag "${key}")" == "true" ]]; then
    issue "dangerous alpha flag enabled: $(env_var_for_flag "${key}")"
  fi
done < <(jq -r '.dangerousIfEnabled[]' "${PROFILE_FILE}")

if [[ -n "${RUNTIME_URL}" ]]; then
  [[ -n "${ACCESS_TOKEN}" ]] || { echo "--runtime-url requires --access-token or ALPHA_SCOPE_ACCESS_TOKEN." >&2; exit 2; }
  command -v curl >/dev/null 2>&1 || { echo "curl is required for runtime checks." >&2; exit 1; }
  runtime_json="$(curl -fsS -H "Authorization: Bearer ${ACCESS_TOKEN}" "${RUNTIME_URL%/}/ops/feature-flags")"
  while IFS= read -r row; do
    key="$(jq -r '.key' <<<"${row}")"
    expected="$(jq -r '.expected' <<<"${row}")"
    actual="$(jq -r --arg key "${key}" '.flags[]? | select(.key == $key) | .value' <<<"${runtime_json}")"
    if [[ "${actual}" != "${expected}" ]]; then
      issue "runtime flag ${key} expected ${expected}, got '${actual:-missing}'"
    fi
  done < <(
    {
      jq -r '.requiredEnabled | keys[] | {key:., expected:"true"} | @json' "${PROFILE_FILE}"
      jq -r '.requiredDisabled | keys[] | {key:., expected:"false"} | @json' "${PROFILE_FILE}"
    }
  )
fi

if (( ${#ISSUES[@]} > 0 )); then
  echo "Alpha scope validation failed:" >&2
  for item in "${ISSUES[@]}"; do echo "- ${item}" >&2; done
  exit 1
fi

echo "Alpha scope validation passed for $(jq -r '.profile' "${PROFILE_FILE}") using ${ENV_FILE}."
