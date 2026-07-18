import json
import logging
from copy import deepcopy
from typing import Any

import httpx
import pytest
from fastapi.testclient import TestClient

from app.config import Settings
from app.core.errors import ItineraryGenerationError
from app.main import create_app
from app.schemas.destination_context import DestinationContext
from app.schemas.itinerary import GenerateItineraryRequest
from app.schemas.knowledge import KnowledgeSearchResult
from app.services.generator_factory import get_itinerary_generator
from app.services.itinerary_generator import MockItineraryGenerator
from app.services.llm_response_parser import LLMResponseParseError, parse_itinerary_response
from app.services.ollama_itinerary_generator import OllamaItineraryGenerator

VALID_PAYLOAD = {
    "tripId": "550e8400-e29b-41d4-a716-446655440000",
    "destination": "Rome",
    "startDate": "2026-08-10",
    "days": 2,
    "budgetAmount": 600,
    "budgetCurrency": "EUR",
    "travelers": 2,
    "interests": ["food", "history", "hidden_gems"],
    "pace": "balanced",
}


def _settings(**overrides: Any) -> Settings:
    values = {
        "itinerary_generator_mode": "ollama",
        "ollama_base_url": "http://ollama:11434",
        "ollama_model": "llama3.1:8b",
        "ollama_fallback_to_mock": True,
    }
    values.update(overrides)
    return Settings(**values)


def _request(**overrides: Any) -> GenerateItineraryRequest:
    payload = deepcopy(VALID_PAYLOAD)
    payload.update(overrides)
    return GenerateItineraryRequest.model_validate(payload)


class StaticDestinationKnowledgeProvider:
    def __init__(self, context: DestinationContext | None) -> None:
        self.context = context
        self.requests: list[str] = []

    def get_context(self, destination: str) -> DestinationContext | None:
        self.requests.append(destination)
        return self.context


class FailingDestinationKnowledgeProvider:
    def get_context(self, destination: str) -> DestinationContext | None:
        raise AssertionError("provider should not be called")


class StaticKnowledgeSearchService:
    def __init__(self, items: list[KnowledgeSearchResult]) -> None:
        self.items = items
        self.calls: list[dict[str, Any]] = []

    def search(
        self,
        destination: str,
        interests: list[str],
        query: str | None,
        top_k: int,
    ) -> list[KnowledgeSearchResult]:
        self.calls.append(
            {
                "destination": destination,
                "interests": interests,
                "query": query,
                "top_k": top_k,
            }
        )
        return self.items


def _itinerary_body(days: int = 2) -> dict[str, Any]:
    return {
        "days": [
            {
                "day": day_number,
                "title": f"Day {day_number}: Rome practical highlights",
                "items": [
                    {
                        "time": "09:00",
                        "type": "place",
                        "name": "Historic center walk",
                        "note": "Start early around the old streets before the busiest crowds.",
                        "estimatedCost": 0,
                    },
                    {
                        "time": "12:30",
                        "type": "food",
                        "name": "Local lunch stop",
                        "note": "Choose a trattoria near the next stop to avoid backtracking.",
                        "estimatedCost": 18,
                    },
                    {
                        "time": "15:30",
                        "type": "activity",
                        "name": "Focused museum visit",
                        "note": "Book the timed entry before leaving in the morning.",
                        "estimatedCost": 16,
                    },
                    {
                        "time": "19:00",
                        "type": "food",
                        "name": "Dinner in a neighborhood district",
                        "note": "Reserve a table away from the most visible tourist streets.",
                        "estimatedCost": 28,
                    },
                ],
            }
            for day_number in range(1, days + 1)
        ]
    }


def test_mock_mode_factory_still_uses_mock_generator() -> None:
    generator = get_itinerary_generator(Settings(itinerary_generator_mode="mock"))

    assert isinstance(generator, MockItineraryGenerator)
    assert len(generator.generate(_request()).days) == VALID_PAYLOAD["days"]


