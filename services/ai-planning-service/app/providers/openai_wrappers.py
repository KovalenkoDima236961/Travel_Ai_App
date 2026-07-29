from __future__ import annotations

import logging
from collections.abc import Callable
from typing import Any

from app.config import Settings
from app.core.errors import ItineraryGenerationError
from app.observability import record_ai_provider_fallback
from app.privacy import redact_text
from app.providers.errors import AIProviderError, AIProviderErrorCode
from app.providers.models import (
    ProviderFallbackMetadata,
    ProviderGenerationResult,
    ProviderTokenUsage,
)
from app.providers.openai_provider import OpenAIProvider, run_async_provider_call
from app.schemas.checklist import GenerateChecklistRequest, GeneratedChecklistResponse
from app.schemas.copilot import CopilotRespondRequest, CopilotRespondResponse
from app.schemas.destination_suggestion import (
    DestinationSuggestionRequest,
    DestinationSuggestionResponse,
)
from app.schemas.generation_repair import (
    RepairGenerationOutputRequest,
    RepairGenerationOutputResponse,
)
from app.schemas.itinerary import (
    BudgetOptimizationProposalResponse,
    GenerateItineraryRequest,
    ItineraryResponse,
    OptimizeBudgetDayRequest,
    RegenerateDayRequest,
    RegenerateDayResponse,
    RegenerateItemRequest,
    RegenerateItemResponse,
)
from app.schemas.repair import RepairItineraryRequest, RepairItineraryResponse
from app.schemas.route_alternatives import RouteAlternativeRequest, RouteAlternativeResponse
from app.schemas.trip_recap import GenerateTripRecapRequest, GenerateTripRecapResponse
from app.services.copilot import CopilotResponder, MockCopilotResponder, OllamaCopilotResponder
from app.services.destination_suggestion import (
    DestinationSuggestionGenerator,
    MockDestinationSuggestionGenerator,
    OllamaDestinationSuggestionGenerator,
)
from app.services.itinerary_generator import ItineraryGenerator, MockItineraryGenerator
from app.services.ollama_itinerary_generator import OllamaItineraryGenerator
from app.services.route_alternatives import (
    MockRouteAlternativeGenerator,
    OllamaRouteAlternativeGenerator,
    RouteAlternativeGenerator,
)
from app.services.trip_recap import (
    MockTripRecapGenerator,
    OllamaTripRecapGenerator,
    TripRecapGenerator,
)

logger = logging.getLogger(__name__)

_FALLBACK_ALLOWED_CODES = {
    AIProviderErrorCode.RATE_LIMITED,
    AIProviderErrorCode.QUOTA_EXCEEDED,
    AIProviderErrorCode.TIMEOUT,
    AIProviderErrorCode.CONNECTION_FAILED,
    AIProviderErrorCode.UNAVAILABLE,
}


class OpenAIItineraryGenerator:
    def __init__(
        self,
        settings: Settings,
        provider: OpenAIProvider,
        fallback_generator: ItineraryGenerator | None = None,
    ) -> None:
        self._settings = settings
        self._provider = provider
        self._fallback_generator = fallback_generator or _itinerary_fallback(settings)
        self.last_provider_result: ProviderGenerationResult[Any] | None = None

    def generate(self, request: GenerateItineraryRequest) -> ItineraryResponse:
        return self._call(
            "generate_itinerary",
            lambda: self._provider.generate_itinerary(request),
            lambda: self._fallback_generator.generate(request),
        )

    def generate_checklist(self, request: GenerateChecklistRequest) -> GeneratedChecklistResponse:
        return self._call(
            "generate_checklist",
            lambda: self._provider.generate_checklist(request),
            lambda: self._fallback_generator.generate_checklist(request),
        )

    def regenerate_day(self, request: RegenerateDayRequest) -> RegenerateDayResponse:
        return self._call(
            "regenerate_day",
            lambda: self._provider.regenerate_day(request),
            lambda: self._fallback_generator.regenerate_day(request),
        )

    def regenerate_item(self, request: RegenerateItemRequest) -> RegenerateItemResponse:
        return self._call(
            "regenerate_item",
            lambda: self._provider.regenerate_item(request),
            lambda: self._fallback_generator.regenerate_item(request),
        )

    def optimize_budget_day(
        self, request: OptimizeBudgetDayRequest
    ) -> BudgetOptimizationProposalResponse:
        return self._call(
            "optimize_budget_day",
            lambda: self._provider.optimize_budget_day(request),
            lambda: self._fallback_generator.optimize_budget_day(request),
        )

    def repair_itinerary(self, request: RepairItineraryRequest) -> RepairItineraryResponse:
        return self._call(
            "repair_itinerary",
            lambda: self._provider.repair_itinerary(request),
            lambda: self._fallback_generator.repair_itinerary(request),
        )

    def repair_generation_output(
        self, request: RepairGenerationOutputRequest
    ) -> RepairGenerationOutputResponse:
        return self._call(
            "repair_generation_output",
            lambda: self._provider.repair_generation_output(request),
            lambda: self._fallback_generator.repair_generation_output(request),
        )

    def _call[T](
        self,
        operation: str,
        provider_call: Callable[[], Any],
        fallback_call: Callable[[], T],
    ) -> T:
        return _call_openai_with_fallback(
            settings=self._settings,
            operation=operation,
            provider_call=provider_call,
            fallback_call=fallback_call,
            set_last_result=self._set_last_result,
        )

    def _set_last_result(self, result: ProviderGenerationResult[Any]) -> None:
        self.last_provider_result = result


