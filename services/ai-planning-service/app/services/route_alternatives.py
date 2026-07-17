from __future__ import annotations

import logging
from decimal import Decimal
from typing import Protocol

import httpx
from pydantic import ValidationError

from app.config import Settings
from app.core.errors import ItineraryGenerationError
from app.schemas.itinerary import (
    Coordinates,
    RouteCost,
    RouteLeg,
    RoutePlace,
    RoutePreferences,
    RouteStop,
    TripRoute,
)
from app.schemas.route_alternatives import (
    RouteAlternative,
    RouteAlternativeBudgetEstimate,
    RouteAlternativeComparisonSummary,
    RouteAlternativePersonalizationFit,
    RouteAlternativeRequest,
    RouteAlternativeResponse,
    RouteAlternativeScores,
)
from app.services.llm_response_parser import (
    LLMResponseParseError,
    parse_route_alternatives_response,
)
from app.services.prompt_builder import (
    build_route_alternatives_prompt,
    build_route_alternatives_repair_prompt,
)

logger = logging.getLogger(__name__)


class RouteAlternativeGenerator(Protocol):
    def suggest(self, request: RouteAlternativeRequest) -> RouteAlternativeResponse: ...


class MockRouteAlternativeGenerator:
    def suggest(self, request: RouteAlternativeRequest) -> RouteAlternativeResponse:
        alternatives = self._alternatives(request)
        count = max(1, min(request.suggestion_count, len(alternatives)))
        selected = [_with_personalization(item, request) for item in alternatives[:count]]
        return RouteAlternativeResponse(
            sessionTitle=_text(request.output_language, "session_title"),
            alternatives=selected,
            comparisonSummary=_comparison(selected),
            followUpQuestions=[_text(request.output_language, "follow_up")],
            warnings=[_text(request.output_language, "warning")],
        )

    def _alternatives(self, request: RouteAlternativeRequest) -> list[RouteAlternative]:
        text = _request_text(request)
        if request.current_route is not None and request.current_route.stops:
            return _current_route_variants(request)
        if _mentions(text, "spain", "barcelona", "madrid", "valencia", "granada"):
            return _spain_alternatives(request)
        if _mentions(text, "austria", "bratislava", "vienna", "salzburg", "hallstatt"):
            return _austria_alternatives(request)
        return _europe_alternatives(request)