def test_ollama_mode_sends_request_to_api_generate() -> None:
    captured: dict[str, Any] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["path"] = request.url.path
        captured["body"] = json.loads(request.content)
        return httpx.Response(200, json={"response": json.dumps(_itinerary_body())})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(settings=_settings(), http_client=http_client)
        response = generator.generate(_request())

    assert captured["path"] == "/api/generate"
    assert captured["body"]["model"] == "llama3.1:8b"
    assert captured["body"]["stream"] is False
    assert captured["body"]["options"] == {"temperature": 0.2, "num_predict": 2048}
    assert len(response.days) == VALID_PAYLOAD["days"]


def test_ollama_mode_parses_valid_json_response_successfully() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"response": json.dumps(_itinerary_body())})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(settings=_settings(), http_client=http_client)
        response = generator.generate(_request())

    assert response.days[0].items[0].name == "Historic center walk"


def test_ollama_mode_parses_markdown_fenced_json_response_successfully() -> None:
    fenced_response = f"```json\n{json.dumps(_itinerary_body())}\n```"

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"response": fenced_response})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(settings=_settings(), http_client=http_client)
        response = generator.generate(_request())

    assert len(response.days) == VALID_PAYLOAD["days"]


def test_parser_strips_markdown_code_fences() -> None:
    response_text = f"```json\n{json.dumps(_itinerary_body(days=1))}\n```"

    itinerary = parse_itinerary_response(response_text, expected_days=1)

    assert len(itinerary.days) == 1
    assert itinerary.days[0].day == 1


def test_parser_validates_days_count_matches_request() -> None:
    response_text = json.dumps(_itinerary_body(days=1))

    with pytest.raises(LLMResponseParseError, match="Expected 2 itinerary day"):
        parse_itinerary_response(response_text, expected_days=2)


def test_ollama_mode_falls_back_to_mock_when_request_fails_and_fallback_enabled() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("connection refused", request=request)

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(
            settings=_settings(ollama_fallback_to_mock=True),
            http_client=http_client,
        )
        response = generator.generate(_request())

    assert len(response.days) == VALID_PAYLOAD["days"]
    assert response.days[0].title.startswith("Day 1: Rome")


def test_ollama_mode_returns_error_when_request_fails_and_fallback_disabled() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("connection refused", request=request)

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        app = create_app()
        app.state.itinerary_generator = OllamaItineraryGenerator(
            settings=_settings(ollama_fallback_to_mock=False),
            http_client=http_client,
        )
        client = TestClient(app)
        response = client.post("/generate-itinerary", json=VALID_PAYLOAD)

    assert response.status_code == 500
    assert response.json() == {"error": "Failed to generate itinerary"}


def test_unknown_generator_mode_fails_clearly() -> None:
    with pytest.raises(ValueError, match="Unknown ITINERARY_GENERATOR_MODE"):
        get_itinerary_generator(Settings(itinerary_generator_mode="invalid"))


def test_missing_ollama_base_url_in_ollama_mode_fails_clearly() -> None:
    with pytest.raises(ValueError, match="OLLAMA_BASE_URL is required"):
        get_itinerary_generator(_settings(ollama_base_url=""))


def test_missing_ollama_model_in_ollama_mode_fails_clearly() -> None:
    with pytest.raises(ValueError, match="OLLAMA_MODEL is required"):
        get_itinerary_generator(_settings(ollama_model=""))


def test_ollama_factory_ignores_missing_destination_context_dir_when_enabled() -> None:
    generator = get_itinerary_generator(
        _settings(destination_context_enabled=True, destination_context_dir="/missing/context")
    )

    assert isinstance(generator, OllamaItineraryGenerator)


def test_generated_response_keeps_itinerary_shape_with_optional_metadata() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"response": json.dumps(_itinerary_body())})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(settings=_settings(), http_client=http_client)
        response = generator.generate(_request())

    body = json.loads(response.model_dump_json(by_alias=True))
    assert set(body.keys()) == {"days", "metadata"}
    assert body["metadata"] is None
    assert set(body["days"][0].keys()) == {"day", "title", "items"}
    assert set(body["days"][0]["items"][0].keys()) == {
        "time",
        "type",
        "name",
        "note",
        "estimatedCost",
    }


