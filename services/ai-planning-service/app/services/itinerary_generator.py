from decimal import Decimal
from typing import Protocol

from app.schemas.itinerary import (
    BudgetOptimizationChange,
    BudgetOptimizationPreservedItem,
    BudgetOptimizationProposalResponse,
    GenerateItineraryRequest,
    ItineraryDay,
    ItineraryItem,
    ItineraryResponse,
    OptimizeBudgetDayRequest,
    RegenerateDayRequest,
    RegenerateDayResponse,
    RegenerateItemRequest,
    RegenerateItemResponse,
    WeatherDay,
)


class ItineraryGenerator(Protocol):
    def generate(self, request: GenerateItineraryRequest) -> ItineraryResponse: ...
    def regenerate_day(self, request: RegenerateDayRequest) -> RegenerateDayResponse: ...
    def regenerate_item(self, request: RegenerateItemRequest) -> RegenerateItemResponse: ...
    def optimize_budget_day(
        self, request: OptimizeBudgetDayRequest
    ) -> BudgetOptimizationProposalResponse: ...


class MockItineraryGenerator:
    def generate(self, request: GenerateItineraryRequest) -> ItineraryResponse:
        currency = request.budget_currency
        days: list[ItineraryDay] = []
        for day_number in range(1, request.days + 1):
            items = self._items_for_day(request, day_number)
            _finalize_item_costs(items, currency)
            days.append(
                ItineraryDay(
                    day=day_number,
                    title=self._title_for_day(request, day_number),
                    items=items,
                )
            )
        return ItineraryResponse(days=days)

    def regenerate_day(self, request: RegenerateDayRequest) -> RegenerateDayResponse:
        destination = request.trip.destination
        cheap = _mentions(request.instruction, "cheap", "cheaper", "budget")
        relaxed = _mentions(request.instruction, "relaxed", "slow", "easy")
        first_time = "10:00" if relaxed else "09:30"
        lunch_cost = Decimal("10") if cheap else Decimal("16")
        weather_day = _weather_day_for_number(
            request.weather_forecast.days if request.weather_forecast else [], request.day_number
        )
        first_note = "A focused replacement day that keeps the rest of the trip intact."
        third_type = "rest" if relaxed else "place"
        third_name = "Quiet cafe break" if relaxed else "Updated signature stop"
        third_note = "Keeps timing realistic alongside the unchanged itinerary days."
        if weather_day and weather_day.precipitation_chance >= 60:
            first_note = "Rain is likely, so start with an indoor museum or covered market."
            third_type = "rest"
            third_name = "Quiet cafe break"
            third_note = "Use this as a dry indoor backup while the weather is unsettled."
        elif weather_day and weather_day.temperature_max_c >= 32:
            first_note = (
                "High heat expected, so keep walking short and avoid exposed midday routes."
            )
            third_note = "Plan this slower indoor break to avoid the afternoon heat."

        items = [
            ItineraryItem(
                time=first_time,
                type="activity",
                name=f"Updated {destination} neighborhood walk",
                note=first_note,
                estimated_cost=Decimal("0"),
            ),
            ItineraryItem(
                time="12:30",
                type="food",
                name="Budget local lunch" if cheap else "Local lunch stop",
                note="Simple local food option selected for partial regeneration.",
                estimated_cost=lunch_cost,
            ),
            ItineraryItem(
                time="15:30",
                type=third_type,
                name=third_name,
                note=third_note,
                estimated_cost=Decimal("6"),
            ),
        ]
        _finalize_item_costs(items, request.trip.budget_currency)
        return RegenerateDayResponse(
            day=ItineraryDay(
                day=request.day_number,
                title=f"Day {request.day_number}: refreshed {destination} plan",
                items=items,
            )
        )

    def regenerate_item(self, request: RegenerateItemRequest) -> RegenerateItemResponse:
        cheap = _mentions(request.instruction, "cheap", "cheaper", "budget")
        weather_day = _weather_day_for_number(
            request.weather_forecast.days if request.weather_forecast else [], request.day_number
        )
        note = (
            f"Mock replacement for zero-based item index {request.item_index} "
            f"on day {request.day_number}."
        )
        item_type = "food"
        item_name = "Budget local food option" if cheap else "Updated local option"
        if weather_day and weather_day.precipitation_chance >= 60:
            item_type = "rest"
            item_name = "Indoor cafe or covered market stop"
            note = "Rain is likely, so this replacement keeps the plan indoors."
        elif weather_day and weather_day.temperature_max_c >= 32:
            note += " High heat expected; choose a shaded or indoor option."
        item = ItineraryItem(
            time="12:30",
            type=item_type,
            name=item_name,
            note=note,
            estimated_cost=Decimal("9") if cheap else Decimal("15"),
        )
        _finalize_item_costs([item], request.trip.budget_currency)
        return RegenerateItemResponse(item=item)

    def optimize_budget_day(
        self, request: OptimizeBudgetDayRequest
    ) -> BudgetOptimizationProposalResponse:
        currency = request.budget_context.currency
        proposed_day = ItineraryDay.model_validate(
            request.current_day.model_dump(by_alias=True, exclude_none=True)
        )
        expensive_index = _most_expensive_item_index(proposed_day.items)
        if expensive_index < 0:
            expensive_index = 0

        old_item = proposed_day.items[expensive_index]
        old_amount = _cost_amount(old_item)
        new_amount = max(Decimal("0"), old_amount - Decimal("35"))
        proposed_day.items[expensive_index] = ItineraryItem(
            time=old_item.time,
            type="activity",
            name="Self-guided low-cost alternative",
            note="Keeps the day theme but avoids the highest estimated cost.",
            estimated_cost={
                "amount": new_amount,
                "currency": currency,
                "category": "activity",
                "confidence": "medium",
                "source": "ai",
            },
        )

        base_total = request.budget_context.day_estimated_total
        savings = max(Decimal("1"), old_amount - new_amount)
        proposed_total = max(Decimal("0"), base_total - savings)

        return BudgetOptimizationProposalResponse(
            summary=(
                f"Reduced estimated Day {request.day_number} cost by about "
                f"{savings} {currency} with a cheaper activity alternative."
            ),
            scope="day",
            dayNumber=request.day_number,
            currency=currency,
            baseDayEstimatedTotal=base_total,
            proposedDayEstimatedTotal=proposed_total,
            estimatedSavingsAmount=savings,
            confidence="medium",
            changes=[
                BudgetOptimizationChange(
                    type="replace_item",
                    oldItemIndex=expensive_index,
                    oldItemName=old_item.name,
                    newItemName=proposed_day.items[expensive_index].name,
                    reason="Replaces the highest-cost item with a lower-cost option.",
                    estimatedSavingsAmount=savings,
                    currency=currency,
                )
            ],
            preservedItems=[
                BudgetOptimizationPreservedItem(
                    itemIndex=0,
                    itemName=proposed_day.items[0].name,
                    reason="Preserved to keep the day structure recognizable.",
                )
            ],
            tradeoffs=["The replacement is less premium but keeps the route and theme practical."],
            warnings=["Estimated savings are approximate and should be reviewed."],
            proposedDay=proposed_day,
        )

    def _title_for_day(self, request: GenerateItineraryRequest, day_number: int) -> str:
        interests = self._personalized_interests(request)
        destination = request.destination

        if "history" in interests and "food" in interests:
            theme = "historic streets and local food"
        elif "history" in interests:
            theme = "historic highlights"
        elif "food" in interests:
            theme = "markets and local flavors"
        elif "hidden_gems" in interests:
            theme = "local neighborhoods"
        else:
            theme = f"{request.pace} city highlights"

        return f"Day {day_number}: {destination} {theme}"

    def _items_for_day(
        self, request: GenerateItineraryRequest, day_number: int
    ) -> list[ItineraryItem]:
        interests = self._personalized_interests(request)
        destination = request.destination

        if request.pace == "relaxed":
            return _apply_weather(
                self._relaxed_items(destination, interests, day_number),
                _weather_day_for_number(
                    request.weather_forecast.days if request.weather_forecast else [], day_number
                ),
            )
        if request.pace == "intensive":
            return _apply_weather(
                self._intensive_items(destination, interests, day_number),
                _weather_day_for_number(
                    request.weather_forecast.days if request.weather_forecast else [], day_number
                ),
            )
        return _apply_weather(
            self._balanced_items(destination, interests, day_number),
            _weather_day_for_number(
                request.weather_forecast.days if request.weather_forecast else [], day_number
            ),
        )

    def _balanced_items(
        self, destination: str, interests: set[str], day_number: int
    ) -> list[ItineraryItem]:
        return [
            self._morning_item("09:00", destination, interests, day_number),
            self._lunch_item("12:30", destination, interests),
            self._afternoon_item("15:30", destination, interests, day_number),
            self._evening_item("19:00", destination, interests),
        ]

    def _relaxed_items(
        self, destination: str, interests: set[str], day_number: int
    ) -> list[ItineraryItem]:
        return [
            self._morning_item("10:00", destination, interests, day_number),
            self._lunch_item("13:00", destination, interests),
            self._afternoon_item("16:30", destination, interests, day_number),
        ]

    def _intensive_items(
        self, destination: str, interests: set[str], day_number: int
    ) -> list[ItineraryItem]:
        return [
            self._morning_item("08:30", destination, interests, day_number),
            ItineraryItem(
                time="11:00",
                type="activity",
                name=self._secondary_activity_name(interests),
                note=self._secondary_activity_note(destination, interests),
                estimated_cost=Decimal("12"),
            ),
            self._lunch_item("13:00", destination, interests),
            self._afternoon_item("15:30", destination, interests, day_number),
            self._evening_item("20:00", destination, interests),
        ]

    def _morning_item(
        self,
        time: str,
        destination: str,
        interests: set[str],
        day_number: int,
    ) -> ItineraryItem:
        if "history" in interests:
            return ItineraryItem(
                time=time,
                type="place",
                name=f"{destination} historic center walk",
                note=(
                    f"Start in {destination} with older streets and landmark context "
                    f"before the day {day_number} crowds build."
                ),
                estimated_cost=Decimal("18"),
            )

        note = f"Begin day {day_number} in {destination} with an easy orientation walk."
        if "hidden_gems" in interests:
            note += " Favor side streets and smaller squares over the main tourist route."

        return ItineraryItem(
            time=time,
            type="place",
            name=f"{destination} city orientation",
            note=note,
            estimated_cost=Decimal("0"),
        )

    def _lunch_item(self, time: str, destination: str, interests: set[str]) -> ItineraryItem:
        if "local_food" in interests:
            note = (
                f"In {destination}, choose a simple local place with seasonal dishes "
                "and prices that suit a budget-conscious food day."
            )
            name = "Local neighborhood lunch"
        elif "food" in interests:
            note = (
                f"In {destination}, pick a small local place and order a seasonal dish "
                "instead of the most visible tourist menu."
            )
            name = "Neighborhood trattoria or market stall"
        else:
            note = (
                f"Keep lunch simple in {destination} and choose somewhere close to the next stop."
            )
            name = "Local lunch stop"

        if "hidden_gems" in interests:
            note += " Look one or two blocks away from the busiest square."

        return ItineraryItem(
            time=time,
            type="food",
            name=name,
            note=note,
            estimated_cost=Decimal("15"),
        )

    def _afternoon_item(
        self,
        time: str,
        destination: str,
        interests: set[str],
        day_number: int,
    ) -> ItineraryItem:
        if "hidden_gems" in interests:
            return ItineraryItem(
                time=time,
                type="activity",
                name="Hidden-gem local neighborhood stop",
                note=(
                    f"Explore a quieter {destination} neighborhood with independent "
                    "shops and small cafes."
                ),
                estimated_cost=Decimal("5"),
            )

        if "history" in interests:
            return ItineraryItem(
                time=time,
                type="activity",
                name="Museum or archaeological site",
                note=(
                    f"Use the afternoon in {destination} for a focused history stop "
                    f"on day {day_number}."
                ),
                estimated_cost=Decimal("16"),
            )

        return ItineraryItem(
            time=time,
            type="activity",
            name="Scenic viewpoint and main square",
            note=f"Balance the day in {destination} with a broad city view and a central stroll.",
            estimated_cost=Decimal("8"),
        )

    def _evening_item(self, time: str, destination: str, interests: set[str]) -> ItineraryItem:
        if "food" in interests:
            note = (
                f"End in {destination} with a dinner area known for local cooking, "
                "then leave room for dessert or a late coffee."
            )
            name = "Dinner in a local food district"
        else:
            note = f"Close the day in {destination} with a low-pressure evening walk."
            name = "Evening stroll"

        return ItineraryItem(
            time=time,
            type="food" if "food" in interests else "activity",
            name=name,
            note=note,
            estimated_cost=Decimal("24") if "food" in interests else Decimal("0"),
        )

    def _secondary_activity_name(self, interests: set[str]) -> str:
        if "history" in interests:
            return "Guided history route"
        if "food" in interests:
            return "Market tasting route"
        if "hidden_gems" in interests:
            return "Less touristy local stop"
        return "Signature landmark stop"

    def _secondary_activity_note(self, destination: str, interests: set[str]) -> str:
        if "history" in interests:
            return f"Add a structured history route in {destination} while energy is still high."
        if "food" in interests:
            return f"Use the late morning in {destination} for a quick market tasting."
        if "hidden_gems" in interests:
            return f"Choose a quieter local corner of {destination} before lunch."
        return f"Add one more recognizable {destination} stop before lunch."

    def _personalized_interests(self, request: GenerateItineraryRequest) -> set[str]:
        interests = {_normalize_interest(value) for value in request.interests}
        preferences = request.user_preferences
        if preferences is None:
            return interests

        interests.update(_normalize_interest(value) for value in preferences.travel_styles)
        food_preferences = {_normalize_interest(value) for value in preferences.food_preferences}
        if "local" in food_preferences:
            interests.add("food")
            interests.add("local_food")
        return interests


