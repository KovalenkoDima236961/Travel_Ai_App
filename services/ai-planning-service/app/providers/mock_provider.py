from __future__ import annotations

import time
from typing import TypeVar

from app.providers.models import ProviderGenerationResult
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
from app.services.copilot import MockCopilotResponder
from app.services.destination_suggestion import MockDestinationSuggestionGenerator
from app.services.itinerary_generator import MockItineraryGenerator
from app.services.route_alternatives import MockRouteAlternativeGenerator
from app.services.trip_recap import MockTripRecapGenerator

T = TypeVar("T")


class MockProvider:
    provider_name = "mock"

    def __init__(self) -> None:
        self._itinerary = MockItineraryGenerator()
        self._destinations = MockDestinationSuggestionGenerator()
        self._routes = MockRouteAlternativeGenerator()
        self._copilot = MockCopilotResponder()
        self._recap = MockTripRecapGenerator()

    async def generate_itinerary(
        self, request: GenerateItineraryRequest
    ) -> ProviderGenerationResult[ItineraryResponse]:
        return self._result(self._itinerary.generate(request), "generate_itinerary")

    async def regenerate_day(
        self, request: RegenerateDayRequest
    ) -> ProviderGenerationResult[RegenerateDayResponse]:
        return self._result(self._itinerary.regenerate_day(request), "regenerate_day")

    async def regenerate_item(
        self, request: RegenerateItemRequest
    ) -> ProviderGenerationResult[RegenerateItemResponse]:
        return self._result(self._itinerary.regenerate_item(request), "regenerate_item")

    async def optimize_budget_day(
        self, request: OptimizeBudgetDayRequest
    ) -> ProviderGenerationResult[BudgetOptimizationProposalResponse]:
        return self._result(self._itinerary.optimize_budget_day(request), "optimize_budget_day")

    async def repair_itinerary(
        self, request: RepairItineraryRequest
    ) -> ProviderGenerationResult[RepairItineraryResponse]:
        return self._result(self._itinerary.repair_itinerary(request), "repair_itinerary")

    async def repair_generation_output(
        self, request: RepairGenerationOutputRequest
    ) -> ProviderGenerationResult[RepairGenerationOutputResponse]:
        return self._result(
            self._itinerary.repair_generation_output(request), "repair_generation_output"
        )

    async def suggest_destinations(
        self, request: DestinationSuggestionRequest
    ) -> ProviderGenerationResult[DestinationSuggestionResponse]:
        return self._result(self._destinations.suggest(request), "suggest_destinations")

    async def suggest_route_alternatives(
        self, request: RouteAlternativeRequest
    ) -> ProviderGenerationResult[RouteAlternativeResponse]:
        return self._result(self._routes.suggest(request), "suggest_route_alternatives")

    async def generate_checklist(
        self, request: GenerateChecklistRequest
    ) -> ProviderGenerationResult[GeneratedChecklistResponse]:
        return self._result(self._itinerary.generate_checklist(request), "generate_checklist")

    async def respond_to_copilot(
        self, request: CopilotRespondRequest
    ) -> ProviderGenerationResult[CopilotRespondResponse]:
        return self._result(self._copilot.respond(request), "copilot_respond")

    async def generate_recap(
        self, request: GenerateTripRecapRequest
    ) -> ProviderGenerationResult[GenerateTripRecapResponse]:
        return self._result(self._recap.generate(request), "generate_trip_recap")

    async def close(self) -> None:
        return None

    def _result(self, content: T, operation: str) -> ProviderGenerationResult[T]:
        started_at = time.monotonic()
        return ProviderGenerationResult(
            content=content,
            provider=self.provider_name,
            model="mock-v1",
            latency_ms=max(0, int((time.monotonic() - started_at) * 1000)),
            metadata={"operation": operation},
        )