def test_wrong_number_of_days_triggers_repair_and_returns_repaired_itinerary() -> None:
    captured_prompts: list[str] = []
    responses = [
        json.dumps(_itinerary_body(days=1)),
        json.dumps(_itinerary_body(days=2)),
    ]

    def handler(request: httpx.Request) -> httpx.Response:
        captured_prompts.append(json.loads(request.content)["prompt"])
        return httpx.Response(200, json={"response": responses.pop(0)})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(settings=_settings(), http_client=http_client)
        response = generator.generate(_request())

    assert len(captured_prompts) == 2
    assert "Validation error:" in captured_prompts[1]
    assert "Return ONLY corrected JSON" in captured_prompts[1]
    assert len(response.days) == 2


def test_ollama_mode_injects_destination_context_into_initial_and_repair_prompts() -> None:
    captured_prompts: list[str] = []
    responses = [
        json.dumps(_itinerary_body(days=1)),
        json.dumps(_itinerary_body(days=2)),
    ]
    provider = StaticDestinationKnowledgeProvider(
        DestinationContext(
            destination="Rome",
            aliases=["Roma"],
            localTips=["Visit popular attractions early."],
            hiddenGems=["Orange Garden"],
            foodTips=["Try carbonara."],
            avoid=["Do not overload Vatican and Colosseum on one day."],
            transportTips=["Group nearby attractions together."],
            budgetTips=["Use free viewpoints."],
        )
    )

    def handler(request: httpx.Request) -> httpx.Response:
        captured_prompts.append(json.loads(request.content)["prompt"])
        return httpx.Response(200, json={"response": responses.pop(0)})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(
            settings=_settings(),
            http_client=http_client,
            destination_knowledge_provider=provider,
        )
        response = generator.generate(_request())

    assert provider.requests == ["Rome"]
    assert len(response.days) == 2
    assert "DESTINATION CONTEXT:" in captured_prompts[0]
    assert "- Destination: Rome" in captured_prompts[0]
    assert "Orange Garden" in captured_prompts[0]
    assert "Try carbonara." in captured_prompts[0]
    assert "DESTINATION CONTEXT:" in captured_prompts[1]
    assert "The corrected JSON should still use the destination context" in captured_prompts[1]


def test_ollama_mode_does_not_lookup_destination_context_when_disabled() -> None:
    captured_prompt: dict[str, str] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured_prompt["prompt"] = json.loads(request.content)["prompt"]
        return httpx.Response(200, json={"response": json.dumps(_itinerary_body())})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(
            settings=_settings(destination_context_enabled=False),
            http_client=http_client,
            destination_knowledge_provider=FailingDestinationKnowledgeProvider(),
        )
        response = generator.generate(_request())

    assert len(response.days) == VALID_PAYLOAD["days"]
    assert "DESTINATION CONTEXT" not in captured_prompt["prompt"]


def test_ollama_mode_calls_rag_search_and_injects_chunks_when_enabled() -> None:
    captured_prompt: dict[str, str] = {}
    search_service = StaticKnowledgeSearchService(
        [
            KnowledgeSearchResult(
                id="rome:food.md:0",
                destination="rome",
                source="food.md",
                content="Use Testaccio for traditional Roman food.",
                score=0.91,
                metadata={"chunkIndex": 0},
            )
        ]
    )

    def handler(request: httpx.Request) -> httpx.Response:
        captured_prompt["prompt"] = json.loads(request.content)["prompt"]
        return httpx.Response(200, json={"response": json.dumps(_itinerary_body())})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(
            settings=_settings(rag_enabled=True, rag_top_k=3),
            http_client=http_client,
            knowledge_search_service=search_service,
        )
        response = generator.generate(_request())

    assert len(response.days) == VALID_PAYLOAD["days"]
    assert search_service.calls
    assert search_service.calls[0]["destination"] == "Rome"
    assert search_service.calls[0]["interests"] == ["food", "history", "hidden_gems"]
    assert search_service.calls[0]["top_k"] == 3
    assert "pace: balanced" in search_service.calls[0]["query"]
    assert "budget: 600 EUR" in search_service.calls[0]["query"]
    assert "RAG CONTEXT:" in captured_prompt["prompt"]
    assert "- Source: food.md" in captured_prompt["prompt"]
    assert "Use Testaccio for traditional Roman food." in captured_prompt["prompt"]


