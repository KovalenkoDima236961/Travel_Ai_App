from copy import deepcopy
from decimal import Decimal, InvalidOperation
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
from app.schemas.repair import (
    RepairChange,
    RepairItineraryRequest,
    RepairItineraryResponse,
    RepairMoney,
    RepairSummary,
)


class ItineraryGenerator(Protocol):
    def generate(self, request: GenerateItineraryRequest) -> ItineraryResponse: ...

    def regenerate_day(self, request: RegenerateDayRequest) -> RegenerateDayResponse: ...

    def regenerate_item(self, request: RegenerateItemRequest) -> RegenerateItemResponse: ...

    def optimize_budget_day(
        self, request: OptimizeBudgetDayRequest
    ) -> BudgetOptimizationProposalResponse: ...

    def repair_itinerary(self, request: RepairItineraryRequest) -> RepairItineraryResponse: ...


class MockItineraryGenerator:
    def generate(self, request: GenerateItineraryRequest) -> ItineraryResponse:
        currency = request.budget_currency
        days: list[ItineraryDay] = []
        for day_number in range(1, request.days + 1):
            items = self._items_for_day(request, day_number)
            _localize_mock_items(items, request.output_language, request.destination)
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
        _localize_mock_items(items, request.output_language, destination)
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
        _localize_mock_items([item], request.output_language, request.trip.destination)
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
            day_number=request.day_number,
            currency=currency,
            base_day_estimated_total=base_total,
            proposed_day_estimated_total=proposed_total,
            estimated_savings_amount=savings,
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
                    item_index=0,
                    item_name=proposed_day.items[0].name,
                    reason="Preserved to keep the day structure recognizable.",
                )
            ],
            tradeoffs=["The replacement is less premium but keeps the route and theme practical."],
            warnings=["Estimated savings are approximate and should be reviewed."],
            proposed_day=proposed_day,
        )

    def repair_itinerary(self, request: RepairItineraryRequest) -> RepairItineraryResponse:
        repaired = deepcopy(request.itinerary)
        currency = _repair_currency(request)
        before_total = _repair_total_cost(repaired, currency)
        changes: list[RepairChange] = []
        issues_addressed = _repair_issue_types(request)
        max_changes = request.constraints.max_changed_items or 10
        mode = request.constraints.repair_mode

        def change_budget() -> None:
            for day, item_index, item in _items_by_cost_desc(repaired):
                if len(changes) >= max_changes:
                    return
                amount = _repair_item_amount(item)
                if amount is None or amount <= 0:
                    continue
                before = _compact_item(item)
                new_amount = max(Decimal("0"), (amount * Decimal("0.70")).quantize(Decimal("0.01")))
                _set_repair_item_amount(item, new_amount, currency)
                item["name"] = _marked_name(item.get("name"), "lower-cost")
                item["note"] = _append_note(
                    item.get("note"),
                    "AI repair lowered this estimated cost for policy review.",
                )
                changes.append(
                    RepairChange(
                        type="item_modified",
                        dayNumber=_day_number(day),
                        itemIndex=item_index,
                        before=before,
                        after=_compact_item(item),
                        reason="Reduce budget risk with a lower-cost version of the item.",
                    )
                )

        def change_late_items() -> None:
            for day in _repair_days(repaired):
                for item_index, item in enumerate(_day_items(day)):
                    if len(changes) >= max_changes:
                        return
                    value = str(item.get("endTime") or item.get("time") or "").strip()
                    if not _is_late_time(value):
                        continue
                    before = _compact_item(item)
                    item["time"] = "19:00"
                    if "endTime" in item:
                        item["endTime"] = "20:30"
                    item["note"] = _append_note(
                        item.get("note"),
                        "AI repair moved this earlier to reduce late-schedule risk.",
                    )
                    changes.append(
                        RepairChange(
                            type="item_moved",
                            dayNumber=_day_number(day),
                            itemIndex=item_index,
                            before=before,
                            after=_compact_item(item),
                            reason="Move late activity earlier.",
                        )
                    )

        def add_rest_blocks() -> None:
            for day in _repair_days(repaired):
                if len(changes) >= max_changes:
                    return
                items = _day_items(day)
                if any(_is_rest_item(item) for item in items):
                    continue
                rest = {
                    "time": "15:00",
                    "type": "rest",
                    "name": "AI repair rest break",
                    "note": "Added reviewable downtime to reduce itinerary density risk.",
                    "estimatedCost": {
                        "amount": 0,
                        "currency": currency,
                        "category": "other",
                        "confidence": "high",
                        "source": "ai",
                    },
                }
                items.append(rest)
                changes.append(
                    RepairChange(
                        type="item_added",
                        dayNumber=_day_number(day),
                        itemIndex=len(items) - 1,
                        before=None,
                        after=_compact_item(rest),
                        reason="Add rest time.",
                    )
                )

        def replace_disallowed() -> None:
            targets = _affected_targets(request)
            for target_day, target_index in targets:
                if len(changes) >= max_changes:
                    return
                item = _item_at(repaired, target_day, target_index)
                if item is None:
                    continue
                before = _compact_item(item)
                item["type"] = "activity"
                item["name"] = _marked_name(item.get("name"), "allowed alternative")
                item["note"] = _append_note(
                    item.get("note"),
                    "AI repair replaced a potentially disallowed item for review.",
                )
                changes.append(
                    RepairChange(
                        type="item_replaced",
                        dayNumber=target_day,
                        itemIndex=target_index,
                        before=before,
                        after=_compact_item(item),
                        reason="Replace potentially disallowed activity.",
                    )
                )

        selected = {item.casefold() for item in request.constraints.selected_issue_types}
        issue_text = " ".join([mode, *selected, *issues_addressed]).casefold()
        if mode in {"policy_compliance", "reduce_budget_risk"} or "budget" in issue_text:
            change_budget()
        if mode in {"policy_compliance", "fix_schedule_risk"} or "late" in issue_text:
            change_late_items()
        if mode in {"policy_compliance", "add_rest_time"} or "rest" in issue_text:
            add_rest_blocks()
        if mode in {"policy_compliance", "replace_disallowed_items"} or "disallowed" in issue_text:
            replace_disallowed()
        if mode == "reduce_walking":
            _add_repair_warning_note(repaired, "AI repair reviewed walking risk; verify routes.")
        if not changes and mode == "selected_issues":
            change_budget()
        if not changes:
            add_rest_blocks()

        after_total = _repair_total_cost(repaired, currency)
        warnings = [
            "Availability must be checked again after repair.",
            "Costs are estimates and should be reviewed before approval.",
        ]
        if len(changes) >= max_changes:
            warnings.append("Repair stopped at the maxChangedItems limit.")

        changed = sum(1 for change in changes if change.type in {"item_modified", "item_replaced"})
        added = sum(1 for change in changes if change.type == "item_added")
        removed = sum(1 for change in changes if change.type == "item_removed")
        moved = sum(1 for change in changes if change.type == "item_moved")

        return RepairItineraryResponse(
            repairedItinerary=repaired,
            repairSummary=RepairSummary(
                repairMode=mode,
                changedItemCount=changed,
                addedItemCount=added,
                removedItemCount=removed,
                movedItemCount=moved,
                estimatedCostBefore=RepairMoney(amount=before_total, currency=currency),
                estimatedCostAfter=RepairMoney(amount=after_total, currency=currency),
                majorChanges=[change.reason or change.type for change in changes[:8]],
                issuesAddressed=issues_addressed,
                issuesRemaining=["availability_unchecked"],
                warnings=warnings,
            ),
            changes=changes,
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

        if request.output_language == "es":
            return f"Día {day_number}: paseo matutino por {destination}"
        if request.output_language == "uk":
            return f"День {day_number}: ранкова прогулянка містом {destination}"
        if request.output_language == "fr":
            return f"Jour {day_number} : promenade matinale à {destination}"
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


_MOCK_LOCALIZED_TEXT = {
    "es": {
        "names": [
            "Paseo matutino por la ciudad",
            "Almuerzo en un restaurante local",
            "Visita cultural por la tarde",
            "Cena recomendada",
            "Pausa tranquila",
        ],
        "note": "Sugerencia local determinista para este itinerario de prueba.",
    },
    "uk": {
        "names": [
            "Ранкова прогулянка містом",
            "Обід у місцевому закладі",
            "Культурна програма по обіді",
            "Рекомендована вечеря",
            "Спокійна перерва",
        ],
        "note": "Детермінована місцева рекомендація для цього тестового маршруту.",
    },
    "fr": {
        "names": [
            "Promenade matinale en ville",
            "Déjeuner dans une adresse locale",
            "Visite culturelle l’après-midi",
            "Dîner recommandé",
            "Pause tranquille",
        ],
        "note": "Suggestion locale déterministe pour cet itinéraire de test.",
    },
}


def _localize_mock_items(items: list[ItineraryItem], language: str, destination: str) -> None:
    localized = _MOCK_LOCALIZED_TEXT.get(language)
    if localized is None:
        return
    names = localized["names"]
    for index, item in enumerate(items):
        item.name = f"{names[index % len(names)]} — {destination}"
        item.note = localized["note"]


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


def _repair_currency(request: RepairItineraryRequest) -> str:
    if request.trip_context.budget is not None:
        return request.trip_context.budget.currency
    raw = str(request.itinerary.get("currency") or "").strip().upper()
    return raw if len(raw) == 3 else "EUR"


def _repair_issue_types(request: RepairItineraryRequest) -> list[str]:
    values = [issue.type for issue in request.issues]
    if request.constraints.selected_issue_types:
        selected = {item.casefold() for item in request.constraints.selected_issue_types}
        values = [item for item in values if item.casefold() in selected]
    out: list[str] = []
    seen: set[str] = set()
    for value in values:
        key = value.casefold()
        if key in seen:
            continue
        seen.add(key)
        out.append(value)
    return out


def _repair_days(itinerary: dict) -> list[dict]:
    days = itinerary.get("days")
    return [day for day in days if isinstance(day, dict)] if isinstance(days, list) else []


def _day_items(day: dict) -> list[dict]:
    items = day.get("items")
    if not isinstance(items, list):
        day["items"] = []
        return day["items"]
    filtered = [item for item in items if isinstance(item, dict)]
    if len(filtered) != len(items):
        day["items"] = filtered
    return filtered


def _day_number(day: dict) -> int | None:
    value = day.get("day")
    return value if isinstance(value, int) and value > 0 else None


def _repair_total_cost(itinerary: dict, currency: str) -> Decimal:
    total = Decimal("0")
    for day in _repair_days(itinerary):
        for item in _day_items(day):
            amount = _repair_item_amount(item)
            if amount is None or amount < 0:
                continue
            cost = item.get("estimatedCost")
            item_currency = currency
            if isinstance(cost, dict):
                item_currency = str(cost.get("currency") or currency).strip().upper()
            if item_currency == currency:
                total += amount
    return total.quantize(Decimal("0.01"))


def _repair_item_amount(item: dict) -> Decimal | None:
    cost = item.get("estimatedCost")
    raw: object = cost.get("amount") if isinstance(cost, dict) else cost
    if raw is None or raw == "":
        return None
    try:
        return Decimal(str(raw))
    except (InvalidOperation, ValueError):
        return None


def _set_repair_item_amount(item: dict, amount: Decimal, currency: str) -> None:
    cost = item.get("estimatedCost")
    if not isinstance(cost, dict):
        cost = {}
        item["estimatedCost"] = cost
    cost["amount"] = _repair_serialize_decimal(amount)
    cost["currency"] = str(cost.get("currency") or currency).strip().upper() or currency
    cost["category"] = str(cost.get("category") or item.get("type") or "other").strip() or "other"
    cost["confidence"] = str(cost.get("confidence") or "medium").strip() or "medium"
    cost["source"] = "ai"


def _items_by_cost_desc(itinerary: dict) -> list[tuple[dict, int, dict]]:
    refs: list[tuple[Decimal, dict, int, dict]] = []
    for day in _repair_days(itinerary):
        for index, item in enumerate(_day_items(day)):
            amount = _repair_item_amount(item)
            if amount is not None:
                refs.append((amount, day, index, item))
    refs.sort(key=lambda value: value[0], reverse=True)
    return [(day, index, item) for _, day, index, item in refs]


def _repair_serialize_decimal(value: Decimal) -> int | float:
    if value == value.to_integral_value():
        return int(value)
    return float(value)


def _compact_item(item: dict) -> dict:
    return {
        key: item.get(key)
        for key in ("time", "endTime", "type", "name", "estimatedCost")
        if key in item
    }


def _marked_name(raw: object, marker: str) -> str:
    name = str(raw or "itinerary item").strip()
    if marker.casefold() in name.casefold():
        return name
    return f"{name} ({marker})"


def _append_note(raw: object, extra: str) -> str:
    note = str(raw or "").strip()
    if not note:
        return extra
    if extra in note:
        return note
    return f"{note} {extra}"


def _is_late_time(raw: str) -> bool:
    if len(raw) < 5 or raw[2] != ":":
        return False
    try:
        hour = int(raw[:2])
        minute = int(raw[3:5])
    except ValueError:
        return False
    return hour * 60 + minute > 21 * 60


def _is_rest_item(item: dict) -> bool:
    value = str(item.get("type") or "").strip().casefold()
    return value in {"rest", "break", "free_time", "freetime"}


def _affected_targets(request: RepairItineraryRequest) -> list[tuple[int, int]]:
    targets: list[tuple[int, int]] = []
    for issue in request.issues:
        affected = issue.affected
        day_number: int | None = None
        item_index: int | None = None
        if hasattr(affected, "day_number"):
            day_number = affected.day_number  # type: ignore[union-attr]
            item_index = affected.item_index  # type: ignore[union-attr]
        elif isinstance(affected, dict):
            day_raw = affected.get("dayNumber")
            item_raw = affected.get("itemIndex")
            day_number = day_raw if isinstance(day_raw, int) else None
            item_index = item_raw if isinstance(item_raw, int) else None
        if day_number is not None and item_index is not None:
            targets.append((day_number, item_index))
    if targets:
        return targets
    for day in _repair_days(request.itinerary):
        items = _day_items(day)
        if items:
            number = _day_number(day)
            if number is not None:
                return [(number, 0)]
    return []


def _item_at(itinerary: dict, day_number: int, item_index: int) -> dict | None:
    for day in _repair_days(itinerary):
        if _day_number(day) != day_number:
            continue
        items = _day_items(day)
        if 0 <= item_index < len(items):
            return items[item_index]
    return None


def _add_repair_warning_note(itinerary: dict, note: str) -> None:
    for day in _repair_days(itinerary):
        for item in _day_items(day)[:1]:
            item["note"] = _append_note(item.get("note"), note)
            return


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
