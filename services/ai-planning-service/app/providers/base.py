from __future__ import annotations

from typing import Protocol

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


class AIModelProvider(Protocol):
    async def generate_itinerary(
        self, request: GenerateItineraryRequest
    ) -> ProviderGenerationResult[ItineraryResponse]: ...

    async def regenerate_day(
        self, request: RegenerateDayRequest
    ) -> ProviderGenerationResult[RegenerateDayResponse]: ...

    async def regenerate_item(
        self, request: RegenerateItemRequest
    ) -> ProviderGenerationResult[RegenerateItemResponse]: ...

    async def repair_itinerary(
        self, request: RepairItineraryRequest
    ) -> ProviderGenerationResult[RepairItineraryResponse]: ...

    async def repair_generation_output(
        self, request: RepairGenerationOutputRequest
    ) -> ProviderGenerationResult[RepairGenerationOutputResponse]: ...

    async def suggest_destinations(
        self, request: DestinationSuggestionRequest
    ) -> ProviderGenerationResult[DestinationSuggestionResponse]: ...

    async def suggest_route_alternatives(
        self, request: RouteAlternativeRequest
    ) -> ProviderGenerationResult[RouteAlternativeResponse]: ...

    async def optimize_budget_day(
        self, request: OptimizeBudgetDayRequest
    ) -> ProviderGenerationResult[BudgetOptimizationProposalResponse]: ...

    async def generate_checklist(
        self, request: GenerateChecklistRequest
    ) -> ProviderGenerationResult[GeneratedChecklistResponse]: ...

    async def respond_to_copilot(
        self, request: CopilotRespondRequest
    ) -> ProviderGenerationResult[CopilotRespondResponse]: ...

    async def generate_recap(
        self, request: GenerateTripRecapRequest
    ) -> ProviderGenerationResult[GenerateTripRecapResponse]: ...

    async def close(self) -> None: ...
