#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
AI_DIR="$ROOT_DIR/services/ai-planning-service"
OUTPUT="$ROOT_DIR/docs/api/openapi/ai-planning-service.openapi.yaml"

(
  cd "$AI_DIR"
  PYTHONPATH=. python3 -c '
import json
from app.main import app

schema = app.openapi()
for path in schema.get("paths", {}).values():
    for method, operation in path.items():
        if method not in {"get", "post", "put", "patch", "delete"}:
            continue
        operation.setdefault("tags", ["AI planning"])
        operation.setdefault("responses", {}).setdefault(
            "default",
            {
                "description": "User-safe AI planning failure.",
                "content": {
                    "application/json": {
                        "schema": {"$ref": "#/components/schemas/ErrorResponse"}
                    }
                },
            },
        )
schema.setdefault("components", {}).setdefault("schemas", {}).setdefault(
    "ErrorResponse",
    {
        "type": "object",
        "properties": {
            "error": {"type": "string"},
            "message": {"type": "string"},
            "requestId": {"type": "string"},
        },
        "required": ["error"],
    },
)
print(json.dumps(schema, indent=2, sort_keys=True))
'
) > "$OUTPUT"

echo "Exported $OUTPUT from FastAPI."
