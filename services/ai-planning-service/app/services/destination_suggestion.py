import json
import logging
import re
from typing import Protocol

import httpx
from pydantic import ValidationError

from app.config import Settings
from app.core.errors import ItineraryGenerationError
from app.schemas.destination_suggestion import (
    DestinationBudgetEstimate,
    DestinationConcern,
    DestinationSuggestion,
    DestinationSuggestionMode,
    DestinationSuggestionRequest,
    DestinationSuggestionResponse,
    DestinationTripPreview,
)
from app.services.prompt_builder import build_destination_suggestion_prompt

logger = logging.getLogger(__name__)


class DestinationSuggestionGenerator(Protocol):
    def suggest(self, request: DestinationSuggestionRequest) -> DestinationSuggestionResponse: ...


class MockDestinationSuggestionGenerator:
    def suggest(self, request: DestinationSuggestionRequest) -> DestinationSuggestionResponse:
        destinations = self._destinations(request)
        count = request.constraints.suggestion_count
        suggestions = [
            self._suggestion(name, request, index)
            for index, name in enumerate(destinations[:count])
        ]
        language = request.output_language
        return DestinationSuggestionResponse(
            sessionTitle=_text(language, "title"),
            suggestions=suggestions,
            followUpQuestions=[_text(language, "follow_up")],
            warnings=[_text(language, "warning")],
        )

    def _destinations(self, request: DestinationSuggestionRequest) -> list[str]:
        text = request.prompt.casefold()
        if request.refinement is not None:
            text += " " + request.refinement.instruction.casefold()

        if request.mode == DestinationSuggestionMode.REFINE:
            if _mentions(text, "cheaper", "cheap", "budget"):
                choices = ["Brno", "Kraków", "Budapest", "Ljubljana", "Valencia"]
            elif _mentions(text, "warmer", "warm", "sun"):
                choices = ["Valencia", "Naples", "Lisbon", "Split", "Nice"]
            elif _mentions(text, "nature", "mountain", "outdoor"):
                choices = ["Ljubljana", "Salzburg", "Innsbruck", "Split", "Brno"]
            elif _mentions(text, "city", "museum", "culture"):
                choices = ["Vienna", "Florence", "Paris", "Budapest", "Kraków"]
            else:
                choices = ["Ljubljana", "Valencia", "Vienna", "Kraków", "Salzburg"]
        elif request.mode == DestinationSuggestionMode.SURPRISE:
            choices = self._surprise_destinations(request)
        elif _mentions(text, "mountain", "nature", "hiking"):
            choices = ["Salzburg", "Ljubljana", "Innsbruck", "Split", "Brno"]
        elif _mentions(text, "cheap", "weekend", "budget"):
            choices = ["Kraków", "Budapest", "Brno", "Ljubljana", "Valencia"]
        elif _mentions(text, "museum", "culture", "art"):
            choices = ["Vienna", "Paris", "Florence", "Kraków", "Lisbon"]
        elif _mentions(text, "beach", "sea", "coast"):
            choices = ["Valencia", "Nice", "Split", "Lisbon", "Naples"]
        elif _mentions(text, "warm", "food", "romantic"):
            choices = ["Valencia", "Naples", "Lisbon", "Split", "Florence"]
        else:
            choices = ["Valencia", "Ljubljana", "Vienna", "Kraków", "Lisbon"]

        visited = {trip.destination.casefold() for trip in request.previous_trips}
        if request.constraints.avoid_previously_visited:
            choices = [choice for choice in choices if choice.casefold() not in visited]
        return _unique(choices + ["Valencia", "Ljubljana", "Vienna", "Kraków", "Salzburg"])

    def _surprise_destinations(self, request: DestinationSuggestionRequest) -> list[str]:
        styles: list[str] = []
        if request.user_context and request.user_context.preferences:
            preferences = request.user_context.preferences
            styles = preferences.travel_styles + preferences.food_preferences
        style_text = " ".join(styles).casefold()
        visited = {trip.destination.casefold() for trip in request.previous_trips}
        choices: list[str] = []
        if "prague" in visited:
            choices += ["Vienna", "Kraków", "Ljubljana"]
        if _mentions(style_text, "food", "local"):
            choices += ["Valencia", "Naples", "Lisbon"]
        if _mentions(style_text, "nature", "mountain", "outdoor"):
            choices += ["Salzburg", "Ljubljana", "Innsbruck"]
        choices += ["Valencia", "Ljubljana", "Vienna", "Kraków", "Lisbon"]
        if request.constraints.avoid_previously_visited:
            choices = [choice for choice in choices if choice.casefold() not in visited]
        return _unique(choices)

    def _suggestion(
        self,
        city: str,
        request: DestinationSuggestionRequest,
        index: int,
    ) -> DestinationSuggestion:
        info = _DESTINATIONS[city]
        duration = request.trip_context.duration_days if request.trip_context else None
        duration = duration or info["days"]
        currency = "EUR"
        requested_budget = None
        if request.trip_context and request.trip_context.budget:
            currency = request.trip_context.budget.currency
            requested_budget = request.trip_context.budget.amount
        amount = float(info["budget"])
        if requested_budget is not None:
            amount = min(amount, requested_budget * (0.72 + index * 0.04))
        language = request.output_language
        destination = f"{city}, {info['country']}"
        return DestinationSuggestion(
            id=f"{_slug(city)}-{_slug(str(info['country']))}",
            destination=destination,
            city=city,
            country=str(info["country"]),
            region=info.get("region"),
            matchScore=max(72, 91 - index * 4),
            recommendedDurationDays=duration,
            bestFor=list(info["tags"])[:3],
            estimatedBudget=DestinationBudgetEstimate(
                amount=round(amount, 2),
                currency=currency,
                confidence="medium",
            ),
            bestTimeToGo=_text(language, "best_time"),
            whyItFits=_format_text(language, "why", destination=destination),
            possibleDownsides=[_text(language, "downside")],
            tripPreview=DestinationTripPreview(
                title=_format_text(language, "preview_title", destination=city),
                summary=_format_text(language, "preview_summary", destination=city),
                sampleDay=list(info["sample"]),
            ),
            tags=list(info["tags"]),
            suggestedPromptForItinerary=_format_text(
                language,
                "itinerary_prompt",
                days=duration,
                destination=destination,
            ),
            concerns=[
                DestinationConcern(
                    type="budget_uncertainty",
                    message=_text(language, "concern"),
                )
            ],
        )