class OpenAIDestinationSuggestionGenerator:
    def __init__(
        self,
        settings: Settings,
        provider: OpenAIProvider,
        fallback: DestinationSuggestionGenerator | None = None,
    ) -> None:
        self._settings = settings
        self._provider = provider
        self._fallback = fallback or _destination_fallback(settings)
        self.last_provider_result: ProviderGenerationResult[Any] | None = None

    def suggest(self, request: DestinationSuggestionRequest) -> DestinationSuggestionResponse:
        return _call_openai_with_fallback(
            settings=self._settings,
            operation="suggest_destinations",
            provider_call=lambda: self._provider.suggest_destinations(request),
            fallback_call=lambda: self._fallback.suggest(request),
            set_last_result=self._set_last_result,
        )

    def _set_last_result(self, result: ProviderGenerationResult[Any]) -> None:
        self.last_provider_result = result


class OpenAIRouteAlternativeGenerator:
    def __init__(
        self,
        settings: Settings,
        provider: OpenAIProvider,
        fallback: RouteAlternativeGenerator | None = None,
    ) -> None:
        self._settings = settings
        self._provider = provider
        self._fallback = fallback or _route_fallback(settings)
        self.last_provider_result: ProviderGenerationResult[Any] | None = None

    def suggest(self, request: RouteAlternativeRequest) -> RouteAlternativeResponse:
        return _call_openai_with_fallback(
            settings=self._settings,
            operation="suggest_route_alternatives",
            provider_call=lambda: self._provider.suggest_route_alternatives(request),
            fallback_call=lambda: self._fallback.suggest(request),
            set_last_result=self._set_last_result,
        )

    def _set_last_result(self, result: ProviderGenerationResult[Any]) -> None:
        self.last_provider_result = result


class OpenAICopilotResponder:
    def __init__(
        self,
        settings: Settings,
        provider: OpenAIProvider,
        fallback: CopilotResponder | None = None,
    ) -> None:
        self._settings = settings
        self._provider = provider
        self._fallback = fallback or _copilot_fallback(settings)
        self.last_provider_result: ProviderGenerationResult[Any] | None = None

    def respond(self, request: CopilotRespondRequest) -> CopilotRespondResponse:
        return _call_openai_with_fallback(
            settings=self._settings,
            operation="copilot_respond",
            provider_call=lambda: self._provider.respond_to_copilot(request),
            fallback_call=lambda: self._fallback.respond(request),
            set_last_result=self._set_last_result,
        )

    def _set_last_result(self, result: ProviderGenerationResult[Any]) -> None:
        self.last_provider_result = result