def test_repair_response_invalid_and_fallback_enabled_returns_mock_itinerary() -> None:
    calls = 0

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal calls
        calls += 1
        return httpx.Response(200, json={"response": json.dumps(_itinerary_body(days=1))})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(
            settings=_settings(ollama_fallback_to_mock=True),
            http_client=http_client,
        )
        response = generator.generate(_request())

    assert calls == 2
    assert len(response.days) == VALID_PAYLOAD["days"]
    assert response.days[0].title.startswith("Day 1: Rome")


def test_repair_response_invalid_and_fallback_disabled_raises_generation_error() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"response": json.dumps(_itinerary_body(days=1))})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(
            settings=_settings(ollama_fallback_to_mock=False),
            http_client=http_client,
        )

        with pytest.raises(ItineraryGenerationError):
            generator.generate(_request())


def test_initial_invalid_json_and_repair_enabled_triggers_repair() -> None:
    calls = 0

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal calls
        calls += 1
        if calls == 1:
            return httpx.Response(200, json={"response": "this is not json"})
        return httpx.Response(200, json={"response": json.dumps(_itinerary_body())})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(settings=_settings(), http_client=http_client)
        response = generator.generate(_request())

    assert calls == 2
    assert len(response.days) == VALID_PAYLOAD["days"]


def test_initial_invalid_json_and_repair_disabled_falls_back_when_fallback_enabled() -> None:
    calls = 0

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal calls
        calls += 1
        return httpx.Response(200, json={"response": "this is not json"})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(
            settings=_settings(ollama_repair_enabled=False, ollama_fallback_to_mock=True),
            http_client=http_client,
        )
        response = generator.generate(_request())

    assert calls == 1
    assert len(response.days) == VALID_PAYLOAD["days"]
    assert response.days[0].title.startswith("Day 1: Rome")


def test_log_llm_payloads_false_does_not_log_full_prompt_or_response(
    caplog: pytest.LogCaptureFixture,
) -> None:
    caplog.set_level(logging.DEBUG, logger="app.services.ollama_itinerary_generator")
    raw_response = json.dumps(_itinerary_body())

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"response": raw_response})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(
            settings=_settings(log_llm_payloads=False),
            http_client=http_client,
        )
        generator.generate(_request())

    assert all("prompt" not in record.__dict__ for record in caplog.records)
    assert all("raw_llm_response" not in record.__dict__ for record in caplog.records)


def test_log_llm_payloads_true_in_development_allows_payload_logging(
    caplog: pytest.LogCaptureFixture,
) -> None:
    caplog.set_level(logging.DEBUG, logger="app.services.ollama_itinerary_generator")
    raw_response = json.dumps(_itinerary_body())

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"response": raw_response})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(
            settings=_settings(app_env="development", log_llm_payloads=True),
            http_client=http_client,
        )
        generator.generate(_request())

    assert any("Trip request:" in record.__dict__.get("prompt", "") for record in caplog.records)
    assert any(record.__dict__.get("raw_llm_response") == raw_response for record in caplog.records)


def test_log_llm_payloads_true_outside_development_does_not_log_payloads(
    caplog: pytest.LogCaptureFixture,
) -> None:
    caplog.set_level(logging.DEBUG, logger="app.services.ollama_itinerary_generator")
    raw_response = json.dumps(_itinerary_body())

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"response": raw_response})

    with httpx.Client(transport=httpx.MockTransport(handler)) as http_client:
        generator = OllamaItineraryGenerator(
            settings=_settings(app_env="production", log_llm_payloads=True),
            http_client=http_client,
        )
        generator.generate(_request())

    assert all("prompt" not in record.__dict__ for record in caplog.records)
    assert all("raw_llm_response" not in record.__dict__ for record in caplog.records)


def test_ollama_repair_attempts_greater_than_one_is_clamped_to_one() -> None:
    assert _settings(ollama_repair_attempts=3).ollama_repair_attempts == 1