class OllamaDestinationSuggestionGenerator:
    def __init__(
        self,
        settings: Settings,
        fallback: DestinationSuggestionGenerator | None = None,
        http_client: httpx.Client | None = None,
    ) -> None:
        self._settings = settings
        self._fallback = fallback or MockDestinationSuggestionGenerator()
        self._http_client = http_client

    def suggest(self, request: DestinationSuggestionRequest) -> DestinationSuggestionResponse:
        try:
            response = self._call_ollama(build_destination_suggestion_prompt(request))
            parsed = _parse_json_object(response)
            return DestinationSuggestionResponse.model_validate(parsed)
        except (httpx.HTTPError, ValueError, ValidationError) as exc:
            if self._settings.ollama_fallback_to_mock:
                logger.warning(
                    "Ollama destination suggestions failed; using deterministic fallback",
                    extra={"mode": request.mode.value},
                )
                return self._fallback.suggest(request)
            raise ItineraryGenerationError("Failed to suggest destinations") from exc

    def _call_ollama(self, prompt: str) -> str:
        payload = {
            "model": self._settings.ollama_model,
            "prompt": prompt,
            "stream": False,
            "options": {
                "temperature": max(self._settings.ollama_temperature, 0.35),
                "num_predict": self._settings.ollama_num_predict,
            },
        }
        if self._http_client is not None:
            response = self._http_client.post("/api/generate", json=payload)
        else:
            with httpx.Client(
                base_url=self._settings.ollama_base_url.rstrip("/"),
                timeout=self._settings.ollama_timeout_seconds,
            ) as client:
                response = client.post("/api/generate", json=payload)
        response.raise_for_status()
        body = response.json()
        result = body.get("response")
        if not isinstance(result, str) or not result.strip():
            raise ValueError("Ollama response is missing response text")
        return result


