#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_DIR="$(cd -- "${SCRIPT_DIR}/../services/ai-planning-service" && pwd)"

cd "${SERVICE_DIR}"
python3 -m ruff check app tests
if [[ "${PYTHON_TEST_COVERAGE:-false}" == "true" ]]; then
  python3 -m pytest --cov=app --cov-report=term-missing --cov-report=xml
else
  python3 -m pytest
fi

echo "PASS Python AI service lint and tests"
