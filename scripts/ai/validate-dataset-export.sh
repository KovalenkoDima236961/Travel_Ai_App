#!/usr/bin/env bash
set -euo pipefail

EXPORT_DIR="${1:-}"

if [[ -z "$EXPORT_DIR" ]]; then
  echo "Usage: scripts/ai/validate-dataset-export.sh <export-directory>" >&2
  exit 2
fi

python3 - "$EXPORT_DIR" <<'PY'
import json
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])
required = ["train.jsonl", "validation.jsonl", "test.jsonl", "holdout.jsonl", "manifest.json", "checksums.txt", "README.md"]
email = re.compile(r"\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b", re.I)
phone = re.compile(r"\b(?:\+?\d{1,3}[\s.-]?)?(?:\(?\d{3}\)?[\s.-]?)\d{3}[\s.-]?\d{4}\b")
private_words = re.compile(r"(password|api[_-]?key|access[_-]?token|refresh[_-]?token|publicsharetoken|receiptocr|system_prompt|chain_of_thought)", re.I)
errors = []

if not root.is_dir():
    errors.append(f"{root} is not a directory")

for name in required:
    path = root / name
    if not path.exists():
        errors.append(f"missing {name}")
        continue
    text = path.read_text()
    if email.search(text):
        errors.append(f"{name} contains email-like text")
    if phone.search(text):
        errors.append(f"{name} contains phone-like text")
    if private_words.search(text):
        errors.append(f"{name} contains private-data marker")

counts = {}
for name in ["train.jsonl", "validation.jsonl", "test.jsonl", "holdout.jsonl"]:
    path = root / name
    if not path.exists():
        continue
    count = 0
    for line_number, line in enumerate(path.read_text().splitlines(), 1):
        if not line.strip():
            continue
        count += 1
        try:
            payload = json.loads(line)
        except Exception as exc:
            errors.append(f"{name}:{line_number}: invalid JSONL: {exc}")
            continue
        for key in ["id", "task", "language", "schemaVersion", "input", "output", "labels", "metadata"]:
            if key not in payload:
                errors.append(f"{name}:{line_number}: missing {key}")
    counts[name] = count

manifest_path = root / "manifest.json"
if manifest_path.exists():
    try:
        manifest = json.loads(manifest_path.read_text())
        if manifest.get("schemaVersion") != "ai_dataset_v1":
            errors.append("manifest schemaVersion is not ai_dataset_v1")
        if not isinstance(manifest.get("splitCounts"), dict):
            errors.append("manifest missing splitCounts")
    except Exception as exc:
        errors.append(f"manifest.json invalid: {exc}")

if errors:
    print("AI dataset export validation failed:", file=sys.stderr)
    for err in errors:
        print(f"- {err}", file=sys.stderr)
    sys.exit(1)

print(f"Validated AI dataset export {root}: {counts}")
PY
