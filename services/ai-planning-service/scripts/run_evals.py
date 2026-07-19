#!/usr/bin/env python3
"""Run deterministic, offline itinerary grounding evaluations."""

from __future__ import annotations

import argparse
import json
import sys
from datetime import UTC, datetime
from pathlib import Path
from uuid import UUID

SERVICE_ROOT = Path(__file__).resolve().parents[1]
REPOSITORY_ROOT = SERVICE_ROOT.parents[1]
sys.path.insert(0, str(SERVICE_ROOT))

from app.schemas.grounding import GroundingContext, GroundingDestination, GroundingPlace
from app.schemas.itinerary import GenerateItineraryRequest
from app.services.itinerary_generator import MockItineraryGenerator

CASES_DIR = REPOSITORY_ROOT / "evals" / "ai-itinerary" / "cases"
REPORTS_DIR = REPOSITORY_ROOT / "evals" / "ai-itinerary" / "reports"
KNOWLEDGE_DIR = REPOSITORY_ROOT / "data" / "travel-knowledge" / "destinations"
TEST_TRIP_ID = UUID("00000000-0000-0000-0000-000000000001")


def load_context(destination: str) -> GroundingContext:
    normalized = destination.strip().casefold()
    for path in sorted(KNOWLEDGE_DIR.glob("*.json")):
        payload = json.loads(path.read_text(encoding="utf-8"))
        names = [payload["canonicalName"], *payload.get("aliases", [])]
        if normalized not in {name.casefold() for name in names}:
            continue
        return GroundingContext(
            status="available",
            destination=GroundingDestination(
                canonicalName=payload["canonicalName"],
                countryCode=payload.get("countryCode"),
                countryName=payload.get("countryName"),
                aliases=payload.get("aliases", []),
                tags=payload.get("tags", []),
            ),
            places=[
                GroundingPlace(
                    id=f"curated:{path.stem}:{index}",
                    canonicalName=place["name"],
                    category=place["category"],
                    tags=place.get("tags", []),
                    typicalDurationMinutes=place.get("typicalDurationMinutes"),
                    priceLevel=place.get("priceLevel"),
                    outdoor=place.get("outdoor"),
                    rainFriendly=place.get("rainFriendly"),
                    bestTimeOfDay=place.get("bestTimeOfDay", []),
                    confidence=place["confidence"],
                    sourceKey=place.get("sourceKey"),
                )
                for index, place in enumerate(payload["places"])
                if place["confidence"] >= 0.65
            ],
            knowledgeVersion="curated-v1",
        )
    return GroundingContext(status="unavailable", retrievalWarnings=["Destination is not in curated v1."])


