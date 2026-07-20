#!/usr/bin/env bash
# Assert that CI cannot reach a real travel data provider.
#
# Knowledge provider adapters that make network calls belong in External
# Integrations Service, behind its quota and cache guards. This check fails the
# build if the knowledge module grows a direct HTTP client or starts reading
# provider credentials -- the mistakes that would silently turn a deterministic
# test suite into one depending on a third-party API.
#
# Comments and test fixtures are excluded: this inspects code, and the module
# deliberately *documents* that it holds no credentials.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
KNOWLEDGE_DIR="${ROOT_DIR}/services/trip-service/internal/knowledge"

if [ ! -d "${KNOWLEDGE_DIR}" ]; then
  echo "FAIL: knowledge module not found at ${KNOWLEDGE_DIR}" >&2
  exit 1
fi

status=0
code="$(mktemp)"
trap 'rm -f "${code}"' EXIT

# Strip comments so prose about secrets does not trip the credential check.
while IFS= read -r file; do
  sed -e 's|//.*||' "${file}" | grep -v '^[[:space:]]*$' | sed "s|^|${file}:|" >> "${code}"
done < <(find "${KNOWLEDGE_DIR}" -name '*.go' -not -name '*_test.go' | sort)

if [ ! -s "${code}" ]; then
  echo "FAIL: no Go sources found in the knowledge module" >&2
  exit 1
fi

if grep -q '"net/http"' "${code}"; then
  echo "FAIL: the knowledge module must not perform HTTP calls directly." >&2
  echo "      Real provider adapters belong in External Integrations Service." >&2
  grep '"net/http"' "${code}" >&2
  status=1
fi

if grep -qiE '(apikey|api_key|clientsecret|client_secret|accesstoken|bearer )' "${code}"; then
  echo "FAIL: the knowledge module must not handle provider credentials." >&2
  grep -inE '(apikey|api_key|clientsecret|client_secret|accesstoken|bearer )' "${code}" >&2
  status=1
fi

# The mock provider must remain the default so an unconfigured environment
# never attempts real provider traffic.
for selection in \
  "${ROOT_DIR}/services/worker-service/internal/knowledge/provider_runner.go" \
  "${ROOT_DIR}/services/trip-service/internal/app/di.go"; do
  if [ -f "${selection}" ] && ! grep -q 'ProviderMock' "${selection}"; then
    echo "FAIL: ${selection} does not reference the mock provider default." >&2
    status=1
  fi
done

if [ "${status}" -eq 0 ]; then
  echo "Provider isolation OK: the knowledge module makes no direct provider calls,"
  echo "handles no credentials, and defaults to the mock provider."
fi
exit "${status}"
