"""Template adaptation service (mock + Ollama modes).

Mirrors the itinerary generator design: a ``TemplateAdapter`` protocol with a
deterministic ``MockTemplateAdapter`` and an ``OllamaTemplateAdapter`` that
builds a strict-JSON prompt, parses/validates the response, attempts one repair
pass, and can fall back to the mock adapter.
"""

import logging
import time
from datetime import date, timedelta
from typing import Any, Protocol

import httpx

from app.config import Settings
from app.core.errors import ItineraryGenerationError
from app.observability import record_ai_repair_attempt
from app.schemas.itinerary import EstimatedCost
from app.schemas.template_adaptation import (
    AdaptationSummary,
    AdaptedDay,
    AdaptedItem,
    AdaptedItinerary,
    AdaptedPlace,
    TemplateAdaptationRequest,
    TemplateAdaptationResponse,
    TemplateDayInput,
)
from app.services.llm_response_parser import (
    LLMResponseParseError,
    parse_template_adaptation_response,
)
from app.services.prompt_builder import (
    build_template_adaptation_prompt,
    build_template_adaptation_repair_prompt,
)
from app.services.template_adaptation_validator import (
    TemplateAdaptationValidationError,
    TemplateAdaptationValidator,
)

logger = logging.getLogger(__name__)

_PLACEHOLDER_TITLE = "Flexible exploration day"


class TemplateAdapter(Protocol):
    def adapt(self, request: TemplateAdaptationRequest) -> TemplateAdaptationResponse: ...


class MockTemplateAdapter:
    """Deterministic template adaptation with no external calls.

    Preserves day/item structure, renames items deterministically, shifts dates
    from the target start date, and trims/extends days to the target duration.
    """

    def adapt(self, request: TemplateAdaptationRequest) -> TemplateAdaptationResponse:
        target = request.target
        currency = target.budget.currency if target.budget else _preferred_currency(request)
        source_days = sorted(request.template.days, key=lambda d: d.day_offset)

        adapted_days: list[AdaptedDay] = []
        used_source = min(len(source_days), target.duration_days)
        for index in range(used_source):
            adapted_days.append(
                self._adapt_day(source_days[index], index, target.destination, currency, target)
            )
        for index in range(used_source, target.duration_days):
            adapted_days.append(self._placeholder_day(index, target.destination, currency, target))

        major_changes = self._major_changes(request, len(source_days), target.duration_days)
        itinerary = AdaptedItinerary(
            title=f"{target.destination} trip",
            destination=target.destination,
            start_date=target.start_date,
            days=adapted_days,
        )
        summary = AdaptationSummary(
            source_duration_days=request.template.duration_days,
            target_duration_days=target.duration_days,
            preserved_structure=request.constraints.preserve_structure,
            changed_destination=True,
            major_changes=major_changes,
        )
        _localize_mock_adaptation(itinerary, summary, request.output_language)
        return TemplateAdaptationResponse(itinerary=itinerary, adaptation_summary=summary)

    def _adapt_day(
        self,
        source: TemplateDayInput,
        index: int,
        destination: str,
        currency: str,
        target: Any,
    ) -> AdaptedDay:
        items = [
            self._adapt_item(item, destination, currency)
            for item in source.items
            if item.name.strip()
        ]
        if not items:
            items = [self._placeholder_item(destination, currency)]
        title = source.title.strip() or f"Day {index + 1} in {destination}"
        return AdaptedDay(
            day_date=_day_date(target.start_date, index),
            title=f"{title} ({destination})" if destination.lower() not in title.lower() else title,
            items=items,
        )

    def _adapt_item(self, item: Any, destination: str, currency: str) -> AdaptedItem:
        place = None
        if item.place is not None and (item.place.name or item.place.category):
            place = AdaptedPlace(
                name=f"{destination} {item.place.name}" if item.place.name else None,
                category=item.place.category,
            )
        return AdaptedItem(
            name=f"{destination} version of {item.name}",
            type=item.type,
            description=item.description,
            time=item.time,
            startTime=item.start_time or item.time,
            endTime=item.end_time,
            place=place,
            estimatedCost=_adapt_cost(item.estimated_cost, currency),
            notes=item.notes or f"Adapted from the template for {destination}; verify details.",
        )

    def _placeholder_day(
        self, index: int, destination: str, currency: str, target: Any
    ) -> AdaptedDay:
        return AdaptedDay(
            day_date=_day_date(target.start_date, index),
            title=f"{_PLACEHOLDER_TITLE} in {destination}",
            items=[self._placeholder_item(destination, currency)],
        )

    def _placeholder_item(self, destination: str, currency: str) -> AdaptedItem:
        return AdaptedItem(
            name=f"Flexible exploration in {destination}",
            type="activity",
            startTime="10:00",
            notes="Added to extend the template to the requested duration; customize freely.",
        )

    def _major_changes(
        self,
        request: TemplateAdaptationRequest,
        source_days: int,
        target_days: int,
    ) -> list[str]:
        destination = request.target.destination
        changes = [f"Adapted the template structure to {destination}."]
        if target_days < source_days:
            changes.append(f"Trimmed the plan from {source_days} to {target_days} day(s).")
        elif target_days > source_days:
            changes.append(
                f"Extended the plan from {source_days} to {target_days} day(s) "
                "with flexible exploration days."
            )
        return changes


