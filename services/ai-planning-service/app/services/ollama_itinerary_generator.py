import logging
import time
from typing import Any

import httpx

from app.config import Settings
from app.core.errors import ItineraryGenerationError
from app.schemas.destination_context import DestinationContext
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
from app.schemas.knowledge import KnowledgeSearchResult
from app.services.destination_knowledge import DestinationKnowledgeProvider
from app.services.itinerary_generator import ItineraryGenerator, MockItineraryGenerator
from app.services.itinerary_validator import ItineraryValidationError, ItineraryValidator
from app.services.knowledge_search import KnowledgeSearchService
from app.services.llm_response_parser import (
    LLMResponseParseError,
    parse_budget_optimization_response,
    parse_itinerary_response,
    parse_regenerate_day_response,
    parse_regenerate_item_response,
)
from app.services.prompt_builder import (
    build_itinerary_prompt,
    build_optimize_budget_day_prompt,
    build_regenerate_day_prompt,
    build_regenerate_day_repair_prompt,
    build_regenerate_item_prompt,
    build_regenerate_item_repair_prompt,
    build_repair_prompt,
)

logger = logging.getLogger(__name__)


class OllamaClientError(RuntimeError):
    """Raised when the Ollama API cannot provide a usable response."""


class OllamaItineraryGenerator:
    def __init__(
        self,
        settings: Settings,
        fallback_generator: ItineraryGenerator | None = None,
        http_client: httpx.Client | None = None,
        destination_knowledge_provider: DestinationKnowledgeProvider | None = None,
        knowledge_search_service: KnowledgeSearchService | None = None,
    ) -> None:
        self._settings = settings
        self._fallback_generator = fallback_generator or MockItineraryGenerator()
        self._http_client = http_client
        self._destination_knowledge_provider = destination_knowledge_provider
        self._knowledge_search_service = knowledge_search_service
        self._validator = ItineraryValidator()

    def generate(self, request: GenerateItineraryRequest) -> ItineraryResponse:
        started_at = time.monotonic()
        log_context = self._base_log_context(request)

        try:
            itinerary = self._generate_with_ollama(request, log_context)
            log_context["generation_duration_ms"] = self._duration_ms(started_at)
            logger.info("Ollama itinerary generation succeeded", extra=log_context)
            return itinerary
        except (
            httpx.HTTPError,
            OllamaClientError,
            LLMResponseParseError,
            ItineraryValidationError,
        ) as exc:
            self._record_generation_error(log_context, exc)
            log_context["generation_duration_ms"] = self._duration_ms(started_at)

            if self._settings.ollama_fallback_to_mock:
                log_context["fallback_used"] = True
                logger.warning(
                    "Ollama itinerary generation failed; falling back to mock generator",
                    extra=log_context,
                    exc_info=True,
                )
                return self._fallback_generator.generate(request)

            logger.error("Ollama itinerary generation failed", extra=log_context, exc_info=True)
            raise ItineraryGenerationError("Failed to generate itinerary") from exc

    def regenerate_day(self, request: RegenerateDayRequest) -> RegenerateDayResponse:
        started_at = time.monotonic()
        log_context = self._base_partial_log_context(request, "regenerate_day")

        try:
            replacement = self._regenerate_day_with_ollama(request, log_context)
            log_context["generation_duration_ms"] = self._duration_ms(started_at)
            logger.info("Ollama itinerary day regeneration succeeded", extra=log_context)
            return replacement
        except (httpx.HTTPError, OllamaClientError, LLMResponseParseError) as exc:
            self._record_generation_error(log_context, exc)
            log_context["generation_duration_ms"] = self._duration_ms(started_at)

            if self._settings.ollama_fallback_to_mock:
                log_context["fallback_used"] = True
                logger.warning(
                    "Ollama itinerary day regeneration failed; falling back to mock generator",
                    extra=log_context,
                    exc_info=True,
                )
                return self._fallback_generator.regenerate_day(request)

            logger.error(
                "Ollama itinerary day regeneration failed",
                extra=log_context,
                exc_info=True,
            )
            raise ItineraryGenerationError("Failed to regenerate itinerary day") from exc

    def regenerate_item(self, request: RegenerateItemRequest) -> RegenerateItemResponse:
        started_at = time.monotonic()
        log_context = self._base_partial_log_context(request, "regenerate_item")

        try:
            replacement = self._regenerate_item_with_ollama(request, log_context)
            log_context["generation_duration_ms"] = self._duration_ms(started_at)
            logger.info("Ollama itinerary item regeneration succeeded", extra=log_context)
            return replacement
        except (httpx.HTTPError, OllamaClientError, LLMResponseParseError) as exc:
            self._record_generation_error(log_context, exc)
            log_context["generation_duration_ms"] = self._duration_ms(started_at)

            if self._settings.ollama_fallback_to_mock:
                log_context["fallback_used"] = True
                logger.warning(
                    "Ollama itinerary item regeneration failed; falling back to mock generator",
                    extra=log_context,
                    exc_info=True,
                )
                return self._fallback_generator.regenerate_item(request)

            logger.error(
                "Ollama itinerary item regeneration failed",
                extra=log_context,
                exc_info=True,
            )
            raise ItineraryGenerationError("Failed to regenerate itinerary item") from exc

    def optimize_budget_day(
        self, request: OptimizeBudgetDayRequest
    ) -> BudgetOptimizationProposalResponse:
        started_at = time.monotonic()
        log_context = self._base_budget_optimization_log_context(request)

        try:
            proposal = self._optimize_budget_day_with_ollama(request, log_context)
            log_context["generation_duration_ms"] = self._duration_ms(started_at)
            logger.info("Ollama budget optimization succeeded", extra=log_context)
            return proposal
        except (httpx.HTTPError, OllamaClientError, LLMResponseParseError) as exc:
            self._record_generation_error(log_context, exc)
            log_context["generation_duration_ms"] = self._duration_ms(started_at)

            if self._settings.ollama_fallback_to_mock:
                log_context["fallback_used"] = True
                logger.warning(
                    "Ollama budget optimization failed; falling back to mock generator",
                    extra=log_context,
                    exc_info=True,
                )
                return self._fallback_generator.optimize_budget_day(request)

            logger.error("Ollama budget optimization failed", extra=log_context, exc_info=True)
            raise ItineraryGenerationError("Failed to optimize itinerary budget") from exc

    def _generate_with_ollama(
        self,
        request: GenerateItineraryRequest,
        log_context: dict[str, Any],
    ) -> ItineraryResponse:
        destination_context = self._get_destination_context(request, log_context)
        rag_chunks = self._get_rag_chunks(request, log_context)
        prompt = build_itinerary_prompt(
            request,
            destination_context=destination_context,
            rag_chunks=rag_chunks,
        )
        self._log_llm_payload("Ollama itinerary prompt", "prompt", prompt, log_context)

        llm_response = self._call_ollama(prompt)
        self._log_llm_payload(
            "Ollama raw itinerary response",
            "raw_llm_response",
            llm_response,
            log_context,
        )

        try:
            return self._parse_and_validate(request, llm_response)
        except (LLMResponseParseError, ItineraryValidationError) as exc:
            self._record_generation_error(log_context, exc)

            if not self._repair_is_enabled():
                raise

            log_context["repair_attempted"] = True
            repair_prompt = build_repair_prompt(
                request=request,
                invalid_response_text=llm_response,
                validation_error=self._validation_error_for_prompt(exc),
                destination_context=destination_context,
                rag_chunks=rag_chunks,
            )
            self._log_llm_payload(
                "Ollama itinerary repair prompt",
                "repair_prompt",
                repair_prompt,
                log_context,
            )

            repair_response = self._call_ollama(repair_prompt)
            self._log_llm_payload(
                "Ollama raw itinerary repair response",
                "raw_repair_response",
                repair_response,
                log_context,
            )

            repaired_itinerary = self._parse_and_validate(request, repair_response)
            log_context["repair_succeeded"] = True
            return repaired_itinerary

    def _regenerate_day_with_ollama(
        self,
        request: RegenerateDayRequest,
        log_context: dict[str, Any],
    ) -> RegenerateDayResponse:
        destination_context = self._get_destination_context_for(
            request.trip.destination,
            log_context,
        )
        rag_chunks = self._get_partial_rag_chunks(request, log_context)
        prompt = build_regenerate_day_prompt(
            request,
            destination_context=destination_context,
            rag_chunks=rag_chunks,
        )
        self._log_llm_payload("Ollama regenerate day prompt", "prompt", prompt, log_context)

        llm_response = self._call_ollama(prompt)
        self._log_llm_payload(
            "Ollama raw regenerate day response",
            "raw_llm_response",
            llm_response,
            log_context,
        )

        try:
            return parse_regenerate_day_response(llm_response, request.day_number)
        except LLMResponseParseError as exc:
            self._record_generation_error(log_context, exc)
            if not self._repair_is_enabled():
                raise

            log_context["repair_attempted"] = True
            repair_prompt = build_regenerate_day_repair_prompt(
                request=request,
                invalid_response_text=llm_response,
                validation_error=self._validation_error_for_prompt(exc),
                destination_context=destination_context,
                rag_chunks=rag_chunks,
            )
            self._log_llm_payload(
                "Ollama regenerate day repair prompt",
                "repair_prompt",
                repair_prompt,
                log_context,
            )

            repair_response = self._call_ollama(repair_prompt)
            self._log_llm_payload(
                "Ollama raw regenerate day repair response",
                "raw_repair_response",
                repair_response,
                log_context,
            )
            repaired = parse_regenerate_day_response(repair_response, request.day_number)
            log_context["repair_succeeded"] = True
            return repaired

    def _regenerate_item_with_ollama(
        self,
        request: RegenerateItemRequest,
        log_context: dict[str, Any],
    ) -> RegenerateItemResponse:
        destination_context = self._get_destination_context_for(
            request.trip.destination,
            log_context,
        )
        rag_chunks = self._get_partial_rag_chunks(request, log_context)
        prompt = build_regenerate_item_prompt(
            request,
            destination_context=destination_context,
            rag_chunks=rag_chunks,
        )
        self._log_llm_payload("Ollama regenerate item prompt", "prompt", prompt, log_context)

        llm_response = self._call_ollama(prompt)
        self._log_llm_payload(
            "Ollama raw regenerate item response",
            "raw_llm_response",
            llm_response,
            log_context,
        )

        try:
            return parse_regenerate_item_response(llm_response)
        except LLMResponseParseError as exc:
            self._record_generation_error(log_context, exc)
            if not self._repair_is_enabled():
                raise

            log_context["repair_attempted"] = True
            repair_prompt = build_regenerate_item_repair_prompt(
                request=request,
                invalid_response_text=llm_response,
                validation_error=self._validation_error_for_prompt(exc),
                destination_context=destination_context,
                rag_chunks=rag_chunks,
            )
            self._log_llm_payload(
                "Ollama regenerate item repair prompt",
                "repair_prompt",
                repair_prompt,
                log_context,
            )

            repair_response = self._call_ollama(repair_prompt)
            self._log_llm_payload(
                "Ollama raw regenerate item repair response",
                "raw_repair_response",
                repair_response,
                log_context,
            )
            repaired = parse_regenerate_item_response(repair_response)
            log_context["repair_succeeded"] = True
            return repaired

    def _optimize_budget_day_with_ollama(
        self,
        request: OptimizeBudgetDayRequest,
        log_context: dict[str, Any],
    ) -> BudgetOptimizationProposalResponse:
        prompt = build_optimize_budget_day_prompt(request)
        self._log_llm_payload("Ollama budget optimization prompt", "prompt", prompt, log_context)

        llm_response = self._call_ollama(prompt)
        self._log_llm_payload(
            "Ollama raw budget optimization response",
            "raw_llm_response",
            llm_response,
            log_context,
        )
        return parse_budget_optimization_response(llm_response, request.day_number)

    def _call_ollama(self, prompt: str) -> str:
        payload = {
            "model": self._settings.ollama_model,
            "prompt": prompt,
            "stream": False,
            "options": {
                "temperature": self._settings.ollama_temperature,
                "num_predict": self._settings.ollama_num_predict,
            },
        }

        response = self._post_to_ollama(payload)
        if response.status_code < 200 or response.status_code >= 300:
            raise OllamaClientError(f"Ollama API returned HTTP {response.status_code}")

        try:
            ollama_body = response.json()
        except ValueError as exc:
            raise OllamaClientError("Ollama API returned invalid JSON") from exc

        llm_response = ollama_body.get("response")
        if not isinstance(llm_response, str) or not llm_response.strip():
            raise OllamaClientError("Ollama API response is missing a non-empty 'response' field")

        return llm_response

    def _parse_and_validate(
        self,
        request: GenerateItineraryRequest,
        response_text: str,
    ) -> ItineraryResponse:
        itinerary = parse_itinerary_response(response_text, expected_days=request.days)
        result = self._validator.validate(request, itinerary)
        if result.warnings:
            logger.warning(
                "Itinerary personalization validation warnings",
                extra={
                    "trip_id": str(request.trip_id),
                    "validation_warning_codes": [warning.code for warning in result.warnings],
                },
            )
        return itinerary

    def _post_to_ollama(self, payload: dict) -> httpx.Response:
        endpoint = f"{self._settings.ollama_base_url.rstrip('/')}/api/generate"
        timeout = self._settings.ollama_timeout_seconds

        if self._http_client is not None:
            return self._http_client.post(endpoint, json=payload, timeout=timeout)

        with httpx.Client(timeout=timeout) as client:
            return client.post(endpoint, json=payload)

    def _repair_is_enabled(self) -> bool:
        return self._settings.ollama_repair_enabled and self._settings.ollama_repair_attempts > 0

    def _base_log_context(self, request: GenerateItineraryRequest) -> dict[str, Any]:
        return {
            "trip_id": str(request.trip_id),
            "destination": request.destination,
            "days": request.days,
            "pace": request.pace,
            "generator_mode": "ollama",
            "model": self._settings.ollama_model,
            "generation_duration_ms": None,
            "repair_enabled": self._repair_is_enabled(),
            "repair_attempted": False,
            "repair_succeeded": False,
            "fallback_used": False,
            "destination_context_used": False,
            "rag_enabled": self._settings.rag_enabled,
            "rag_results_count": 0,
            "rag_search_failed": False,
            "validation_error_code": None,
            "validation_error_message": None,
        }

    def _base_partial_log_context(
        self,
        request: RegenerateDayRequest,
        action: str,
    ) -> dict[str, Any]:
        context = {
            "trip_id": str(request.trip.id),
            "destination": request.trip.destination,
            "day_number": request.day_number,
            "action": action,
            "generator_mode": "ollama",
            "model": self._settings.ollama_model,
            "generation_duration_ms": None,
            "repair_enabled": self._repair_is_enabled(),
            "repair_attempted": False,
            "repair_succeeded": False,
            "fallback_used": False,
            "destination_context_used": False,
            "rag_enabled": self._settings.rag_enabled,
            "rag_results_count": 0,
            "rag_search_failed": False,
            "validation_error_code": None,
            "validation_error_message": None,
            "instruction_present": request.instruction is not None,
        }
        if isinstance(request, RegenerateItemRequest):
            context["item_index"] = request.item_index
        return context

    def _base_budget_optimization_log_context(
        self,
        request: OptimizeBudgetDayRequest,
    ) -> dict[str, Any]:
        return {
            "trip_id": str(request.trip.id),
            "destination": request.trip.destination,
            "day_number": request.day_number,
            "action": "budget_optimization_day",
            "generator_mode": "ollama",
            "model": self._settings.ollama_model,
            "generation_duration_ms": None,
            "repair_enabled": False,
            "repair_attempted": False,
            "repair_succeeded": False,
            "fallback_used": False,
            "destination_context_used": False,
            "rag_enabled": self._settings.rag_enabled,
            "rag_results_count": 0,
            "rag_search_failed": False,
            "validation_error_code": None,
            "validation_error_message": None,
            "instruction_present": request.instruction is not None,
            "target_reduction_amount": str(request.budget_context.target_reduction_amount),
        }

    def _get_destination_context(
        self,
        request: GenerateItineraryRequest,
        log_context: dict[str, Any],
    ) -> DestinationContext | None:
        return self._get_destination_context_for(request.destination, log_context)

    def _get_destination_context_for(
        self,
        destination: str,
        log_context: dict[str, Any],
    ) -> DestinationContext | None:
        if (
            not self._settings.destination_context_enabled
            or self._destination_knowledge_provider is None
        ):
            return None

        try:
            context = self._destination_knowledge_provider.get_context(destination)
        except Exception:
            logger.warning(
                "Destination context lookup failed; continuing without destination context",
                extra=log_context,
                exc_info=True,
            )
            return None

        log_context["destination_context_used"] = context is not None
        return context

    def _get_rag_chunks(
        self,
        request: GenerateItineraryRequest,
        log_context: dict[str, Any],
    ) -> list[KnowledgeSearchResult]:
        if not self._settings.rag_enabled or self._knowledge_search_service is None:
            return []

        query = self._build_rag_query(request)
        try:
            chunks = self._knowledge_search_service.search(
                destination=request.destination,
                interests=request.interests,
                query=query,
                top_k=self._settings.rag_top_k,
            )
        except Exception:
            log_context["rag_search_failed"] = True
            logger.warning(
                "RAG search failed; continuing without RAG context",
                extra=log_context,
                exc_info=True,
            )
            return []

        log_context["rag_search_failed"] = bool(
            getattr(self._knowledge_search_service, "last_search_failed", False)
        )
        log_context["rag_results_count"] = len(chunks)
        return chunks

    def _get_partial_rag_chunks(
        self,
        request: RegenerateDayRequest,
        log_context: dict[str, Any],
    ) -> list[KnowledgeSearchResult]:
        if not self._settings.rag_enabled or self._knowledge_search_service is None:
            return []

        query = self._build_partial_rag_query(request)
        try:
            chunks = self._knowledge_search_service.search(
                destination=request.trip.destination,
                interests=request.trip.interests,
                query=query,
                top_k=self._settings.rag_top_k,
            )
        except Exception:
            log_context["rag_search_failed"] = True
            logger.warning(
                "RAG search failed; continuing without RAG context",
                extra=log_context,
                exc_info=True,
            )
            return []

        log_context["rag_search_failed"] = bool(
            getattr(self._knowledge_search_service, "last_search_failed", False)
        )
        log_context["rag_results_count"] = len(chunks)
        return chunks

    def _build_rag_query(self, request: GenerateItineraryRequest) -> str:
        query_parts = [f"pace: {request.pace}", "travel itinerary"]
        if request.budget_amount is not None:
            query_parts.append(f"budget: {request.budget_amount} {request.budget_currency}")
        if request.interests:
            query_parts.append("interests: " + ", ".join(request.interests))
        return " | ".join(query_parts)

    def _build_partial_rag_query(self, request: RegenerateDayRequest) -> str:
        query_parts = [
            f"destination: {request.trip.destination}",
            f"pace: {request.trip.pace}",
            f"replace day: {request.day_number}",
        ]
        if request.instruction:
            query_parts.append(f"instruction: {request.instruction}")
        if request.trip.interests:
            query_parts.append("interests: " + ", ".join(request.trip.interests))
        selected_day = request.selected_day()
        if selected_day is not None:
            query_parts.append(f"selected day title: {selected_day.title}")
        if isinstance(request, RegenerateItemRequest):
            query_parts.append(f"replace item index: {request.item_index}")
            selected_item = request.selected_item()
            if selected_item is not None:
                query_parts.append(f"selected item: {selected_item.name} {selected_item.type}")
        if request.user_preferences is not None:
            styles = request.user_preferences.travel_styles
            food = request.user_preferences.food_preferences
            avoid = request.user_preferences.avoid
            if styles:
                query_parts.append("travel styles: " + ", ".join(styles))
            if food:
                query_parts.append("food preferences: " + ", ".join(food))
            if avoid:
                query_parts.append("avoid: " + ", ".join(avoid))
        return " | ".join(query_parts)

    def _record_generation_error(
        self,
        log_context: dict[str, Any],
        exc: BaseException,
    ) -> None:
        log_context["validation_error_code"] = getattr(exc, "code", None)
        log_context["validation_error_message"] = str(exc)

    def _validation_error_for_prompt(self, exc: BaseException) -> str:
        code = getattr(exc, "code", None)
        if code:
            return f"{code}: {exc}"
        return str(exc)

    def _log_llm_payload(
        self,
        message: str,
        field_name: str,
        payload: str,
        log_context: dict[str, Any],
    ) -> None:
        if not self._settings.allow_llm_payload_logging:
            return

        logger.info(message, extra={**log_context, field_name: payload})

    def _duration_ms(self, started_at: float) -> int:
        return int((time.monotonic() - started_at) * 1000)
