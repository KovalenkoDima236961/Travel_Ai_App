from __future__ import annotations

from typing import Any

from app.config import Settings
from app.providers.base import AIModelProvider
from app.providers.mock_provider import MockProvider
from app.providers.ollama_provider import OllamaProvider
from app.providers.openai_provider import OpenAIProvider


def build_model_provider(
    settings: Settings,
    *,
    destination_knowledge_provider: Any | None = None,
    knowledge_search_service: Any | None = None,
    client: Any | None = None,
) -> AIModelProvider:
    provider = settings.ai_model_provider.strip().lower() or "mock"
    if provider == "mock":
        return MockProvider()
    if provider == "ollama":
        return OllamaProvider(
            settings,
            destination_knowledge_provider=destination_knowledge_provider,
            knowledge_search_service=knowledge_search_service,
        )
    if provider == "openai":
        return OpenAIProvider(
            settings,
            client=client,
            destination_knowledge_provider=destination_knowledge_provider,
            knowledge_search_service=knowledge_search_service,
        )
    raise ValueError(
        f"Unknown AI_MODEL_PROVIDER {settings.ai_model_provider!r}; "
        "expected mock, ollama, or openai"
    )


def build_openai_provider_if_needed(
    settings: Settings,
    *,
    destination_knowledge_provider: Any | None = None,
    knowledge_search_service: Any | None = None,
    client: Any | None = None,
) -> OpenAIProvider | None:
    if not openai_provider_is_needed(settings):
        return None
    return OpenAIProvider(
        settings,
        client=client,
        destination_knowledge_provider=destination_knowledge_provider,
        knowledge_search_service=knowledge_search_service,
    )


def openai_provider_is_needed(settings: Settings) -> bool:
    modes = {
        settings.ai_model_provider,
        settings.itinerary_generator_mode,
        settings.copilot_mode,
        settings.trip_recap_mode,
    }
    return any(mode.strip().lower() == "openai" for mode in modes)
