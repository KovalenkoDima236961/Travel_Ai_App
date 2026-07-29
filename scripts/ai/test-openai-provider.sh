#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SERVICE_DIR="${ROOT_DIR}/services/ai-planning-service"

cd "${SERVICE_DIR}"

python3 -m ruff check \
  app/providers \
  app/config.py \
  app/privacy.py \
  app/observability.py \
  tests/test_openai_provider.py \
  tests/test_privacy.py

python3 -m pytest tests/test_openai_provider.py tests/test_privacy.py