_ADAPTATION_TEXT = {
    "es": {
        "title": "Viaje a {destination}",
        "day": "Día {day} en {destination}",
        "item": "Experiencia local en {destination}",
        "note": "Adaptado al destino; verifica los detalles y la disponibilidad.",
        "change": "La estructura de la plantilla se adaptó a {destination}.",
    },
    "uk": {
        "title": "Подорож до {destination}",
        "day": "День {day} у {destination}",
        "item": "Місцевий досвід у {destination}",
        "note": "Адаптовано до напрямку; перевірте деталі та доступність.",
        "change": "Структуру шаблону адаптовано до {destination}.",
    },
    "fr": {
        "title": "Voyage à {destination}",
        "day": "Jour {day} à {destination}",
        "item": "Expérience locale à {destination}",
        "note": "Adapté à la destination ; vérifiez les détails et la disponibilité.",
        "change": "La structure du modèle a été adaptée à {destination}.",
    },
}


def _localize_mock_adaptation(
    itinerary: AdaptedItinerary, summary: AdaptationSummary, language: str
) -> None:
    text = _ADAPTATION_TEXT.get(language)
    if text is None:
        return
    destination = itinerary.destination
    itinerary.title = text["title"].format(destination=destination)
    for index, day in enumerate(itinerary.days, start=1):
        day.title = text["day"].format(day=index, destination=destination)
        for item in day.items:
            item.name = text["item"].format(destination=destination)
            item.description = text["note"]
            item.notes = text["note"]
    summary.major_changes = [text["change"].format(destination=destination)]