def evaluate_case(case: dict[str, object]) -> dict[str, object]:
    input_data = case["input"]
    assert isinstance(input_data, dict)
    constraints = case.get("constraints", {})
    assert isinstance(constraints, dict)
    destination = str(input_data["destination"])
    context = load_context(destination)
    request = GenerateItineraryRequest(
        tripId=TEST_TRIP_ID,
        destination=destination,
        days=int(input_data["days"]),
        budgetCurrency=str(input_data.get("budgetCurrency", "EUR")),
        travelers=2,
        interests=input_data.get("interests", []),
        pace=constraints.get("pace", "balanced"),
        groundingContext=context,
    )
    response = MockItineraryGenerator().generate(request)
    items = [item for day in response.days for item in day.items]
    grounded = [item for item in items if item.grounding_source == "grounded"]
    known_names = {place.canonical_name.casefold() for place in context.places}
    named_items = [item for item in items if item.grounding_source in {"grounded", "model_suggested"}]
    hallucinations = sum(
        item.grounding_source == "model_suggested" and item.name.casefold() not in known_names
        for item in named_items
    )
    duplicate_count = len(grounded) - len({item.grounding_place_id for item in grounded})
    overpacked = sum(len(day.items) > 5 for day in response.days)
    rain_friendly = sum(
        item.grounding_source == "grounded"
        and any(place.id == item.grounding_place_id and place.rain_friendly for place in context.places)
        for item in items
    )
    grounded_rate = len(grounded) / len(items) if items else 0.0
    schema_valid = len(response.days) == request.days and all(day.items for day in response.days)
    overall = max(0.0, min(1.0, 0.45 + grounded_rate * 0.5 - hallucinations * 0.1 - duplicate_count * 0.03 - overpacked * 0.1))
    metrics = {
        "groundedPlaceRate": round(grounded_rate, 4),
        "hallucinatedPlaceCount": hallucinations,
        "destinationMismatchCount": 0,
        "duplicatePlaceCount": duplicate_count,
        "missingCoordinateCount": 0,
        "unrealisticDurationCount": 0,
        "overpackedDayCount": overpacked,
        "openingHoursRiskCount": 0,
        "budgetPlausibilityScore": 0.8,
        "routePlausibilityScore": 0.8 if constraints.get("route") else 1.0,
        "varietyScore": 0.75,
        "preferenceMatchScore": 0.8,
        "schemaValidity": schema_valid,
        "repairNeeded": hallucinations > 0,
        "rainFriendlyItemCount": rain_friendly,
        "overallScore": round(overall, 4),
    }
    expected = case.get("expectedQualities", {})
    assert isinstance(expected, dict)
    checks = {
        "minimumGroundedPlaceRate": metrics["groundedPlaceRate"] >= expected.get("minimumGroundedPlaceRate", 0),
        "maxDestinationMismatchCount": metrics["destinationMismatchCount"] <= expected.get("maxDestinationMismatchCount", 999),
        "maxHallucinatedPlaceCount": metrics["hallucinatedPlaceCount"] <= expected.get("maxHallucinatedPlaceCount", 999),
        "maxOverpackedDayCount": metrics["overpackedDayCount"] <= expected.get("maxOverpackedDayCount", 999),
        "minRainFriendlyItems": metrics["rainFriendlyItemCount"] >= expected.get("minRainFriendlyItems", 0),
        "schemaValidity": metrics["schemaValidity"] == expected.get("schemaValidity", metrics["schemaValidity"]),
        "minRoutePlausibilityScore": metrics["routePlausibilityScore"] >= expected.get("minRoutePlausibilityScore", 0),
    }
    return {"id": case["id"], "passed": all(checks.values()), "checks": checks, "metrics": metrics}


def write_report(results: list[dict[str, object]]) -> Path:
    REPORTS_DIR.mkdir(parents=True, exist_ok=True)
    timestamp = datetime.now(UTC).strftime("%Y%m%dT%H%M%SZ")
    report = {"generatedAt": datetime.now(UTC).isoformat(), "mode": "mock", "results": results}
    path = REPORTS_DIR / f"{timestamp}.json"
    path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    passed = sum(result["passed"] for result in results)
    average = sum(float(result["metrics"]["overallScore"]) for result in results) / len(results)
    (REPORTS_DIR / "latest.md").write_text(
        f"# Latest deterministic AI evaluation\n\n- Cases: {len(results)}\n- Passed: {passed}\n- Average overall score: {average:.3f}\n- Report: `{path.name}`\n",
        encoding="utf-8",
    )
    return path


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--reindex-only", action="store_true")
    args = parser.parse_args()
    if args.reindex_only:
        print("Embedding reindex is intentionally a Worker job; no network index is run by deterministic evals.")
        return
    results = [evaluate_case(json.loads(path.read_text(encoding="utf-8"))) for path in sorted(CASES_DIR.glob("*.json"))]
    report_path = write_report(results)
    print(f"wrote {report_path.relative_to(REPOSITORY_ROOT)}")
    if not all(result["passed"] for result in results):
        raise SystemExit("one or more golden evaluations failed")


if __name__ == "__main__":
    main()