def get_destination_suggestion_generator(settings: Settings) -> DestinationSuggestionGenerator:
    mode = settings.itinerary_generator_mode.strip().lower()
    if mode == "ollama":
        return OllamaDestinationSuggestionGenerator(settings)
    return MockDestinationSuggestionGenerator()


def _parse_json_object(value: str) -> dict[str, object]:
    text = value.strip()
    if text.startswith("```"):
        text = re.sub(r"^```(?:json)?\s*", "", text)
        text = re.sub(r"\s*```$", "", text)
    start = text.find("{")
    end = text.rfind("}")
    if start < 0 or end < start:
        raise ValueError("LLM response did not contain a JSON object")
    parsed = json.loads(text[start : end + 1])
    if not isinstance(parsed, dict):
        raise ValueError("LLM response must be a JSON object")
    return parsed


def _mentions(text: str, *values: str) -> bool:
    return any(value in text for value in values)


def _unique(values: list[str]) -> list[str]:
    return list(dict.fromkeys(values))


def _slug(value: str) -> str:
    return re.sub(r"[^a-z0-9]+", "-", value.casefold()).strip("-")


def _text(language: str, key: str) -> str:
    return _TEXT.get(language, _TEXT["en"])[key]


def _format_text(language: str, key: str, **values: object) -> str:
    return _text(language, key).format(**values)


_DESTINATIONS: dict[str, dict[str, object]] = {
    "Valencia": {
        "country": "Spain",
        "region": "Valencian Community",
        "budget": 520,
        "days": 4,
        "tags": ["food", "city_break", "warm", "architecture", "beach"],
        "sample": ["Central Market and old town", "Turia Gardens", "Paella dinner"],
    },
    "Naples": {
        "country": "Italy",
        "region": "Campania",
        "budget": 560,
        "days": 4,
        "tags": ["food", "history", "warm", "coast"],
        "sample": ["Historic centre", "Archaeology museum", "Neighborhood pizzeria"],
    },
    "Lisbon": {
        "country": "Portugal",
        "region": "Lisbon",
        "budget": 590,
        "days": 4,
        "tags": ["food", "city_break", "warm", "coast"],
        "sample": ["Alfama lanes", "Riverside tram", "Local food market"],
    },
    "Salzburg": {
        "country": "Austria",
        "region": "Salzburg",
        "budget": 540,
        "days": 3,
        "tags": ["mountains", "nature", "culture", "train"],
        "sample": ["Old town", "River walk", "Untersberg viewpoint"],
    },
    "Ljubljana": {
        "country": "Slovenia",
        "region": "Central Slovenia",
        "budget": 430,
        "days": 3,
        "tags": ["nature", "city_break", "food", "hidden_gem"],
        "sample": ["Riverside market", "Castle funicular", "Tivoli Park"],
    },
    "Innsbruck": {
        "country": "Austria",
        "region": "Tyrol",
        "budget": 610,
        "days": 3,
        "tags": ["mountains", "nature", "scenery", "train"],
        "sample": ["Golden Roof", "Nordkette cable car", "Old town dinner"],
    },
    "Kraków": {
        "country": "Poland",
        "region": "Lesser Poland",
        "budget": 360,
        "days": 3,
        "tags": ["budget", "weekend", "food", "culture"],
        "sample": ["Main Market Square", "Kazimierz", "Milk bar lunch"],
    },
    "Budapest": {
        "country": "Hungary",
        "region": "Central Hungary",
        "budget": 390,
        "days": 3,
        "tags": ["budget", "city_break", "food", "spa"],
        "sample": ["Castle district", "Danube promenade", "Thermal bath"],
    },
    "Brno": {
        "country": "Czechia",
        "region": "South Moravia",
        "budget": 310,
        "days": 2,
        "tags": ["budget", "weekend", "food", "hidden_gem"],
        "sample": ["Vegetable Market", "Špilberk Castle", "Moravian dinner"],
    },
    "Vienna": {
        "country": "Austria",
        "region": "Vienna",
        "budget": 570,
        "days": 3,
        "tags": ["museums", "culture", "architecture", "food"],
        "sample": ["Museum quarter", "Historic centre", "Coffee house"],
    },
    "Paris": {
        "country": "France",
        "region": "Île-de-France",
        "budget": 760,
        "days": 4,
        "tags": ["museums", "culture", "food", "romantic"],
        "sample": ["Left Bank walk", "Museum visit", "Neighborhood bistro"],
    },
    "Florence": {
        "country": "Italy",
        "region": "Tuscany",
        "budget": 650,
        "days": 4,
        "tags": ["museums", "culture", "food", "architecture"],
        "sample": ["Duomo district", "Uffizi Gallery", "Oltrarno dinner"],
    },
    "Nice": {
        "country": "France",
        "region": "Provence-Alpes-Côte d'Azur",
        "budget": 680,
        "days": 4,
        "tags": ["beach", "warm", "food", "coast"],
        "sample": ["Old Nice market", "Promenade", "Hilltop sunset"],
    },
    "Split": {
        "country": "Croatia",
        "region": "Dalmatia",
        "budget": 520,
        "days": 4,
        "tags": ["beach", "warm", "history", "nature"],
        "sample": ["Diocletian's Palace", "Marjan Park", "Waterfront dinner"],
    },
}

