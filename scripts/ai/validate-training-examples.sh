#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-data/ai-training/manual}"

python3 - "$ROOT" <<'PY'
import json
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])
required = {"id", "taskType", "language", "input", "output", "labels", "provenance"}
allowed_tasks = {
    "itinerary_generation",
    "day_regeneration",
    "item_regeneration",
    "place_replacement",
    "policy_repair",
    "budget_optimization",
    "route_alternatives",
    "checklist_generation",
    "copilot_response",
    "recap_generation",
}
sensitive_key = re.compile(
    r"(email|phone|password|token|api[_-]?key|secret|calendar|ocr|receipt|comment|private[_-]?notes|system[_-]?prompt|chain[_-]?of[_-]?thought)",
    re.I,
)
email = re.compile(r"\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b", re.I)
phone = re.compile(r"\b(?:\+?\d{1,3}[\s.-]?)?(?:\(?\d{3}\)?[\s.-]?)\d{3}[\s.-]?\d{4}\b")
token = re.compile(r"\b[A-Za-z0-9_-]{32,}\b")

errors = []
files = sorted(root.glob("**/*.json")) if root.exists() else []

def walk(value, path):
    if isinstance(value, dict):
        for key, child in value.items():
            child_path = f"{path}.{key}" if path else key
            if sensitive_key.search(key):
                errors.append(f"{current}: sensitive key {child_path}")
            walk(child, child_path)
    elif isinstance(value, list):
        for index, child in enumerate(value):
            walk(child, f"{path}[{index}]")
    elif isinstance(value, str):
        if email.search(value):
            errors.append(f"{current}: email-like value at {path}")
        if phone.search(value):
            errors.append(f"{current}: phone-like value at {path}")
        if token.search(value) and re.search(r"(token|secret|key)", value, re.I):
            errors.append(f"{current}: token-like value at {path}")

for path in files:
    current = path
    try:
        payload = json.loads(path.read_text())
    except Exception as exc:
        errors.append(f"{path}: invalid JSON: {exc}")
        continue
    missing = required - payload.keys()
    if missing:
        errors.append(f"{path}: missing required fields: {', '.join(sorted(missing))}")
    if payload.get("taskType") not in allowed_tasks:
        errors.append(f"{path}: unsupported taskType {payload.get('taskType')!r}")
    if not isinstance(payload.get("input"), dict) or not isinstance(payload.get("output"), dict):
        errors.append(f"{path}: input and output must be objects")
    provenance = payload.get("provenance")
    if not isinstance(provenance, dict):
        errors.append(f"{path}: provenance must be an object")
    else:
        license_status = provenance.get("licenseStatus")
        copies_text = provenance.get("copiesProviderText")
        if license_status in {None, "unknown", "incompatible", "not_allowed"} and copies_text is True:
            errors.append(f"{path}: provider text copy is not allowed for licenseStatus={license_status!r}")
    walk(payload, "")

if errors:
    print("AI training example validation failed:", file=sys.stderr)
    for err in errors:
        print(f"- {err}", file=sys.stderr)
    sys.exit(1)

print(f"Validated {len(files)} AI training example file(s) under {root}.")
PY
