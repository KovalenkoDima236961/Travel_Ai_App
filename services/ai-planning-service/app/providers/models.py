from __future__ import annotations

from dataclasses import dataclass, field
from enum import StrEnum
from typing import Any


class AIProviderName(StrEnum):
    MOCK = "mock"
    OLLAMA = "ollama"
    OPENAI = "openai"


class AIFallbackProvider(StrEnum):
    NONE = "none"
    MOCK = "mock"
    OLLAMA = "ollama"


class AIOperation(StrEnum):
    ITINERARY = "generate_itinerary"
    REGENERATION = "regenerate"
    REGENERATE_DAY = "regenerate_day"
    REGENERATE_ITEM = "regenerate_item"
    REPAIR = "repair"
    REPAIR_ITINERARY = "repair_itinerary"
    REPAIR_GENERATION_OUTPUT = "repair_generation_output"
    DISCOVERY = "suggest_destinations"
    ROUTE_ALTERNATIVES = "suggest_route_alternatives"
    BUDGET_OPTIMIZATION = "optimize_budget_day"
    CHECKLIST = "generate_checklist"
    COPILOT = "copilot_respond"
    RECAP = "generate_trip_recap"
    EVALUATION = "evaluation"


@dataclass(frozen=True)
class ProviderTokenUsage:
    input_tokens: int = 0
    output_tokens: int = 0
    total_tokens: int = 0
    cached_input_tokens: int | None = None
    reasoning_tokens: int | None = None


@dataclass(frozen=True)
class ProviderFallbackMetadata:
    fallback_used: bool = False
    original_provider: str | None = None
    original_error_code: str | None = None
    fallback_provider: str | None = None
    fallback_reason: str | None = None
    quality_status: str | None = None
    needs_review: bool = False


@dataclass(frozen=True)
class ProviderGenerationResult[T]:
    content: T
    provider: str
    model: str | None
    provider_request_id: str | None = None
    token_usage: ProviderTokenUsage = field(default_factory=ProviderTokenUsage)
    latency_ms: int = 0
    retry_count: int = 0
    completion_status: str = "completed"
    fallback: ProviderFallbackMetadata = field(default_factory=ProviderFallbackMetadata)
    metadata: dict[str, Any] = field(default_factory=dict)
