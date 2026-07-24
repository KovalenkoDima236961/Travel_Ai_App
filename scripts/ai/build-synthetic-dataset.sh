#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="${1:-data/ai-training/manual/itinerary-generation}"
mkdir -p "$OUT_DIR"

python3 - "$OUT_DIR" <<'PY'
import json
import pathlib
import sys

out_dir = pathlib.Path(sys.argv[1])
target = out_dir / "synthetic-lisbon-walkable.json"
if target.exists():
    print(f"{target} already exists.")
    raise SystemExit(0)

payload = {
    "id": "synthetic-lisbon-walkable",
    "taskType": "itinerary_generation",
    "language": "en",
    "input": {
        "destinationName": "Lisbon",
        "days": 2,
        "budgetCurrency": "EUR",
        "interests": ["food", "viewpoints", "walkable neighborhoods"],
        "pace": "balanced",
    },
    "grounding": {
        "groundingIds": ["places/time-out-market", "places/miradouro-santa-catarina"],
        "confidence": 0.95,
    },
    "output": {
        "days": [
            {
                "day": 1,
                "items": [
                    {"name": "Baixa orientation walk", "area": "Baixa", "durationMinutes": 75},
                    {"name": "Time Out Market lunch", "area": "Cais do Sodre", "durationMinutes": 90},
                    {"name": "Miradouro de Santa Catarina", "area": "Bica", "durationMinutes": 45},
                ],
            },
            {
                "day": 2,
                "items": [
                    {"name": "Alfama morning walk", "area": "Alfama", "durationMinutes": 120},
                    {"name": "Tile museum visit", "area": "Xabregas", "durationMinutes": 90},
                    {"name": "Riverside dinner", "area": "Cais do Sodre", "durationMinutes": 90},
                ],
            },
        ]
    },
    "labels": {
        "matchesPreferences": True,
        "budgetPlausible": True,
        "userAccepted": True,
        "groundedPlaceRate": 1,
    },
    "provenance": {
        "source": "synthetic_manual_seed",
        "licenseStatus": "project_owned",
        "copiesProviderText": False,
        "reviewed": True,
    },
}
target.write_text(json.dumps(payload, indent=2) + "\n")
print(f"Wrote {target}.")
PY

scripts/ai/validate-training-examples.sh "$OUT_DIR"