_TYPE_TO_COST_CATEGORY = {
    "food": "food",
    "transport": "transport",
    "activity": "activity",
    "place": "ticket",
    "rest": "other",
}


def _infer_cost_category(item_type: str) -> str:
    return _TYPE_TO_COST_CATEGORY.get(item_type.strip().lower(), "other")


def _finalize_item_costs(items: list[ItineraryItem], currency: str | None) -> None:
    """Fill in the currency, category and source for sample item estimates.

    The deterministic items are built with a bare amount; this fills the rest of
    the structured estimate so mock output mirrors real generated costs.
    """
    for item in items:
        cost = item.estimated_cost
        if cost is None:
            continue
        if cost.currency is None and currency:
            cost.currency = currency
        if cost.category in (None, "other"):
            cost.category = _infer_cost_category(item.type)
        if cost.source is None:
            cost.source = "ai"


def _normalize_interest(value: str) -> str:
    return value.strip().lower().replace(" ", "_")


def _mentions(value: str | None, *terms: str) -> bool:
    if not value:
        return False
    normalized = value.casefold()
    return any(term in normalized for term in terms)


def _most_expensive_item_index(items: list[ItineraryItem]) -> int:
    best_index = -1
    best_amount = Decimal("-1")
    for index, item in enumerate(items):
        amount = _cost_amount(item)
        if amount > best_amount:
            best_amount = amount
            best_index = index
    return best_index


def _cost_amount(item: ItineraryItem) -> Decimal:
    if item.estimated_cost is None or item.estimated_cost.amount is None:
        return Decimal("0")
    return item.estimated_cost.amount


def _weather_day_for_number(days: list[WeatherDay], day_number: int) -> WeatherDay | None:
    index = day_number - 1
    if index < 0 or index >= len(days):
        return None
    return days[index]


def _apply_weather(
    items: list[ItineraryItem], weather_day: WeatherDay | None
) -> list[ItineraryItem]:
    if weather_day is None or not items:
        return items

    adjusted = [item.model_copy() for item in items]
    if weather_day.precipitation_chance >= 60:
        adjusted[0] = ItineraryItem(
            time=adjusted[0].time,
            type="place",
            name="Indoor museum or covered market",
            note="Rain is likely, so start indoors and keep outdoor stops as backups.",
            estimated_cost=Decimal("16"),
        )
        return adjusted

    if weather_day.temperature_max_c >= 32:
        midday_index = min(2, len(adjusted) - 1)
        note = adjusted[midday_index].note or ""
        adjusted[midday_index].note = (
            note + " High heat expected; avoid long outdoor walks at midday."
        ).strip()

    return adjusted