class OpenAITripRecapGenerator:
    def __init__(
        self,
        settings: Settings,
        provider: OpenAIProvider,
        fallback: TripRecapGenerator | None = None,
    ) -> None:
        self._settings = settings
        self._provider = provider
        self._fallback = fallback or _recap_fallback(settings)
        self.last_provider_result: ProviderGenerationResult[Any] | None = None

    def generate(self, request: GenerateTripRecapRequest) -> GenerateTripRecapResponse:
        return _call_openai_with_fallback(
            settings=self._settings,
            operation="generate_trip_recap",
            provider_call=lambda: self._provider.generate_recap(request),
            fallback_call=lambda: self._fallback.generate(request),
            set_last_result=self._set_last_result,
        )

    def _set_last_result(self, result: ProviderGenerationResult[Any]) -> None:
        self.last_provider_result = result


def _call_openai_with_fallback[T](
    *,
    settings: Settings,
    operation: str,
    provider_call: Callable[[], Any],
    fallback_call: Callable[[], T],
    set_last_result: Callable[[ProviderGenerationResult[Any]], None],
) -> T:
    try:
        result = run_async_provider_call(provider_call())
        set_last_result(result)
        return result.content
    except AIProviderError as exc:
        if not _fallback_allowed(settings, exc):
            logger.warning(
                "OpenAI provider request failed without fallback",
                extra={"operation": operation, **exc.safe_metadata()},
            )
            raise ItineraryGenerationError("Failed to generate itinerary") from exc

        fallback_provider = settings.ai_model_provider_fallback.strip().lower()
        logger.warning(
            "OpenAI provider request failed; using configured fallback",
            extra={
                "operation": operation,
                "fallbackProvider": fallback_provider,
                "errorCode": exc.code.value,
                "requestId": exc.request_id,
                "model": exc.model,
            },
        )
        record_ai_provider_fallback(
            provider="openai",
            operation=operation,
            fallback_provider=fallback_provider,
            reason=exc.code.value,
        )
        content = fallback_call()
        set_last_result(
            ProviderGenerationResult(
                content=content,
                provider=fallback_provider,
                model=_fallback_model(settings, fallback_provider),
                token_usage=ProviderTokenUsage(),
                completion_status="fallback_completed",
                fallback=ProviderFallbackMetadata(
                    fallback_used=True,
                    original_provider="openai",
                    original_error_code=exc.code.value,
                    fallback_provider=fallback_provider,
                    fallback_reason=exc.code.value,
                    quality_status="fallback_mock" if fallback_provider == "mock" else "fallback",
                    needs_review=fallback_provider == "mock",
                ),
                metadata={"originalError": redact_text(exc.code.value, max_chars=120)},
            )
        )
        return content


def _fallback_allowed(settings: Settings, exc: AIProviderError) -> bool:
    return (
        settings.ai_model_provider_fallback_enabled
        and settings.ai_model_provider_fallback.strip().lower() != "none"
        and exc.code in _FALLBACK_ALLOWED_CODES
    )


def _itinerary_fallback(settings: Settings) -> ItineraryGenerator:
    provider = settings.ai_model_provider_fallback.strip().lower()
    if provider == "ollama":
        return OllamaItineraryGenerator(settings)
    return MockItineraryGenerator()


def _destination_fallback(settings: Settings) -> DestinationSuggestionGenerator:
    provider = settings.ai_model_provider_fallback.strip().lower()
    if provider == "ollama":
        return OllamaDestinationSuggestionGenerator(settings)
    return MockDestinationSuggestionGenerator()


def _route_fallback(settings: Settings) -> RouteAlternativeGenerator:
    provider = settings.ai_model_provider_fallback.strip().lower()
    if provider == "ollama":
        return OllamaRouteAlternativeGenerator(settings)
    return MockRouteAlternativeGenerator()


def _copilot_fallback(settings: Settings) -> CopilotResponder:
    provider = settings.ai_model_provider_fallback.strip().lower()
    if provider == "ollama":
        return OllamaCopilotResponder(settings)
    return MockCopilotResponder()


def _recap_fallback(settings: Settings) -> TripRecapGenerator:
    provider = settings.ai_model_provider_fallback.strip().lower()
    if provider == "ollama":
        return OllamaTripRecapGenerator(settings)
    return MockTripRecapGenerator()


def _fallback_model(settings: Settings, provider: str) -> str:
    if provider == "ollama":
        return settings.ollama_model
    return "mock-v1"
