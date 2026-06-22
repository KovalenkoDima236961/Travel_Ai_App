import json
from copy import deepcopy

from fastapi.testclient import TestClient

from app.config import Settings, get_settings
from app.main import create_app
from app.services.destination_knowledge import FileDestinationKnowledgeProvider
from app.services.generator_factory import get_destination_knowledge_provider
from app.services.itinerary_generator import MockItineraryGenerator

VALID_PAYLOAD = {
    "tripId": "550e8400-e29b-41d4-a716-446655440000",
    "destination": "Rome",
    "startDate": "2026-08-10",
    "days": 4,
    "budgetAmount": 600,
    "budgetCurrency": "EUR",
    "travelers": 2,
    "interests": ["food", "history", "hidden_gems"],
    "pace": "balanced",
}


def _write_contexts(data_dir) -> None:
    (data_dir / "rome.json").write_text(
        json.dumps(
            {
                "destination": "Rome",
                "aliases": ["roma"],
                "localTips": ["Visit popular attractions early."],
                "hiddenGems": ["Orange Garden"],
                "foodTips": ["Try carbonara."],
                "avoid": ["Avoid overloading one day."],
                "transportTips": ["Group nearby attractions together."],
                "budgetTips": ["Use free viewpoints."],
            }
        ),
        encoding="utf-8",
    )
    (data_dir / "paris.json").write_text(
        json.dumps({"destination": "Paris", "aliases": ["paris, france"]}),
        encoding="utf-8",
    )


def _client(settings: Settings, provider: FileDestinationKnowledgeProvider | None) -> TestClient:
    app = create_app()
    app.state.settings = settings
    app.state.destination_knowledge_provider = provider
    app.state.itinerary_generator = MockItineraryGenerator()
    return TestClient(app)


def test_get_destination_context_returns_list_response(tmp_path) -> None:
    _write_contexts(tmp_path)
    client = _client(
        Settings(destination_context_enabled=True),
        FileDestinationKnowledgeProvider(tmp_path),
    )

    response = client.get("/destination-context")

    assert response.status_code == 200
    assert response.json() == {
        "items": [
            {"destination": "Paris", "aliases": ["paris, france"], "source": "file"},
            {"destination": "Rome", "aliases": ["roma"], "source": "file"},
        ]
    }


def test_get_destination_context_by_destination_returns_context_when_found(tmp_path) -> None:
    _write_contexts(tmp_path)
    client = _client(
        Settings(destination_context_enabled=True),
        FileDestinationKnowledgeProvider(tmp_path),
    )

    response = client.get("/destination-context/roma")

    assert response.status_code == 200
    body = response.json()
    assert body["destination"] == "Rome"
    assert body["aliases"] == ["roma"]
    assert body["localTips"] == ["Visit popular attractions early."]


def test_get_destination_context_by_destination_returns_404_when_missing(tmp_path) -> None:
    _write_contexts(tmp_path)
    client = _client(
        Settings(destination_context_enabled=True),
        FileDestinationKnowledgeProvider(tmp_path),
    )

    response = client.get("/destination-context/madrid")

    assert response.status_code == 404
    assert response.json() == {"error": "Destination context not found"}


def test_preview_prompt_returns_prompt_with_context_when_found(tmp_path) -> None:
    _write_contexts(tmp_path)
    client = _client(
        Settings(destination_context_enabled=True),
        FileDestinationKnowledgeProvider(tmp_path),
    )

    response = client.post("/destination-context/rome/preview-prompt", json=VALID_PAYLOAD)

    assert response.status_code == 200
    body = response.json()
    assert body["destinationContextFound"] is True
    assert body["destinationContext"]["destination"] == "Rome"
    assert "DESTINATION CONTEXT:" in body["prompt"]
    assert "Orange Garden" in body["prompt"]
    assert "- Destination: Rome" in body["prompt"]


