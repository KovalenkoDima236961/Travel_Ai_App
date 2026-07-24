#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${TRIP_SERVICE_URL:-http://localhost:8081}"
AUTH_HEADER=()
if [[ -n "${TRIP_SERVICE_TOKEN:-}" ]]; then
  AUTH_HEADER=(-H "Authorization: Bearer ${TRIP_SERVICE_TOKEN}")
fi

payload="$(curl -fsS "${AUTH_HEADER[@]}" "${BASE_URL%/}/ops/ai/fine-tuning/readiness")"

READINESS_PAYLOAD="$payload" python3 - "${ALLOW_NOT_READY:-0}" <<'PY'
import json
import os
import sys

allow_not_ready = sys.argv[1] == "1"
payload = json.loads(os.environ["READINESS_PAYLOAD"])
print(json.dumps(payload, indent=2))

if not payload.get("ready") and not allow_not_ready:
    blockers = payload.get("blockers") or ["readiness check returned ready=false"]
    print("Fine-tuning readiness blocked:", file=sys.stderr)
    for blocker in blockers:
        print(f"- {blocker}", file=sys.stderr)
    sys.exit(1)
PY