_TEXT = {
    "en": {
        "title": "Destination ideas for your next trip",
        "follow_up": "Would you prefer a historic city, nature, or the coast?",
        "warning": "Budgets are rough estimates and exclude live flight and hotel prices.",
        "best_time": "Spring or early autumn",
        "why": "{destination} fits your stated travel style, pace, and practical constraints.",
        "downside": "Transport and accommodation prices can change significantly.",
        "preview_title": "{destination} discovery escape",
        "preview_summary": "A balanced first look at local food, culture, and relaxed exploration.",
        "itinerary_prompt": "Create a {days}-day trip to {destination} with a balanced pace.",
        "concern": "Transport cost from your origin has not been verified.",
    },
    "es": {
        "title": "Ideas de destinos para tu próximo viaje",
        "follow_up": "¿Prefieres una ciudad histórica, naturaleza o la costa?",
        "warning": "Los presupuestos son estimaciones y no incluyen precios reales.",
        "best_time": "Primavera o principios de otoño",
        "why": "{destination} encaja con tu estilo, ritmo y restricciones prácticas.",
        "downside": "Los precios de transporte y alojamiento pueden variar.",
        "preview_title": "Escapada para descubrir {destination}",
        "preview_summary": "Una primera visita equilibrada con gastronomía y cultura local.",
        "itinerary_prompt": "Crea un viaje de {days} días a {destination} con ritmo equilibrado.",
        "concern": "No se ha verificado el coste del transporte desde tu origen.",
    },
    "uk": {
        "title": "Ідеї напрямків для вашої наступної подорожі",
        "follow_up": "Ви віддаєте перевагу історичному місту, природі чи узбережжю?",
        "warning": "Бюджети орієнтовні й не включають актуальні ціни на переліт і готель.",
        "best_time": "Весна або рання осінь",
        "why": "{destination} відповідає вашому стилю, темпу та практичним обмеженням.",
        "downside": "Ціни на транспорт і проживання можуть суттєво змінюватися.",
        "preview_title": "Подорож-знайомство з {destination}",
        "preview_summary": "Збалансоване знайомство з місцевою кухнею, культурою та містом.",
        "itinerary_prompt": "Створи {days}-денну подорож до {destination} у збалансованому темпі.",
        "concern": "Вартість транспорту з вашого міста не перевірена.",
    },
    "fr": {
        "title": "Des idées de destinations pour votre prochain voyage",
        "follow_up": "Préférez-vous une ville historique, la nature ou la côte ?",
        "warning": "Les budgets sont estimatifs et excluent les prix en temps réel.",
        "best_time": "Printemps ou début d'automne",
        "why": "{destination} correspond à votre style, votre rythme et vos contraintes.",
        "downside": "Les prix du transport et de l'hébergement peuvent varier.",
        "preview_title": "Escapade découverte à {destination}",
        "preview_summary": "Une première visite équilibrée entre cuisine locale et culture.",
        "itinerary_prompt": "Crée un voyage de {days} jours à {destination} à un rythme équilibré.",
        "concern": "Le coût du transport depuis votre origine n'a pas été vérifié.",
    },
}