def test_preview_prompt_returns_prompt_without_context_when_missing(tmp_path) -> None:
    _write_contexts(tmp_path)
    client = _client(
        Settings(destination_context_enabled=True),
        FileDestinationKnowledgeProvider(tmp_path),
    )

    response = client.post("/destination-context/madrid/preview-prompt", json=VALID_PAYLOAD)

    assert response.status_code == 200
    body = response.json()
    assert body["destinationContextFound"] is False
    assert body["destinationContext"] is None
    assert "DESTINATION CONTEXT:" not in body["prompt"]
    assert "- Destination: Rome" in body["prompt"]


def test_preview_prompt_uses_path_destination_only_for_context_lookup(tmp_path) -> None:
    _write_contexts(tmp_path)
    client = _client(
        Settings(destination_context_enabled=True),
        FileDestinationKnowledgeProvider(tmp_path),
    )
    payload = deepcopy(VALID_PAYLOAD)
    payload["destination"] = "Paris"

    response = client.post("/destination-context/rome/preview-prompt", json=payload)

    assert response.status_code == 200
    body = response.json()
    assert body["destinationContextFound"] is True
    assert body["destinationContext"]["destination"] == "Rome"
    assert "- Destination: Paris" in body["prompt"]
    assert "- Destination: Rome" in body["prompt"]


def test_generate_itinerary_still_works_after_adding_destination_context_routes(tmp_path) -> None:
    _write_contexts(tmp_path)
    client = _client(
        Settings(destination_context_enabled=True),
        FileDestinationKnowledgeProvider(tmp_path),
    )

    response = client.post("/generate-itinerary", json=VALID_PAYLOAD)

    assert response.status_code == 200
    assert "days" in response.json()


def test_destination_context_disabled_returns_empty_list(tmp_path) -> None:
    _write_contexts(tmp_path)
    client = _client(
        Settings(destination_context_enabled=False),
        FileDestinationKnowledgeProvider(tmp_path),
    )

    response = client.get("/destination-context")

    assert response.status_code == 200
    assert response.json() == {"items": []}


def test_destination_context_disabled_returns_404_for_destination(tmp_path) -> None:
    _write_contexts(tmp_path)
    client = _client(
        Settings(destination_context_enabled=False),
        FileDestinationKnowledgeProvider(tmp_path),
    )

    response = client.get("/destination-context/rome")

    assert response.status_code == 404
    assert response.json() == {"error": "Destination context not found"}


def test_destination_context_disabled_preview_prompt_returns_prompt_without_context(
    tmp_path,
) -> None:
    _write_contexts(tmp_path)
    client = _client(
        Settings(destination_context_enabled=False),
        FileDestinationKnowledgeProvider(tmp_path),
    )

    response = client.post("/destination-context/rome/preview-prompt", json=VALID_PAYLOAD)

    assert response.status_code == 200
    body = response.json()
    assert body["destinationContextFound"] is False
    assert body["destinationContext"] is None
    assert "DESTINATION CONTEXT:" not in body["prompt"]


def test_invalid_destination_context_dir_does_not_crash_service(
    tmp_path,
    monkeypatch,
) -> None:
    get_settings.cache_clear()
    monkeypatch.setenv("ITINERARY_GENERATOR_MODE", "mock")
    monkeypatch.setenv("DESTINATION_CONTEXT_ENABLED", "true")
    monkeypatch.setenv("DESTINATION_CONTEXT_DIR", str(tmp_path / "missing"))

    try:
        app = create_app()
        client = TestClient(app)
        response = client.get("/destination-context")
    finally:
        get_settings.cache_clear()

    assert app.state.destination_knowledge_provider is None
    assert response.status_code == 200
    assert response.json() == {"items": []}


def test_invalid_destination_context_dir_provider_returns_none(tmp_path) -> None:
    provider = get_destination_knowledge_provider(
        Settings(
            destination_context_enabled=True,
            destination_context_dir=str(tmp_path / "missing"),
        )
    )

    assert provider is None
