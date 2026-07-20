#!/usr/bin/env python3
"""Validate knowledge source provenance before ingestion is trusted.

This guards the licensing rules in docs/ai/trusted-travel-data-providers.md at
the file level, where curated sources are defined. The equivalent runtime rule
lives in EnsureProviderSource, which refuses to register a provider source
without a license.

Checks:
  * every source declares a non-empty source_key
  * every non-curated, non-mock source declares license and attribution
  * every source_type and trust_level is one of the supported values
  * no place references an undeclared source
  * no place claims high confidence from an unknown-trust source
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
KNOWLEDGE_DIR = ROOT / "data" / "travel-knowledge"

# Kept in sync with the CHECK constraints in migration 000042.
SUPPORTED_SOURCE_TYPES = {
    "manual_curated",
    "provider_place",
    "open_data",
    "user_approved_match",
    "user_feedback",
    "mock_test_data",
}
SUPPORTED_TRUST_LEVELS = {
    "trusted_curated",
    "trusted_provider",
    "public_open_data",
    "app_observed",
    "user_feedback",
    "mock",
    "unknown",
}
# Sources whose content is original to this project and therefore need no
# third-party attribution.
SELF_OWNED_TRUST_LEVELS = {"trusted_curated", "mock"}

# A record from an unknown-trust source must not assert high confidence; the
# quality model caps such records, and the curated files must agree.
UNKNOWN_TRUST_MAX_CONFIDENCE = 0.5


def fail(errors: list[str]) -> None:
    print("Knowledge source validation FAILED:\n", file=sys.stderr)
    for error in errors:
        print(f"  - {error}", file=sys.stderr)
    sys.exit(1)


def main() -> int:
    errors: list[str] = []

    sources_path = KNOWLEDGE_DIR / "sources.json"
    if not sources_path.exists():
        fail([f"missing {sources_path.relative_to(ROOT)}"])

    sources = json.loads(sources_path.read_text(encoding="utf-8"))
    if not isinstance(sources, list) or not sources:
        fail(["sources.json must be a non-empty array"])

    by_key: dict[str, dict] = {}
    for index, source in enumerate(sources):
        key = str(source.get("sourceKey", "")).strip()
        label = key or f"sources[{index}]"

        if not key:
            errors.append(f"{label}: sourceKey is required")
        elif key in by_key:
            errors.append(f"{label}: duplicate sourceKey")
        else:
            by_key[key] = source

        source_type = source.get("sourceType")
        if source_type not in SUPPORTED_SOURCE_TYPES:
            errors.append(f"{label}: unsupported sourceType {source_type!r}")

        trust_level = source.get("trustLevel")
        if trust_level not in SUPPORTED_TRUST_LEVELS:
            errors.append(f"{label}: unsupported trustLevel {trust_level!r}")

        if not str(source.get("displayName", "")).strip():
            errors.append(f"{label}: displayName is required")

        # Licensing: anything not original to this project must say where it
        # came from and under what terms.
        if trust_level not in SELF_OWNED_TRUST_LEVELS:
            if not str(source.get("licenseName", "")).strip():
                errors.append(f"{label}: licenseName is required for a third-party source")
            if not str(source.get("attribution", "")).strip():
                errors.append(f"{label}: attribution is required for a third-party source")

    destinations_dir = KNOWLEDGE_DIR / "destinations"
    place_count = 0
    for path in sorted(destinations_dir.glob("*.json")):
        destination = json.loads(path.read_text(encoding="utf-8"))
        name = destination.get("canonicalName", path.stem)
        for place in destination.get("places", []):
            place_count += 1
            place_name = place.get("name", "<unnamed>")
            label = f"{name}/{place_name}"

            source_key = str(place.get("sourceKey", "")).strip()
            if not source_key:
                errors.append(f"{label}: sourceKey is required")
                continue
            source = by_key.get(source_key)
            if source is None:
                errors.append(f"{label}: references undeclared source {source_key!r}")
                continue
            if not source.get("enabled", False):
                errors.append(f"{label}: references disabled source {source_key!r}")

            trust_level = source.get("trustLevel")
            confidence = place.get("confidence")
            if trust_level == "unknown" and isinstance(confidence, (int, float)):
                if confidence > UNKNOWN_TRUST_MAX_CONFIDENCE:
                    errors.append(
                        f"{label}: confidence {confidence} is too high for an "
                        f"unknown-trust source (max {UNKNOWN_TRUST_MAX_CONFIDENCE})"
                    )

            if trust_level not in SELF_OWNED_TRUST_LEVELS:
                if not str(place.get("attribution", "")).strip():
                    errors.append(f"{label}: attribution is required for a third-party record")

    if errors:
        fail(errors)

    print(
        f"Knowledge source validation OK: {len(by_key)} source(s), "
        f"{place_count} place(s) across {len(list(destinations_dir.glob('*.json')))} destination(s)."
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
