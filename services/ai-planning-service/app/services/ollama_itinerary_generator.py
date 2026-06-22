import logging

import httpx

from app.config import Settings
from app.core.errors import ItineraryGenerationError
from app.schemas.itinerary import GenerateItineraryRequest, ItineraryResponse
from app.services.itinerary_generator import ItineraryGenerator, MockItineraryGenerator
from app.services.llm_response_parser import LLMResponseParseError, parse_itinerary_response
from app.services.prompt_builder import build_itinerary_prompt

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

    def generate(self, request: GenerateItineraryRequest) -> ItineraryResponse:
        try:
            return self._generate_with_ollama(request)
        except (httpx.HTTPError, OllamaClientError, LLMResponseParseError) as exc:
            if self._settings.ollama_fallback_to_mock:
                logger.warning(
                    "Ollama itinerary generation failed; falling back to mock generator",
                    exc_info=True,
                )
                return self._fallback_generator.generate(request)

            logger.error("Ollama itinerary generation failed", exc_info=True)
            raise ItineraryGenerationError("Failed to generate itinerary") from exc

    def _generate_with_ollama(self, request: GenerateItineraryRequest) -> ItineraryResponse:
        payload = {
            "model": self._settings.ollama_model,
            "prompt": build_itinerary_prompt(request),
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

        return parse_itinerary_response(llm_response, expected_days=request.days)

    def _post_to_ollama(self, payload: dict) -> httpx.Response:
        endpoint = f"{self._settings.ollama_base_url.rstrip('/')}/api/generate"
        timeout = self._settings.ollama_timeout_seconds

        if self._http_client is not None:
            return self._http_client.post(endpoint, json=payload, timeout=timeout)

        with httpx.Client(timeout=timeout) as client:
            return client.post(endpoint, json=payload)