class OllamaTemplateAdapter:
    def __init__(
        self,
        settings: Settings,
        fallback_adapter: TemplateAdapter | None = None,
        http_client: httpx.Client | None = None,
    ) -> None:
        self._settings = settings
        self._fallback_adapter = fallback_adapter or MockTemplateAdapter()
        self._http_client = http_client
        self._validator = TemplateAdaptationValidator()

    def adapt(self, request: TemplateAdaptationRequest) -> TemplateAdaptationResponse:
        started_at = time.monotonic()
        log_context = {
            "destination": request.target.destination,
            "source_duration_days": request.template.duration_days,
            "target_duration_days": request.target.duration_days,
            "adapter_mode": "ollama",
            "model": self._settings.ollama_model,
            "repair_attempted": False,
            "repair_succeeded": False,
            "fallback_used": False,
        }
        try:
            response = self._adapt_with_ollama(request, log_context)
            log_context["duration_ms"] = int((time.monotonic() - started_at) * 1000)
            logger.info("Ollama template adaptation succeeded", extra=log_context)
            return response
        except (
            httpx.HTTPError,
            OllamaTemplateAdapterError,
            LLMResponseParseError,
            TemplateAdaptationValidationError,
        ) as exc:
            log_context["duration_ms"] = int((time.monotonic() - started_at) * 1000)
            if self._settings.template_adaptation_fallback_enabled:
                log_context["fallback_used"] = True
                logger.warning(
                    "Ollama template adaptation failed; falling back to mock adapter",
                    extra=log_context,
                    exc_info=True,
                )
                fallback = self._fallback_adapter.adapt(request)
                fallback.adaptation_summary.fallback_used = True
                fallback.adaptation_summary.warnings.append(
                    "AI adaptation was unavailable; a deterministic adaptation was used."
                )
                return fallback
            logger.error("Ollama template adaptation failed", extra=log_context, exc_info=True)
            raise ItineraryGenerationError("Failed to adapt template") from exc

    def _adapt_with_ollama(
        self,
        request: TemplateAdaptationRequest,
        log_context: dict[str, Any],
    ) -> TemplateAdaptationResponse:
        prompt = build_template_adaptation_prompt(request)
        llm_response = self._call_ollama(prompt)
        try:
            return self._parse_and_validate(request, llm_response)
        except (LLMResponseParseError, TemplateAdaptationValidationError) as exc:
            if not self._repair_is_enabled():
                raise
            log_context["repair_attempted"] = True
            record_ai_repair_attempt("adapt_template", "attempted")
            repair_prompt = build_template_adaptation_repair_prompt(
                request=request,
                invalid_response_text=llm_response,
                validation_error=str(exc),
            )
            repair_response = self._call_ollama(repair_prompt)
            repaired = self._parse_and_validate(request, repair_response)
            log_context["repair_succeeded"] = True
            record_ai_repair_attempt("adapt_template", "success")
            return repaired

    def _parse_and_validate(
        self,
        request: TemplateAdaptationRequest,
        response_text: str,
    ) -> TemplateAdaptationResponse:
        response = parse_template_adaptation_response(
            response_text, expected_days=request.target.duration_days
        )
        result = self._validator.validate(request, response.itinerary)
        _merge_warnings(response.adaptation_summary, result.warnings)
        return response

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
        timeout = self._settings.template_adaptation_timeout_seconds
        endpoint = f"{self._settings.ollama_base_url.rstrip('/')}/api/generate"
        if self._http_client is not None:
            response = self._http_client.post(endpoint, json=payload, timeout=timeout)
        else:
            with httpx.Client(timeout=timeout) as client:
                response = client.post(endpoint, json=payload)

        if response.status_code < 200 or response.status_code >= 300:
            raise OllamaTemplateAdapterError(f"Ollama API returned HTTP {response.status_code}")
        try:
            body = response.json()
        except ValueError as exc:
            raise OllamaTemplateAdapterError("Ollama API returned invalid JSON") from exc
        llm_response = body.get("response")
        if not isinstance(llm_response, str) or not llm_response.strip():
            raise OllamaTemplateAdapterError(
                "Ollama API response is missing a non-empty 'response' field"
            )
        return llm_response

    def _repair_is_enabled(self) -> bool:
        return self._settings.ollama_repair_enabled and self._settings.ollama_repair_attempts > 0


class OllamaTemplateAdapterError(RuntimeError):
    """Raised when the Ollama API cannot provide a usable adaptation response."""


def get_template_adapter(settings: Settings) -> TemplateAdapter:
    mode = settings.template_adaptation_mode.strip().lower() or "mock"
    if mode == "mock":
        return MockTemplateAdapter()
    if mode == "ollama":
        if not settings.ollama_base_url.strip():
            raise ValueError("OLLAMA_BASE_URL is required when AI_TEMPLATE_ADAPTATION_MODE=ollama")
        if not settings.ollama_model.strip():
            raise ValueError("OLLAMA_MODEL is required when AI_TEMPLATE_ADAPTATION_MODE=ollama")
        return OllamaTemplateAdapter(settings=settings)
    raise ValueError(
        f"Unknown AI_TEMPLATE_ADAPTATION_MODE {settings.template_adaptation_mode!r}; "
        "expected 'mock' or 'ollama'"
    )


def validate_adaptation(
    request: TemplateAdaptationRequest,
    response: TemplateAdaptationResponse,
) -> None:
    """Validate a (mock or already-parsed) adaptation and attach warnings."""
    result = TemplateAdaptationValidator().validate(request, response.itinerary)
    _merge_warnings(response.adaptation_summary, result.warnings)


def _merge_warnings(summary: AdaptationSummary, warnings: list[str]) -> None:
    existing = {warning.casefold() for warning in summary.warnings}
    for warning in warnings:
        if warning.casefold() not in existing:
            summary.warnings.append(warning)
            existing.add(warning.casefold())


def _adapt_cost(cost: EstimatedCost | None, currency: str) -> EstimatedCost | None:
    if cost is None or cost.amount is None:
        return None
    return EstimatedCost(
        amount=cost.amount,
        currency=cost.currency or currency,
        category=cost.category or "other",
        confidence="low",
        source="ai",
        note="Estimated from the template; verify current price.",
    )


def _day_date(start: date, offset: int) -> date:
    return start + timedelta(days=offset)


def _preferred_currency(request: TemplateAdaptationRequest) -> str:
    context = request.context
    if context and context.user_preferences:
        currency = context.user_preferences.get("preferredCurrency")
        if isinstance(currency, str) and len(currency.strip()) == 3:
            return currency.strip().upper()
    return "EUR"
