#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-}"
DAYS="${2:-14}"
if [[ -z "${ROOT}" || ! -d "${ROOT}" ]]; then
  echo "Usage: scripts/ai/cleanup-training-checkpoints.sh <artifact-root> [older-than-days]" >&2
  exit 2
fi

find "${ROOT}" -type d -name 'checkpoint-*' -mtime "+${DAYS}" -print
echo "Dry run only. Re-run the printed paths through an explicit removal command after review."