class OllamaRouteAlternativeGenerator:
    def __init__(
        self,
        settings: Settings,
        fallback: RouteAlternativeGenerator | None = None,
        http_client: httpx.Client | None = None,
    ) -> None:
        self._settings = settings
        self._fallback = fallback or MockRouteAlternativeGenerator()
        self._http_client = http_client

    def suggest(self, request: RouteAlternativeRequest) -> RouteAlternativeResponse:
        try:
            response = self._call_ollama(build_route_alternatives_prompt(request))
            return parse_route_alternatives_response(response)
        except (httpx.HTTPError, ValueError, ValidationError, LLMResponseParseError) as exc:
            if self._settings.ollama_repair_enabled and not isinstance(exc, httpx.HTTPError):
                try:
                    repair_prompt = build_route_alternatives_repair_prompt(
                        request,
                        response if "response" in locals() else "",
                        str(exc),
                    )
                    repaired = self._call_ollama(repair_prompt)
                    return parse_route_alternatives_response(repaired)
                except Exception:
                    logger.warning("Ollama route alternatives repair failed", exc_info=True)
            if self._settings.ollama_fallback_to_mock:
                logger.warning(
                    "Ollama route alternatives failed; using deterministic fallback",
                    extra={"suggestion_count": request.suggestion_count},
                )
                return self._fallback.suggest(request)
            raise ItineraryGenerationError("Failed to suggest route alternatives") from exc

    def _call_ollama(self, prompt: str) -> str:
        payload = {
            "model": self._settings.ollama_model,
            "prompt": prompt,
            "stream": False,
            "options": {
                "temperature": max(self._settings.ollama_temperature, 0.25),
                "num_predict": max(self._settings.ollama_num_predict, 3072),
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


def get_route_alternative_generator(settings: Settings) -> RouteAlternativeGenerator:
    mode = settings.itinerary_generator_mode.strip().lower()
    if mode == "ollama":
        return OllamaRouteAlternativeGenerator(settings)
    return MockRouteAlternativeGenerator()


def _with_personalization(
    alternative: RouteAlternative, request: RouteAlternativeRequest
) -> RouteAlternative:
    summary = request.planning_constraints.personalization if request.planning_constraints else None
    if summary is None:
        return alternative
    reasons: list[str] = []
    concerns: list[str] = []
    modes = [leg.mode.casefold() for leg in alternative.route.legs]
    if "train" in {mode.casefold() for mode in summary.transport_bias} and "train" in modes:
        reasons.append("Mostly train-based, matching your saved transport preference.")
    if summary.walking_tolerance in {"low", "moderate"}:
        reasons.append("Balances route movement with your walking tolerance.")
    if summary.budget_comfort == "low" and alternative.scores.budget_fit < 70:
        concerns.append("Its estimated budget may be above your usual comfort range.")
    if not reasons:
        reasons.append("Balances your saved preferences with the route you requested.")
    return alternative.model_copy(
        update={
            "personalization_fit": RouteAlternativePersonalizationFit(
                score=alternative.scores.overall_fit,
                reasons=reasons[:3],
                concerns=concerns[:2],
            )
        }
    )


def _austria_alternatives(request: RouteAlternativeRequest) -> list[RouteAlternative]:
    mode = _primary_mode(request, default="train")
    car_mode = _car_mode(request)
    nature_mode = car_mode if car_mode and mode not in {"train"} else mode
    return [
        _alternative(
            request,
            "classic-austria-train-route",
            _title(request, "classic_austria"),
            _summary(request, "classic_austria"),
            [
                _stop("stop_1", "Vienna", "Austria", 2, 48.2082, 16.3738, "hotel"),
                _stop("stop_2", "Salzburg", "Austria", 2, 47.8095, 13.0550, "guesthouse"),
                _stop("stop_3", "Hallstatt", "Austria", 1, 47.5622, 13.6493, "guesthouse"),
            ],
            mode,
            [70, 150, 150],
            [18, 35, 20],
            RouteAlternativeScores(
                overallFit=88,
                budgetFit=_budget_score(request, 650),
                timeEfficiency=70,
                relaxation=65,
                nature=80,
                culture=90,
                transportSimplicity=82 if mode == "train" else 72,
                policyCompliance=_policy_score(request),
            ),
            estimated_budget=650,
            difficulty="balanced",
            best_for=["culture", "nature", "train travel"],
        ),
        _alternative(
            request,
            "relaxed-two-city-route",
            _title(request, "relaxed_two_city"),
            _summary(request, "relaxed_two_city"),
            [
                _stop("stop_1", "Vienna", "Austria", 2, 48.2082, 16.3738, "hotel"),
                _stop("stop_2", "Salzburg", "Austria", 3, 47.8095, 13.0550, "guesthouse"),
            ],
            mode,
            [70, 150],
            [18, 35],
            RouteAlternativeScores(
                overallFit=84,
                budgetFit=_budget_score(request, 560),
                timeEfficiency=82,
                relaxation=88,
                nature=67,
                culture=88,
                transportSimplicity=90,
                policyCompliance=_policy_score(request),
            ),
            estimated_budget=560,
            difficulty="relaxed",
            best_for=["relaxed pace", "culture", "simple transfers"],
        ),
        _alternative(
            request,
            "nature-heavy-route",
            _title(request, "nature_heavy"),
            _summary(request, "nature_heavy"),
            [
                _stop("stop_1", "Graz", "Austria", 1, 47.0707, 15.4395, "guesthouse"),
                _stop("stop_2", "Hallstatt", "Austria", 2, 47.5622, 13.6493, "cabin"),
                _stop("stop_3", "Salzburg", "Austria", 2, 47.8095, 13.0550, "guesthouse"),
            ],
            nature_mode,
            [155, 180, 150],
            [32, 38, 20],
            RouteAlternativeScores(
                overallFit=82,
                budgetFit=_budget_score(request, 620),
                timeEfficiency=62,
                relaxation=58,
                nature=94,
                culture=72,
                transportSimplicity=68,
                policyCompliance=_policy_score(request),
            ),
            estimated_budget=620,
            difficulty="intense",
            best_for=["nature", "hiking", "old towns"],
        ),
    ]


def _spain_alternatives(request: RouteAlternativeRequest) -> list[RouteAlternative]:
    mode = _primary_mode(request, default="train")
    return [
        _alternative(
            request,
            "barcelona-valencia-madrid",
            "Barcelona, Valencia and Madrid",
            "A practical city route using Spain's main rail spine.",
            [
                _stop("stop_1", "Barcelona", "Spain", 2, 41.3874, 2.1686, "hotel"),
                _stop("stop_2", "Valencia", "Spain", 1, 39.4699, -0.3763, "hotel"),
                _stop("stop_3", "Madrid", "Spain", 2, 40.4168, -3.7038, "hotel"),
            ],
            mode,
            [30, 180, 120],
            [4, 32, 28],
            RouteAlternativeScores(overallFit=86, nature=55, culture=92),
            estimated_budget=680,
            difficulty="balanced",
            best_for=["cities", "food", "train travel"],
        ),
        _alternative(
            request,
            "barcelona-girona-costa-brava",
            "Barcelona, Girona and Costa Brava",
            "A slower Catalonia route with old towns and coast.",
            [
                _stop("stop_1", "Barcelona", "Spain", 2, 41.3874, 2.1686, "hotel"),
                _stop("stop_2", "Girona", "Spain", 1, 41.9794, 2.8214, "guesthouse"),
                _stop("stop_3", "Costa Brava", "Spain", 2, 41.887, 3.184, "apartment"),
            ],
            mode,
            [30, 45, 70],
            [4, 12, 14],
            RouteAlternativeScores(overallFit=82, relaxation=82, nature=82, culture=76),
            estimated_budget=610,
            difficulty="relaxed",
            best_for=["coast", "old towns", "short transfers"],
        ),
        _alternative(
            request,
            "madrid-toledo-granada",
            "Madrid, Toledo and Granada",
            "A culture-heavy route with one longer southbound transfer.",
            [
                _stop("stop_1", "Madrid", "Spain", 2, 40.4168, -3.7038, "hotel"),
                _stop("stop_2", "Toledo", "Spain", 1, 39.8628, -4.0273, "guesthouse"),
                _stop("stop_3", "Granada", "Spain", 2, 37.1773, -3.5986, "hotel"),
            ],
            mode,
            [30, 35, 240],
            [4, 12, 45],
            RouteAlternativeScores(overallFit=80, timeEfficiency=62, culture=96),
            estimated_budget=640,
            difficulty="intense",
            best_for=["history", "culture", "architecture"],
        ),
    ]


def _europe_alternatives(request: RouteAlternativeRequest) -> list[RouteAlternative]:
    mode = _primary_mode(request, default="train")
    return [
        _alternative(
            request,
            "vienna-salzburg-munich",
            "Vienna, Salzburg and Munich",
            "A classic Central Europe route with simple train transfers.",
            [
                _stop("stop_1", "Vienna", "Austria", 2, 48.2082, 16.3738, "hotel"),
                _stop("stop_2", "Salzburg", "Austria", 1, 47.8095, 13.0550, "guesthouse"),
                _stop("stop_3", "Munich", "Germany", 2, 48.1351, 11.5820, "hotel"),
            ],
            mode,
            [70, 150, 110],
            [18, 35, 28],
            RouteAlternativeScores(overallFit=84, culture=88, transportSimplicity=86),
            estimated_budget=690,
            difficulty="balanced",
            best_for=["train travel", "culture", "old towns"],
        ),
        _alternative(
            request,
            "prague-dresden-berlin",
            "Prague, Dresden and Berlin",
            "A city-forward route with strong rail links and varied culture.",
            [
                _stop("stop_1", "Prague", "Czechia", 2, 50.0755, 14.4378, "hotel"),
                _stop("stop_2", "Dresden", "Germany", 1, 51.0504, 13.7373, "hotel"),
                _stop("stop_3", "Berlin", "Germany", 2, 52.52, 13.405, "hotel"),
            ],
            mode,
            [210, 140, 130],
            [24, 22, 24],
            RouteAlternativeScores(
                overallFit=82,
                budgetFit=_budget_score(request, 610),
                culture=90,
            ),
            estimated_budget=610,
            difficulty="balanced",
            best_for=["cities", "museums", "rail"],
        ),
        _alternative(
            request,
            "budapest-vienna-bratislava",
            "Budapest, Vienna and Bratislava",
            "A compact Danube route with shorter transfers and lower costs.",
            [
                _stop("stop_1", "Budapest", "Hungary", 2, 47.4979, 19.0402, "hotel"),
                _stop("stop_2", "Vienna", "Austria", 2, 48.2082, 16.3738, "hotel"),
                _stop("stop_3", "Bratislava", "Slovakia", 1, 48.1486, 17.1077, "hotel"),
            ],
            mode,
            [150, 150, 70],
            [20, 28, 12],
            RouteAlternativeScores(
                overallFit=80,
                budgetFit=_budget_score(request, 520),
                relaxation=76,
            ),
            estimated_budget=520,
            difficulty="balanced",
            best_for=["budget", "short transfers", "culture"],
        ),
    ]


def _current_route_variants(request: RouteAlternativeRequest) -> list[RouteAlternative]:
    route = request.current_route
    assert route is not None
    route_copy = TripRoute.model_validate(route.model_dump(by_alias=True, exclude_none=True))
    route_copy.preferences.preferred_modes = route_copy.preferences.preferred_modes or [
        _primary_mode(request, default="train")
    ]
    base = RouteAlternative(
        id="current-route-balanced",
        title=_title(request, "current_route"),
        summary=_summary(request, "current_route"),
        route=route_copy,
        scores=RouteAlternativeScores(overallFit=78, relaxation=65, nature=70, culture=70),
        estimatedBudget=_money(_estimate_budget(request, len(route_copy.stops))),
        estimatedTransferMinutes=_sum_minutes(route_copy),
        estimatedTransferCost=_money(_sum_cost(route_copy)),
        difficulty=_difficulty(route_copy, request.duration_days),
        bestFor=["current route", "familiar plan"],
        pros=[_text(request.output_language, "pro")],
        cons=[_text(request.output_language, "con")],
        warnings=[_text(request.output_language, "warning")],
        suggestedItineraryPrompt=_suggested_prompt(request, route_copy),
    )
    stops = route.stops[: max(1, min(2, len(route.stops)))]
    fewer = TripRoute(
        origin=route.origin,
        returnToOrigin=False,
        stops=stops,
        legs=[leg for leg in route.legs if leg.to_stop_id in {stop.id for stop in stops}],
        preferences=route.preferences,
    )
    return [
        base,
        RouteAlternative(
            id="fewer-stops-relaxed-route",
            title=_title(request, "fewer_stops"),
            summary=_summary(request, "fewer_stops"),
            route=fewer,
            scores=RouteAlternativeScores(overallFit=80, relaxation=88, transportSimplicity=86),
            estimatedBudget=_money(_estimate_budget(request, len(fewer.stops)) * Decimal("0.88")),
            estimatedTransferMinutes=_sum_minutes(fewer),
            estimatedTransferCost=_money(_sum_cost(fewer)),
            difficulty="relaxed",
            bestFor=["relaxed pace", "fewer stops"],
            pros=[_text(request.output_language, "pro")],
            cons=[_text(request.output_language, "con")],
            warnings=[_text(request.output_language, "warning")],
            suggestedItineraryPrompt=_suggested_prompt(request, fewer),
        ),
    ]


def _alternative(
    request: RouteAlternativeRequest,
    alt_id: str,
    title: str,
    summary: str,
    stops: list[RouteStop],
    mode: str,
    durations: list[int],
    costs: list[float],
    scores: RouteAlternativeScores,
    estimated_budget: float,
    difficulty: str,
    best_for: list[str],
) -> RouteAlternative:
    currency = _currency(request)
    route = TripRoute(
        origin=_origin(request),
        returnToOrigin=False,
        stops=stops,
        legs=_legs(request, stops, mode, durations, costs),
        preferences=RoutePreferences(
            preferredModes=[mode],
            avoidModes=_avoid_modes(request),
            carAvailable=_car_available(request),
            maxTransferHoursPerDay=_max_transfer_hours(request),
            tripStyles=_trip_styles(request),
        ),
    )
    return RouteAlternative(
        id=alt_id,
        title=title,
        summary=summary,
        route=route,
        scores=scores,
        estimatedBudget=_money(Decimal(str(estimated_budget)), currency),
        estimatedTransferMinutes=sum(durations),
        estimatedTransferCost=_money(Decimal(str(sum(costs))), currency),
        difficulty=difficulty,
        bestFor=best_for,
        pros=[_text(request.output_language, "pro")],
        cons=[_text(request.output_language, "con")],
        warnings=[_text(request.output_language, "warning")],
        suggestedItineraryPrompt=_suggested_prompt(request, route),
    )


def _legs(
    request: RouteAlternativeRequest,
    stops: list[RouteStop],
    mode: str,
    durations: list[int],
    costs: list[float],
) -> list[RouteLeg]:
    origin_name = _origin(request).name or "Origin"
    legs: list[RouteLeg] = []
    for index, stop in enumerate(stops):
        from_id = "origin" if index == 0 else stops[index - 1].id
        from_name = origin_name if index == 0 else stops[index - 1].destination
        duration = durations[min(index, len(durations) - 1)]
        cost = costs[min(index, len(costs) - 1)]
        legs.append(
            RouteLeg(
                id=f"leg_{index + 1}",
                fromStopId=from_id,
                toStopId=stop.id,
                fromName=from_name,
                toName=stop.destination,
                mode=mode,
                estimatedDurationMinutes=duration,
                estimatedDistanceKm=round(max(20, duration * 1.55), 1),
                estimatedCost=RouteCost(
                    amount=Decimal(str(cost)),
                    currency=_currency(request),
                    category="transport",
                    confidence="medium",
                    source="ai",
                    note=_text(request.output_language, "estimate_note"),
                ),
                notes=_text(request.output_language, "estimate_note"),
            )
        )
    return legs


def _stop(
    stop_id: str,
    city: str,
    country: str,
    nights: int,
    lat: float,
    lng: float,
    accommodation_hint: str,
) -> RouteStop:
    return RouteStop(
        id=stop_id,
        destination=city,
        city=city,
        country=country,
        nights=nights,
        coordinates=Coordinates(lat=lat, lng=lng),
        accommodationHint=accommodation_hint,
    )


def _comparison(alternatives: list[RouteAlternative]) -> RouteAlternativeComparisonSummary:
    cheapest = min(alternatives, key=lambda alt: _budget_amount(alt.estimated_budget), default=None)
    relaxed = max(alternatives, key=lambda alt: alt.scores.relaxation, default=None)
    nature = max(alternatives, key=lambda alt: alt.scores.nature, default=None)
    overall = max(alternatives, key=lambda alt: alt.scores.overall_fit, default=None)
    return RouteAlternativeComparisonSummary(
        cheapestAlternativeId=cheapest.id if cheapest else None,
        mostRelaxedAlternativeId=relaxed.id if relaxed else None,
        bestNatureAlternativeId=nature.id if nature else None,
        bestOverallAlternativeId=overall.id if overall else None,
    )


def _request_text(request: RouteAlternativeRequest) -> str:
    pieces = [request.prompt or ""]
    if request.refinement.instruction:
        pieces.append(request.refinement.instruction)
    if request.planning_constraints:
        pieces.extend(request.planning_constraints.trip_styles)
        pieces.extend(request.planning_constraints.interests)
    return " ".join(pieces).casefold()


def _mentions(text: str, *values: str) -> bool:
    return any(value in text for value in values)


def _origin(request: RouteAlternativeRequest) -> RoutePlace:
    if request.origin is not None:
        return request.origin
    return RoutePlace(
        name="Bratislava",
        country="Slovakia",
        coordinates=Coordinates(lat=48.1486, lng=17.1077),
    )


def _currency(request: RouteAlternativeRequest) -> str:
    if request.budget is not None and request.budget.currency:
        return request.budget.currency
    if request.planning_constraints and request.planning_constraints.budget:
        return request.planning_constraints.budget.currency
    return "EUR"


def _money(amount: Decimal, currency: str = "EUR") -> RouteAlternativeBudgetEstimate:
    return RouteAlternativeBudgetEstimate(
        amount=amount.quantize(Decimal("0.01")),
        currency=currency,
        confidence="medium",
    )


def _budget_amount(estimate: RouteAlternativeBudgetEstimate | None) -> Decimal:
    if estimate is None or estimate.amount is None:
        return Decimal("999999")
    return estimate.amount


def _budget_score(request: RouteAlternativeRequest, amount: float) -> int:
    budget = None
    if request.budget is not None:
        budget = request.budget.amount
    elif request.planning_constraints and request.planning_constraints.budget:
        raw = request.planning_constraints.budget.amount
        budget = Decimal(str(raw)) if raw is not None else None
    if budget is None or budget <= 0:
        return 70
    ratio = Decimal(str(amount)) / budget
    if ratio <= Decimal("1.0"):
        return 94
    if ratio <= Decimal("1.1"):
        return 70
    if ratio <= Decimal("1.3"):
        return 50
    return 25


def _policy_score(request: RouteAlternativeRequest) -> int:
    if request.planning_constraints is None:
        return 100
    if request.planning_constraints.blockers:
        return 35
    if request.planning_constraints.warnings:
        return 82
    return 100


def _preferred_modes(request: RouteAlternativeRequest) -> list[str]:
    modes: list[str] = []
    if request.planning_constraints:
        modes.extend(request.planning_constraints.transport.preferred_modes)
    if request.current_route:
        modes.extend(request.current_route.preferences.preferred_modes)
    return [_mode_token(mode) for mode in modes if _mode_token(mode)]


def _avoid_modes(request: RouteAlternativeRequest) -> list[str]:
    modes: list[str] = []
    if request.planning_constraints:
        modes.extend(request.planning_constraints.transport.avoid_modes)
        modes.extend(request.planning_constraints.transport.disallowed_modes)
    if request.current_route:
        modes.extend(request.current_route.preferences.avoid_modes)
    seen: list[str] = []
    for mode in modes:
        token = _mode_token(mode)
        if token and token not in seen:
            seen.append(token)
    return seen


def _primary_mode(request: RouteAlternativeRequest, default: str) -> str:
    preferred = _preferred_modes(request)
    if "train" in preferred:
        return "train"
    if preferred:
        mode = preferred[0]
        if mode not in _avoid_modes(request):
            return mode
    return default if default not in _avoid_modes(request) else "train"


def _car_mode(request: RouteAlternativeRequest) -> str | None:
    if not _car_available(request):
        return None
    if "road_trip" in _trip_styles(request):
        return "rental_car"
    return "car"


def _car_available(request: RouteAlternativeRequest) -> bool:
    if request.planning_constraints:
        return request.planning_constraints.transport.car_available
    if request.current_route:
        return request.current_route.preferences.car_available
    return False


def _trip_styles(request: RouteAlternativeRequest) -> list[str]:
    styles: list[str] = []
    if request.planning_constraints:
        styles.extend(request.planning_constraints.trip_styles)
    if request.current_route:
        styles.extend(request.current_route.preferences.trip_styles)
    text = _request_text(request)
    if _mentions(text, "camping"):
        styles.append("camping")
    if _mentions(text, "hiking"):
        styles.append("hiking")
    if _mentions(text, "nature"):
        styles.append("nature")
    if _mentions(text, "road trip", "road_trip"):
        styles.append("road_trip")
    out: list[str] = []
    for style in styles:
        token = style.strip().lower().replace("-", "_").replace(" ", "_")
        if token and token not in out:
            out.append(token)
    return out


def _max_transfer_hours(request: RouteAlternativeRequest) -> int:
    if (
        request.planning_constraints
        and request.planning_constraints.transport.max_transfer_hours_per_day
    ):
        return request.planning_constraints.transport.max_transfer_hours_per_day
    if request.current_route and request.current_route.preferences.max_transfer_hours_per_day:
        return request.current_route.preferences.max_transfer_hours_per_day
    return 6


def _mode_token(value: str) -> str:
    return value.strip().lower().replace("-", "_").replace(" ", "_")


def _sum_minutes(route: TripRoute) -> int:
    return sum(leg.estimated_duration_minutes or 0 for leg in route.legs)


def _sum_cost(route: TripRoute) -> Decimal:
    total = Decimal("0")
    for leg in route.legs:
        if leg.estimated_cost and leg.estimated_cost.amount is not None:
            total += leg.estimated_cost.amount
    return total


def _estimate_budget(request: RouteAlternativeRequest, stop_count: int) -> Decimal:
    if request.budget and request.budget.amount is not None:
        return max(Decimal("120"), request.budget.amount * Decimal("0.85"))
    days = request.duration_days or 5
    return Decimal(str(days * 90 + stop_count * 45))


def _difficulty(route: TripRoute, duration_days: int | None) -> str:
    days = max(1, duration_days or len(route.stops) or 1)
    transfer_minutes = _sum_minutes(route)
    if len(route.stops) <= 2 and transfer_minutes / days <= 80:
        return "relaxed"
    if len(route.stops) / days > 0.7 or transfer_minutes / days > 150:
        return "intense"
    return "balanced"


def _suggested_prompt(request: RouteAlternativeRequest, route: TripRoute) -> str:
    stop_names = ", ".join(stop.destination for stop in route.stops)
    origin = _origin(request).name or "the origin"
    days = request.duration_days or max(1, len(route.stops) + 2)
    return (
        f"Create a {days}-day route from {origin} through {stop_names}. "
        "Keep transport estimates approximate."
    )


def _title(request: RouteAlternativeRequest, key: str) -> str:
    return _text(request.output_language, key)


def _summary(request: RouteAlternativeRequest, key: str) -> str:
    return _text(request.output_language, key + "_summary")


def _text(language: str, key: str) -> str:
    return _TEXT.get(language, _TEXT["en"]).get(key, _TEXT["en"][key])


_TEXT = {
    "en": {
        "session_title": "Route alternatives",
        "follow_up": "Would you prefer fewer stops, lower cost, or more nature?",
        "warning": "Route estimates are approximate and do not include live ticket prices.",
        "estimate_note": "Approximate estimate; verify schedules and prices before travel.",
        "pro": "Clear route structure with practical transfer days.",
        "con": "Some transfer times and costs still need checking.",
        "classic_austria": "Classic Austria Train Route",
        "classic_austria_summary": "A balanced route through Vienna, Salzburg, and Hallstatt.",
        "relaxed_two_city": "Relaxed Two-City Route",
        "relaxed_two_city_summary": "A simpler Vienna and Salzburg route with more breathing room.",
        "nature_heavy": "Nature-Heavy Route",
        "nature_heavy_summary": (
            "A more outdoor-focused route through Graz, Hallstatt, and Salzburg."
        ),
        "current_route": "Current Route Baseline",
        "current_route_summary": "Your current route preserved as a comparison baseline.",
        "fewer_stops": "Fewer-Stops Relaxed Route",
        "fewer_stops_summary": "A reduced version that keeps the route easier to execute.",
    },
    "es": {
        "session_title": "Alternativas de ruta",
        "follow_up": "¿Prefieres menos paradas, menor coste o más naturaleza?",
        "warning": "Las estimaciones son aproximadas y no incluyen precios en vivo.",
        "estimate_note": "Estimación aproximada; verifica horarios y precios antes de viajar.",
        "pro": "Ruta clara con días de traslado prácticos.",
        "con": "Algunos tiempos y costes aún deben comprobarse.",
        "classic_austria": "Ruta clásica en tren por Austria",
        "classic_austria_summary": "Una ruta equilibrada por Viena, Salzburgo y Hallstatt.",
        "relaxed_two_city": "Ruta relajada de dos ciudades",
        "relaxed_two_city_summary": "Una ruta más sencilla por Viena y Salzburgo.",
        "nature_heavy": "Ruta centrada en naturaleza",
        "nature_heavy_summary": "Una ruta más al aire libre por Graz, Hallstatt y Salzburgo.",
        "current_route": "Ruta actual",
        "current_route_summary": "Tu ruta actual como base de comparación.",
        "fewer_stops": "Ruta relajada con menos paradas",
        "fewer_stops_summary": "Una versión reducida más fácil de ejecutar.",
    },
    "uk": {
        "session_title": "Варіанти маршруту",
        "follow_up": "Вам важливіше менше зупинок, нижча ціна чи більше природи?",
        "warning": "Оцінки маршруту приблизні й не включають актуальні ціни на квитки.",
        "estimate_note": "Приблизна оцінка; перевірте розклади й ціни перед поїздкою.",
        "pro": "Зрозуміла структура маршруту з реалістичними днями переїздів.",
        "con": "Деякі переїзди та витрати ще потрібно перевірити.",
        "classic_austria": "Класичний залізничний маршрут Австрією",
        "classic_austria_summary": "Збалансований маршрут через Відень, Зальцбург і Гальштат.",
        "relaxed_two_city": "Спокійний маршрут двома містами",
        "relaxed_two_city_summary": "Простіший маршрут через Відень і Зальцбург із запасом часу.",
        "nature_heavy": "Маршрут з акцентом на природу",
        "nature_heavy_summary": "Більше природи у маршруті через Грац, Гальштат і Зальцбург.",
        "current_route": "Поточний маршрут",
        "current_route_summary": "Ваш поточний маршрут як основа для порівняння.",
        "fewer_stops": "Спокійний маршрут із меншою кількістю зупинок",
        "fewer_stops_summary": "Скорочена версія, яку легше виконати.",
    },
    "fr": {
        "session_title": "Alternatives d'itinéraire",
        "follow_up": "Préférez-vous moins d'arrêts, un coût plus bas ou plus de nature ?",
        "warning": "Les estimations sont approximatives et n'incluent pas les prix en direct.",
        "estimate_note": "Estimation approximative ; vérifiez horaires et prix avant le voyage.",
        "pro": "Structure claire avec des jours de transfert réalistes.",
        "con": "Certains temps et coûts doivent encore être vérifiés.",
        "classic_austria": "Itinéraire classique en train en Autriche",
        "classic_austria_summary": "Un itinéraire équilibré par Vienne, Salzbourg et Hallstatt.",
        "relaxed_two_city": "Itinéraire détendu à deux villes",
        "relaxed_two_city_summary": "Un parcours plus simple entre Vienne et Salzbourg.",
        "nature_heavy": "Itinéraire axé nature",
        "nature_heavy_summary": "Un parcours plus nature par Graz, Hallstatt et Salzbourg.",
        "current_route": "Itinéraire actuel",
        "current_route_summary": "Votre itinéraire actuel comme base de comparaison.",
        "fewer_stops": "Itinéraire détendu avec moins d'arrêts",
        "fewer_stops_summary": "Une version réduite plus facile à réaliser.",
    },
}
