import logging
import time
from typing import Any

import httpx

from app.config import Settings
from app.core.errors import ItineraryGenerationError
from app.schemas.itinerary import GenerateItineraryRequest, ItineraryResponse
from app.services.itinerary_generator import ItineraryGenerator, MockItineraryGenerator
from app.services.itinerary_validator import ItineraryValidationError, ItineraryValidator
from app.services.llm_response_parser import LLMResponseParseError, parse_itinerary_response
from app.services.prompt_builder import build_itinerary_prompt, build_repair_prompt

logger = logging.getLogger(__name__)


class OllamaClientError(RuntimeError):
    """Raised when the Ollama API cannot provide a usable response."""


class OllamaItineraryGenerator:
    def __init__(
        self,
        settings: Settings,
        fallback_generator: ItineraryGenerator | None = None,
        http_client: httpx.Client | None = None,
    ) -> None:
        self._settings = settings
        self._fallback_generator = fallback_generator or MockItineraryGenerator()
        self._http_client = http_client
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

    def _generate_with_ollama(
        self,
        request: GenerateItineraryRequest,
        log_context: dict[str, Any],
    ) -> ItineraryResponse:
        prompt = build_itinerary_prompt(request)
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
        self._validator.validate(request, itinerary)
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
            "validation_error_code": None,
            "validation_error_message": None,
        }

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
