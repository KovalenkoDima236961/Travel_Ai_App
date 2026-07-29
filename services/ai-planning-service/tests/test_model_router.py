from uuid import UUID

import pytest
from pydantic import ValidationError

from app.model_router.assignment import deterministic_bucket, rollout_selected
from app.model_router.metadata import routing_metadata_from_request
from app.schemas.itinerary import GenerateItineraryRequest


def _valid_generate_payload() -> dict[str, object]:
    return {
        "tripId": "00000000-0000-0000-0000-000000000001",
        "destination": "Rome",
        "days": 2,
        "budgetCurrency": "EUR",
        "travelers": 1,
        "interests": [],
        "pace": "balanced",
    }


def test_deterministic_bucket_is_stable() -> None:
    first = deterministic_bucket("salt-v1", "user:00000000-0000-0000-0000-000000000001")
    second = deterministic_bucket("salt-v1", "user:00000000-0000-0000-0000-000000000001")

    assert first == second
    assert 0 <= first <= 9999


def test_rollout_selected_uses_percent_bucket_limit() -> None:
    assert rollout_selected(499, 5)
    assert not rollout_selected(500, 5)
    assert rollout_selected(9999, 100)
    assert not rollout_selected(0, 0)


def test_generate_request_accepts_safe_routing_metadata() -> None:
    assignment_id = UUID("00000000-0000-0000-0000-000000000010")
    payload = _valid_generate_payload() | {
        "deploymentKey": "grounded-baseline",
        "requestAssignmentId": str(assignment_id),
        "inferenceMode": "primary",
    }

    request = GenerateItineraryRequest.model_validate(payload)
    metadata = routing_metadata_from_request(request)

    assert metadata["deploymentKey"] == "grounded-baseline"
    assert metadata["requestAssignmentId"] == str(assignment_id)
    assert metadata["inferenceMode"] == "primary"


def test_generate_request_rejects_path_like_deployment_key() -> None:
    payload = _valid_generate_payload() | {"deploymentKey": "../private/adapter"}

    with pytest.raises(ValidationError):
        GenerateItineraryRequest.model_validate(payload)
