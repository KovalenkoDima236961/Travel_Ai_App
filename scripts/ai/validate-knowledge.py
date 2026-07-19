#!/usr/bin/env python3
"""Validate the small, rights-safe curated knowledge contract without network access."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[2]
KNOWLEDGE_DIR = ROOT / "data" / "travel-knowledge"
ALLOWED_CATEGORIES = {
    "landmark", "museum", "park", "neighborhood", "viewpoint", "market",
    "restaurant", "cafe", "activity", "nature", "transport", "other",
}
ALLOWED_TRUST = {
    "trusted_curated", "trusted_provider", "public_open_data", "app_observed",
    "user_feedback", "mock", "unknown",
}
SOURCE_KEY = re.compile(r"^[a-z0-9][a-z0-9_-]*$")


def fail(message: str) -> None:
    print(f"knowledge validation error: {message}", file=sys.stderr)
    raise SystemExit(1)


def read_json(path: Path) -> Any:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        fail(f"{path.relative_to(ROOT)}: {exc}")


def check_coordinate(value: object, lower: float, upper: float, label: str, path: Path) -> None:
    if value is None:
        return
    if not isinstance(value, int | float) or not lower <= float(value) <= upper:
        fail(f"{path.relative_to(ROOT)}: invalid {label}")


def validate_sources() -> set[str]:
    payload = read_json(KNOWLEDGE_DIR / "sources.json")
    if not isinstance(payload, list) or not payload:
        fail("data/travel-knowledge/sources.json must contain at least one source")
    keys: set[str] = set()
    for source in payload:
        if not isinstance(source, dict):
            fail("source must be an object")
        key = source.get("sourceKey")
        if not isinstance(key, str) or not SOURCE_KEY.fullmatch(key):
            fail("source has invalid sourceKey")
        if key in keys:
            fail(f"duplicate sourceKey {key}")
        if source.get("trustLevel") not in ALLOWED_TRUST:
            fail(f"source {key} has invalid trustLevel")
        if not isinstance(source.get("enabled"), bool):
            fail(f"source {key} must declare enabled")
        keys.add(key)
    return keys


def validate_destination(path: Path, source_keys: set[str]) -> tuple[int, int]:
    payload = read_json(path)
    if not isinstance(payload, dict):
        fail(f"{path.relative_to(ROOT)} must be an object")
    name = payload.get("canonicalName")
    country = payload.get("countryCode")
    places = payload.get("places")
    if not isinstance(name, str) or not name.strip():
        fail(f"{path.relative_to(ROOT)}: canonicalName is required")
    if not isinstance(country, str) or not re.fullmatch(r"[A-Z]{2}", country):
        fail(f"{path.relative_to(ROOT)}: countryCode must be ISO-3166 alpha-2")
    if not isinstance(places, list) or not places:
        fail(f"{path.relative_to(ROOT)}: places must not be empty")
    check_coordinate(payload.get("lat"), -90, 90, "destination latitude", path)
    check_coordinate(payload.get("lng"), -180, 180, "destination longitude", path)
    names: set[str] = set()
    for place in places:
        if not isinstance(place, dict):
            fail(f"{path.relative_to(ROOT)}: place must be an object")
        place_name = place.get("name")
        category = place.get("category")
        confidence = place.get("confidence")
        source = place.get("sourceKey")
        if not isinstance(place_name, str) or not place_name.strip():
            fail(f"{path.relative_to(ROOT)}: place name is required")
        normalized = place_name.strip().casefold()
        if normalized in names:
            fail(f"{path.relative_to(ROOT)}: duplicate place {place_name}")
        names.add(normalized)
        if category not in ALLOWED_CATEGORIES:
            fail(f"{path.relative_to(ROOT)}: {place_name} has invalid category")
        if not isinstance(confidence, int | float) or not 0 <= float(confidence) <= 1:
            fail(f"{path.relative_to(ROOT)}: {place_name} confidence must be between 0 and 1")
        if source not in source_keys:
            fail(f"{path.relative_to(ROOT)}: {place_name} references unknown source {source}")
        check_coordinate(place.get("lat"), -90, 90, "place latitude", path)
        check_coordinate(place.get("lng"), -180, 180, "place longitude", path)
        duration = place.get("typicalDurationMinutes")
        if duration is not None and (not isinstance(duration, int) or not 5 <= duration <= 720):
            fail(f"{path.relative_to(ROOT)}: {place_name} duration must be 5..720 minutes")
    return 1, len(places)


def main() -> None:
    source_keys = validate_sources()
    destinations = places = 0
    for path in sorted((KNOWLEDGE_DIR / "destinations").glob("*.json")):
        count, place_count = validate_destination(path, source_keys)
        destinations += count
        places += place_count
    if destinations < 4:
        fail("at least four curated destinations are required")
    for destination in ("rome", "paris", "vienna", "bratislava"):
        if not (KNOWLEDGE_DIR / "documents" / f"{destination}.en.md").is_file():
            fail(f"missing original planning document for {destination}")
    print(f"validated {destinations} destination file(s), {places} place record(s), and {len(source_keys)} source(s)")


if __name__ == "__main__":
    main()
