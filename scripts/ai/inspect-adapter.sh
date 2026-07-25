#!/usr/bin/env bash
set -euo pipefail

ADAPTER_DIR="${1:-}"
if [[ -z "${ADAPTER_DIR}" ]]; then
  echo "Usage: scripts/ai/inspect-adapter.sh <adapter-dir>" >&2
  exit 2
fi
if [[ ! -d "${ADAPTER_DIR}" ]]; then
  echo "Adapter directory not found: ${ADAPTER_DIR}" >&2
  exit 1
fi
if [[ ! -f "${ADAPTER_DIR}/adapter_config.json" ]]; then
  echo "Missing adapter_config.json" >&2
  exit 1
fi

python3 - "${ADAPTER_DIR}" <<'PY'
import hashlib
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1]).resolve()
digest = hashlib.sha256()
files = [path for path in sorted(root.rglob("*")) if path.is_file()]
for path in files:
    rel = path.relative_to(root).as_posix()
    digest.update(rel.encode())
    digest.update(path.read_bytes())
config = json.loads((root / "adapter_config.json").read_text())
print(json.dumps({
    "adapterDir": str(root),
    "fileCount": len(files),
    "sha256": digest.hexdigest(),
    "peftType": config.get("peft_type"),
    "baseModelNameOrPath": config.get("base_model_name_or_path"),
}, indent=2, sort_keys=True))
PY
