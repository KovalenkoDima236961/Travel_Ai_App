"""Quality-aware grounding: the model must be told which records it may trust.

These tests cover the contract between Trip Service's knowledge quality scoring
and the prompt: strong records may be named directly, weak records must be
marked for review, and excluded records must never reach the prompt at all.
"""

from uuid import UUID

from app.schemas.grounding import GroundingContext
from app.schemas.itinerary import GenerateItineraryRequest
from app.services.prompt_builder import build_itinerary_prompt


def _place(
    place_id: str,
    name: str,
    strength: str,
    quality: float,
    review_status: str = "auto",
    **extra: object,
) -> dict:
    return {
        "id": place_id,
        "canonicalName": name,
        "category": "landmark",
        "confidence": 0.9,
        "qualityScore": quality,
        "groundingStrength": strength,
        "reviewStatus": review_status,
        **extra,
    }


def _request(grounding: dict) -> GenerateItineraryRequest:
    return GenerateItineraryRequest(
        tripId=UUID("00000000-0000-0000-0000-000000000001"),
        destination="Rome",
        days=2,
        travelers=2,
        interests=["culture"],
        groundingContext=grounding,
    )


def test_grounding_place_parses_quality_metadata() -> None:
    context = GroundingContext.model_validate(
        {
            "status": "available",
            "places": [
                _place("p1", "Colosseum", "strong", 0.91, review_status="approved"),
                _place("p2", "Testaccio Market", "weak", 0.60, review_status="needs_review"),
            ],
        }
    )

    assert len(context.places) == 2
    assert context.places[0].is_strong
    assert context.places[0].quality_score == 0.91
    assert context.places[1].grounding_strength == "weak"
    assert [place.canonical_name for place in context.strong_places] == ["Colosseum"]
    assert [place.canonical_name for place in context.weak_places] == ["Testaccio Market"]


def test_excluded_places_are_dropped_at_the_schema_boundary() -> None:
    """Defence in depth: Trip Service filters in SQL, the schema filters again."""
    context = GroundingContext.model_validate(
        {
            "status": "available",
            "places": [
                _place("p1", "Colosseum", "strong", 0.91),
                _place("p2", "Rejected Place", "excluded", 0.10, review_status="rejected"),
                _place("p3", "Merged Place", "excluded", 0.0, review_status="merged"),
            ],
        }
    )

    names = [place.canonical_name for place in context.places]
    assert names == ["Colosseum"]
    assert "Rejected Place" not in names
    assert "Merged Place" not in names


def test_prompt_separates_verified_from_unverified_places() -> None:
    request = _request(
        {
            "status": "available",
            "destination": {"canonicalName": "Rome", "countryCode": "IT"},
            "places": [
                _place("p1", "Colosseum", "strong", 0.91, review_status="approved"),
                _place("p2", "Testaccio Market", "weak", 0.60, review_status="needs_review"),
            ],
        }
    )

    prompt = build_itinerary_prompt(request)

    assert "Verified places (safe to name directly)" in prompt
    assert "Unverified places" in prompt
    assert "needsPlaceReview=true" in prompt
    # Quality metadata must be visible so the split is justified to the model.
    assert "quality=0.91" in prompt
    assert "grounding=strong" in prompt
    assert "grounding=weak" in prompt


def test_prompt_omits_the_unverified_section_when_all_records_are_strong() -> None:
    request = _request(
        {
            "status": "available",
            "places": [
                _place("p1", "Colosseum", "strong", 0.91),
                _place("p2", "Pantheon", "strong", 0.88),
            ],
        }
    )

    prompt = build_itinerary_prompt(request)

    assert "Verified places (safe to name directly)" in prompt
    assert "Unverified places" not in prompt


def test_prompt_reports_limited_coverage_and_prefers_generic_activities() -> None:
    request = _request(
        {
            "status": "partial",
            "places": [_place("p1", "Colosseum", "weak", 0.58)],
            "coverage": {
                "placeCount": 3,
                "highQualityPlaceCount": 0,
                "coverageScore": 0.2,
                "status": "limited",
                "warnings": ["Limited verified place data for this destination."],
            },
            "retrievalWarnings": ["Limited verified place data for this destination."],
        }
    )

    prompt = build_itinerary_prompt(request)

    assert "Destination coverage: limited" in prompt
    assert "prefer generic activities" in prompt
    assert "Limited verified place data for this destination." in prompt


def test_prompt_includes_place_warnings_and_opening_hours() -> None:
    request = _request(
        {
            "status": "available",
            "places": [
                _place(
                    "p1",
                    "Testaccio Market",
                    "weak",
                    0.62,
                    openingHoursSummary="Mon,Tue 07:00-15:30",
                    warnings=["Opening hours unknown; confirm before visiting."],
                )
            ],
        }
    )

    prompt = build_itinerary_prompt(request)

    assert "hours=Mon,Tue 07:00-15:30" in prompt
    assert "Opening hours unknown" in prompt


def test_prompt_includes_attribution_when_present() -> None:
    request = _request(
        {
            "status": "available",
            "places": [_place("p1", "Colosseum", "strong", 0.91)],
            "attributions": ["OpenStreetMap contributors"],
        }
    )

    prompt = build_itinerary_prompt(request)

    assert "Data attribution: OpenStreetMap contributors" in prompt


def test_unavailable_grounding_still_instructs_generic_activities() -> None:
    request = _request({"status": "unavailable", "places": []})

    prompt = build_itinerary_prompt(request)

    assert "GROUNDING CONTEXT: unavailable" in prompt
    assert "generic activities rather than invented place names" in prompt


def test_grounding_defaults_are_conservative() -> None:
    """A record without quality metadata must default to weak, never strong."""
    context = GroundingContext.model_validate(
        {
            "status": "available",
            "places": [
                {
                    "id": "legacy",
                    "canonicalName": "Legacy Place",
                    "category": "landmark",
                    "confidence": 0.9,
                }
            ],
        }
    )

    assert context.places[0].grounding_strength == "weak"
    assert context.places[0].quality_score == 0.0
    assert not context.places[0].is_strong
