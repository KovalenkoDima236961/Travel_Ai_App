from copy import deepcopy
from datetime import timedelta
from decimal import Decimal, InvalidOperation
from typing import Protocol

from app.schemas.checklist import GenerateChecklistRequest, GeneratedChecklistResponse
from app.schemas.generation_repair import (
    GenerationRepairChange,
    GenerationValidationIssue,
    RepairGenerationOutputRequest,
    RepairGenerationOutputResponse,
)
from app.schemas.grounding import GroundingPlace
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
from app.services.checklist_generator import generate_mock_checklist

_ITEMS_PER_DAY_BY_PACE = {
    "relaxed": 3,
    "balanced": 4,
    "intensive": 5,
}


class ItineraryGenerator(Protocol):
    def generate(self, request: GenerateItineraryRequest) -> ItineraryResponse: ...

    def generate_checklist(
        self, request: GenerateChecklistRequest
    ) -> GeneratedChecklistResponse: ...

    def regenerate_day(self, request: RegenerateDayRequest) -> RegenerateDayResponse: ...

    def regenerate_item(self, request: RegenerateItemRequest) -> RegenerateItemResponse: ...

    def optimize_budget_day(
        self, request: OptimizeBudgetDayRequest
    ) -> BudgetOptimizationProposalResponse: ...

    def repair_itinerary(self, request: RepairItineraryRequest) -> RepairItineraryResponse: ...

    def repair_generation_output(
        self, request: RepairGenerationOutputRequest
    ) -> RepairGenerationOutputResponse: ...


class MockItineraryGenerator:
    def generate(self, request: GenerateItineraryRequest) -> ItineraryResponse:
        if request.route is not None and len(request.route.stops) > 1:
            return self._generate_route_itinerary(request)

        currency = request.budget_currency
        days: list[ItineraryDay] = []
        for day_number in range(1, request.days + 1):
            items = self._items_for_day(request, day_number)
            if request.grounding_context is None or not request.grounding_context.places:
                _localize_mock_items(items, request.output_language, request.destination)
                _mark_generic_items(items)
            _finalize_item_costs(items, currency)
            days.append(
                ItineraryDay(
                    day=day_number,
                    date=(
                        request.start_date + timedelta(days=day_number - 1)
                        if request.start_date
                        else None
                    ),
                    title=self._title_for_day(request, day_number),
                    items=items,
                )
            )
        return ItineraryResponse(days=days)

    def generate_checklist(self, request: GenerateChecklistRequest) -> GeneratedChecklistResponse:
        return generate_mock_checklist(request)

    def _generate_route_itinerary(self, request: GenerateItineraryRequest) -> ItineraryResponse:
        if request.route is None or not request.route.stops:
            return ItineraryResponse(days=[])

        currency = request.budget_currency
        items_per_day = _ITEMS_PER_DAY_BY_PACE.get(request.pace, 4)
        stop_days = _route_stop_day_counts(request)
        days: list[ItineraryDay] = []

        for stop_index, stop in enumerate(request.route.stops):
            for day_at_stop in range(stop_days[stop_index]):
                if len(days) >= request.days:
                    break
                day_number = len(days) + 1
                transfer_day = day_at_stop == 0
                leg = _route_leg_for_stop(request, stop_index) if transfer_day else None
                items = _route_items_for_day(request, stop_index, stop, leg, transfer_day)
                _pad_route_items(items, stop.destination, currency, items_per_day)
                _localize_mock_items(items, request.output_language, stop.destination)
                _finalize_item_costs(items, currency)
                day_date = (
                    request.start_date + timedelta(days=day_number - 1)
                    if request.start_date
                    else None
                )
                days.append(
                    ItineraryDay(
                        day=day_number,
                        date=day_date,
                        title=(
                            f"Transfer to {_route_stop_name(stop)}"
                            if transfer_day
                            else f"Explore {_route_stop_name(stop)}"
                        ),
                        primaryStopId=stop.id,
                        locationName=_route_stop_name(stop),
                        transferDay=transfer_day,
                        items=items[:items_per_day],
                    )
                )
            if len(days) >= request.days:
                break

        while len(days) < request.days:
            stop = request.route.stops[-1]
            day_number = len(days) + 1
            items = [
                ItineraryItem(
                    time="10:00",
                    type="activity",
                    name=f"Flexible morning in {_route_stop_name(stop)}",
                    note="Use this as buffer time if previous transfers run late.",
                    estimated_cost={"amount": 0, "currency": currency, "category": "activity"},
                ),
            ]
            _pad_route_items(items, stop.destination, currency, items_per_day)
            _localize_mock_items(items, request.output_language, stop.destination)
            _finalize_item_costs(items, currency)
            day_date = (
                request.start_date + timedelta(days=day_number - 1) if request.start_date else None
            )
            days.append(
                ItineraryDay(
                    day=day_number,
                    date=day_date,
                    title=f"Flexible final day in {_route_stop_name(stop)}",
                    primaryStopId=stop.id,
                    locationName=_route_stop_name(stop),
                    transferDay=False,
                    items=items[:items_per_day],
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

    def repair_generation_output(
        self, request: RepairGenerationOutputRequest
    ) -> RepairGenerationOutputResponse:
        repaired = deepcopy(request.current_output)
        currency = _generation_repair_currency(request)
        changes: list[GenerationRepairChange] = []
        warnings: list[str] = []

        _ensure_generation_output_shape(repaired, currency)

        if _needs_day_count_repair(request):
            changes.extend(_normalize_generation_day_count(repaired, request, currency))

        for issue in request.validation_issues:
            issue_id = issue.id.casefold()
            if issue_id.startswith("schema_missing_required_field"):
                changes.extend(_repair_generation_schema_issue(repaired, issue, currency))
            elif "activity_during_transport" in issue_id or "activity_before_transport" in issue_id:
                changes.extend(_move_generation_item_after_transport(repaired, request, issue))
            elif (
                "transfer_item_missing" in issue_id
                or "missing_transfer_between_stops" in issue_id
            ):
                changes.extend(_add_generation_transfer_item(repaired, request, issue, currency))
            elif "place_likely_closed" in issue_id:
                changes.extend(_move_generation_item_to_opening_hours(repaired, issue))
            elif issue.category == "budget" or "budget" in issue_id:
                changes.extend(_reduce_generation_cost_risk(repaired, issue, currency))

        if not changes:
            warnings.append("No deterministic repair was available for the supplied issues.")

        for day in _generation_days(repaired):
            _sort_generation_day_items(day)

        return RepairGenerationOutputResponse(
            repaired_output=repaired,
            changes_made=changes,
            warnings=warnings,
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
        if request.grounding_context is not None and request.grounding_context.places:
            return self._grounded_items_for_day(request, day_number)
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

    def _grounded_items_for_day(
        self, request: GenerateItineraryRequest, day_number: int
    ) -> list[ItineraryItem]:
        assert request.grounding_context is not None
        target_count = _ITEMS_PER_DAY_BY_PACE.get(request.pace, 4)
        times = {
            3: ["10:00", "13:00", "16:30"],
            4: ["09:00", "12:30", "15:30", "19:00"],
            5: ["08:30", "11:00", "13:00", "15:30", "20:00"],
        }[target_count]
        candidates = self._grounding_candidates(request, day_number)
        offset = (day_number - 1) * target_count
        return [
            _grounded_item(candidates[(offset + index) % len(candidates)], times[index])
            for index in range(target_count)
        ]

    def _grounding_candidates(
        self, request: GenerateItineraryRequest, day_number: int
    ) -> list[GroundingPlace]:
        assert request.grounding_context is not None
        places = list(request.grounding_context.places)
        weather_day = _weather_day_for_number(
            request.weather_forecast.days if request.weather_forecast else [], day_number
        )
        if weather_day is not None and weather_day.precipitation_chance >= 60:
            rain_safe = [
                place
                for place in places
                if place.rain_friendly is True or place.outdoor is False
            ]
            if rain_safe:
                return rain_safe
        return places

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


def _grounded_item(place: GroundingPlace, time: str) -> ItineraryItem:
    item_type = "food" if place.category in {"restaurant", "cafe", "market"} else "place"
    duration = place.typical_duration_minutes
    note_parts = [f"Grounded in curated knowledge as a {place.category}."]
    if duration is not None:
        note_parts.append(f"Typical visit: about {duration} minutes.")
    if place.outdoor is True:
        note_parts.append("Outdoor; check weather before going.")
    if place.rain_friendly is True:
        note_parts.append("Suitable as a rain-friendly option.")
    return ItineraryItem(
        time=time,
        type=item_type,
        name=place.canonical_name,
        note=" ".join(note_parts),
        durationMinutes=duration,
        estimated_cost=Decimal("0"),
        groundingSource="grounded",
        groundingPlaceId=place.id,
        groundingConfidence=place.confidence,
        needsPlaceReview=False,
        groundingWarnings=[],
    )


def _mark_generic_items(items: list[ItineraryItem]) -> None:
    for item in items:
        item.grounding_source = "generic"
        item.needs_place_review = True
        item.grounding_warnings = ["No destination-specific knowledge record was available."]


def _route_stop_day_counts(request: GenerateItineraryRequest) -> list[int]:
    assert request.route is not None
    stop_count = len(request.route.stops)
    remaining_days = max(1, request.days)
    counts: list[int] = []
    for index, stop in enumerate(request.route.stops):
        remaining_stops = stop_count - index
        if remaining_stops <= 1:
            count = max(1, remaining_days)
        elif stop.nights and stop.nights > 0:
            count = min(max(1, stop.nights), max(1, remaining_days - remaining_stops + 1))
        else:
            count = max(1, remaining_days // remaining_stops)
        counts.append(count)
        remaining_days -= count
    if sum(counts) < request.days:
        counts[-1] += request.days - sum(counts)
    return counts


def _route_leg_for_stop(request: GenerateItineraryRequest, stop_index: int):
    if request.route is None:
        return None
    to_id = request.route.stops[stop_index].id
    from_id = "origin" if stop_index == 0 else request.route.stops[stop_index - 1].id
    for leg in request.route.legs:
        if leg.from_stop_id == from_id and leg.to_stop_id == to_id:
            return leg
    return None


def _route_stop_name(stop) -> str:
    return stop.city or stop.destination


def _route_items_for_day(
    request: GenerateItineraryRequest,
    stop_index: int,
    stop,
    leg,
    transfer_day: bool,
) -> list[ItineraryItem]:
    currency = request.budget_currency
    items: list[ItineraryItem] = []
    if transfer_day:
        items.append(_transfer_item_for_leg(request, stop_index, stop, leg, currency))

    first_time = "13:00" if transfer_day else "09:30"
    second_time = "15:30" if transfer_day else "12:30"
    third_time = "19:00" if transfer_day else "16:00"
    items.extend(
        [
            ItineraryItem(
                time=first_time,
                type="food",
                name=f"Local meal in {_route_stop_name(stop)}",
                note="Keep this flexible around arrival, check-in, and local opening hours.",
                estimated_cost={"amount": 16, "currency": currency, "category": "food"},
            ),
            ItineraryItem(
                time=second_time,
                type="activity",
                name=f"{_route_stop_name(stop)} orientation walk",
                note=_route_style_note(request),
                estimated_cost={"amount": 0, "currency": currency, "category": "activity"},
            ),
            ItineraryItem(
                time=third_time,
                type="food",
                name=f"Dinner near {_route_stop_name(stop)} center",
                note="A simple local dinner recommendation; verify hours before going.",
                estimated_cost={"amount": 24, "currency": currency, "category": "food"},
            ),
        ]
    )
    return items


def _transfer_item_for_leg(
    request: GenerateItineraryRequest,
    stop_index: int,
    stop,
    leg,
    currency: str,
) -> ItineraryItem:
    mode = leg.mode if leg is not None else _preferred_route_mode(request)
    from_name = "Origin"
    if request.route and request.route.origin and request.route.origin.name:
        from_name = request.route.origin.name
    if leg is not None and leg.from_name:
        from_name = leg.from_name
    elif request.route is not None and stop_index > 0:
        from_name = _route_stop_name(request.route.stops[stop_index - 1])
    to_name = leg.to_name if leg is not None and leg.to_name else _route_stop_name(stop)
    duration = leg.estimated_duration_minutes if leg is not None else None
    distance = leg.estimated_distance_km if leg is not None else None
    cost = _route_transfer_cost(leg, mode, distance, currency)
    duration_text = duration or 90
    end_hour = 9 + duration_text // 60
    end_minute = duration_text % 60
    end_time = f"{min(end_hour, 23):02d}:{end_minute:02d}"
    note = "Verify schedules before travel. This is not a booking or live ticket price."
    if leg is not None and leg.notes:
        note = f"{leg.notes} Verify schedules before travel."
    transfer_payload = {
        "legId": leg.id if leg is not None else None,
        "from": from_name,
        "to": to_name,
        "mode": mode,
        "estimatedDurationMinutes": duration,
        "estimatedDistanceKm": distance,
        "estimatedCost": cost,
        "bookingRequired": False,
        "notes": note,
        "warnings": ["Verify schedules before travel."],
    }
    return ItineraryItem(
        time="09:00",
        endTime=end_time,
        type="transfer",
        name=f"{mode.replace('_', ' ').title()} from {from_name} to {to_name}",
        note=note,
        transportMode=mode,
        durationMinutes=duration,
        transfer=transfer_payload,
        estimated_cost=cost,
    )


def _preferred_route_mode(request: GenerateItineraryRequest) -> str:
    planning = request.planning_constraints
    if planning is not None:
        blocked = {
            *planning.transport.avoid_modes,
            *planning.transport.disallowed_modes,
        }
        for mode in planning.transport.preferred_modes:
            if mode not in blocked:
                return mode
    prefs = request.transport_preferences or (request.route.preferences if request.route else None)
    if prefs and prefs.preferred_modes:
        avoid = set(prefs.avoid_modes)
        for mode in prefs.preferred_modes:
            if mode not in avoid:
                return mode
    return "train"


def _route_transfer_cost(
    leg, mode: str, distance: float | None, currency: str
) -> dict[str, object]:
    if leg is not None and leg.estimated_cost is not None:
        return leg.estimated_cost.model_dump(by_alias=True, exclude_none=True, mode="json")
    km = Decimal(str(distance if distance and distance > 0 else 100))
    multipliers = {
        "bus": Decimal("0.08"),
        "train": Decimal("0.12"),
        "flight": Decimal("0.15"),
        "boat": Decimal("0.20"),
        "ferry": Decimal("0.20"),
        "car": Decimal("0.18"),
        "rental_car": Decimal("0.18"),
        "public_transport": Decimal("0.10"),
    }
    amount = km * multipliers.get(mode, Decimal("0"))
    if mode == "flight":
        amount = max(Decimal("50"), amount)
    return {
        "amount": amount.quantize(Decimal("0.01")),
        "currency": currency,
        "category": "transport",
        "confidence": "medium",
        "source": "ai",
        "note": "Approximate transfer estimate.",
    }


def _route_style_note(request: GenerateItineraryRequest) -> str:
    styles = list(request.trip_styles)
    if request.route is not None:
        styles.extend(request.route.preferences.trip_styles)
    if request.planning_constraints is not None:
        styles.extend(request.planning_constraints.trip_styles)
    if "hiking" in styles:
        return "Keep hiking plans conservative and verify local maps; this is not GPS navigation."
    if "camping" in styles:
        return "Check campsite availability separately; no reservation is confirmed."
    if "island_hopping" in styles:
        return "Ferry and boat times are approximate; verify schedules locally."
    return "Keep the first local activity light so the route stays realistic after transfers."


def _pad_route_items(
    items: list[ItineraryItem],
    destination: str,
    currency: str,
    target_count: int,
) -> None:
    while len(items) < target_count:
        index = len(items)
        next_time = _next_route_time(items)
        if index % 2 == 0:
            items.append(
                ItineraryItem(
                    time=next_time,
                    type="rest",
                    name=f"Buffer time in {destination}",
                    note="Leave this block flexible for check-in, weather, or route delays.",
                    estimated_cost={"amount": 0, "currency": currency, "category": "other"},
                )
            )
        else:
            items.append(
                ItineraryItem(
                    time=next_time,
                    type="activity",
                    name=f"Short local stop in {destination}",
                    note="A low-pressure activity that can be skipped if travel runs late.",
                    estimated_cost={"amount": 0, "currency": currency, "category": "activity"},
                )
            )


def _next_route_time(items: list[ItineraryItem]) -> str:
    if not items:
        return "10:00"
    raw = items[-1].end_time or items[-1].time
    try:
        minutes = int(raw[:2]) * 60 + int(raw[3:5])
    except (ValueError, TypeError):
        minutes = 10 * 60
    minutes = min(23 * 60, minutes + 90)
    return f"{minutes // 60:02d}:{minutes % 60:02d}"


_TYPE_TO_COST_CATEGORY = {
    "food": "food",
    "transport": "transport",
    "transfer": "transport",
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
    return items


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


def _generation_repair_currency(request: RepairGenerationOutputRequest) -> str:
    raw = str(request.current_output.get("currency") or "").strip().upper()
    if len(raw) == 3:
        return raw
    trip_currency = str(
        request.planning_context.trip.get("BudgetCurrency")
        or request.planning_context.trip.get("budgetCurrency")
        or ""
    ).strip().upper()
    return trip_currency if len(trip_currency) == 3 else "EUR"


def _ensure_generation_output_shape(itinerary: dict, currency: str) -> None:
    days = itinerary.get("days")
    if not isinstance(days, list):
        itinerary["days"] = []
    if not str(itinerary.get("currency") or "").strip():
        itinerary["currency"] = currency


def _generation_days(itinerary: dict) -> list[dict]:
    return _repair_days(itinerary)


def _needs_day_count_repair(request: RepairGenerationOutputRequest) -> bool:
    for issue in request.validation_issues:
        issue_id = issue.id.casefold()
        if (
            "day_count" in issue_id
            or "missing_day" in issue_id
            or "duplicate_day" in issue_id
        ):
            return True
    return False


def _generation_expected_days(request: RepairGenerationOutputRequest) -> int:
    raw = request.planning_context.trip.get("Days") or request.planning_context.trip.get("days")
    if isinstance(raw, int):
        return max(raw, 0)
    if isinstance(raw, float):
        return max(int(raw), 0)
    try:
        return max(int(str(raw)), 0)
    except (TypeError, ValueError):
        return 0


def _normalize_generation_day_count(
    itinerary: dict,
    request: RepairGenerationOutputRequest,
    currency: str,
) -> list[GenerationRepairChange]:
    expected = _generation_expected_days(request)
    if expected <= 0:
        return []

    current_days = _generation_days(itinerary)
    existing_by_number: dict[int, dict] = {}
    for index, day in enumerate(current_days, start=1):
        number = _day_number(day) or index
        if number not in existing_by_number:
            existing_by_number[number] = day

    normalized: list[dict] = []
    added = 0
    renumbered = 0
    for number in range(1, expected + 1):
        day = existing_by_number.get(number)
        if day is None:
            day = _generation_placeholder_day(number, currency)
            added += 1
        if day.get("day") != number:
            renumbered += 1
        day["day"] = number
        if not str(day.get("title") or "").strip():
            day["title"] = f"Day {number}: repaired plan"
        _day_items(day)
        normalized.append(day)

    itinerary["days"] = normalized
    if added == 0 and renumbered == 0 and len(current_days) == expected:
        return []

    return [
        GenerationRepairChange(
            type="day_count_normalized",
            description=f"Normalized itinerary to {expected} day(s).",
            metadata={"addedDays": added, "renumberedDays": renumbered},
        )
    ]


def _generation_placeholder_day(day_number: int, currency: str) -> dict:
    return {
        "day": day_number,
        "title": f"Day {day_number}: flexible repaired plan",
        "items": [
            {
                "time": "10:00",
                "endTime": "11:30",
                "type": "activity",
                "name": "Flexible itinerary block",
                "note": "Added by AI repair to keep the trip duration complete.",
                "estimatedCost": {
                    "amount": 0,
                    "currency": currency,
                    "category": "activity",
                    "confidence": "low",
                    "source": "ai",
                },
            }
        ],
    }


def _repair_generation_schema_issue(
    itinerary: dict,
    issue: GenerationValidationIssue,
    currency: str,
) -> list[GenerationRepairChange]:
    day_number = issue.day_number
    item_index = issue.item_index
    changes: list[GenerationRepairChange] = []

    if day_number is not None:
        day = _generation_day_at(itinerary, day_number)
        if day is None:
            day = _generation_placeholder_day(day_number, currency)
            itinerary["days"].append(day)
            changes.append(
                GenerationRepairChange(
                    type="day_added",
                    description="Added missing day for schema repair.",
                    day_number=day_number,
                    metadata={"issueId": issue.id},
                )
            )
        if not str(day.get("title") or "").strip():
            day["title"] = f"Day {day_number}: repaired plan"
            changes.append(
                GenerationRepairChange(
                    type="day_title_added",
                    description="Filled missing day title.",
                    day_number=day_number,
                    metadata={"issueId": issue.id},
                )
            )
        items = _day_items(day)
        if item_index is not None:
            while len(items) <= item_index:
                items.append(_generation_placeholder_item(currency))
            item = items[item_index]
            before = _compact_item(item)
            _ensure_generation_item_schema(item, currency)
            after = _compact_item(item)
            if before != after:
                changes.append(
                    GenerationRepairChange(
                        type="item_schema_filled",
                        description="Filled missing item fields.",
                        day_number=day_number,
                        item_index=item_index,
                        metadata={"issueId": issue.id, "before": before, "after": after},
                    )
                )
        elif not items:
            items.append(_generation_placeholder_item(currency))
            changes.append(
                GenerationRepairChange(
                    type="item_added",
                    description="Added a placeholder item to a day with no items.",
                    day_number=day_number,
                    item_index=0,
                    metadata={"issueId": issue.id},
                )
            )
    elif not _generation_days(itinerary):
        itinerary["days"] = [_generation_placeholder_day(1, currency)]
        changes.append(
            GenerationRepairChange(
                type="day_added",
                description="Added a first itinerary day.",
                day_number=1,
                metadata={"issueId": issue.id},
            )
        )

    return changes


def _generation_placeholder_item(currency: str) -> dict:
    return {
        "time": "10:00",
        "endTime": "11:00",
        "type": "activity",
        "name": "Flexible repaired item",
        "note": "Added by AI repair to complete a required itinerary field.",
        "estimatedCost": {
            "amount": 0,
            "currency": currency,
            "category": "activity",
            "confidence": "low",
            "source": "ai",
        },
    }


def _ensure_generation_item_schema(item: dict, currency: str) -> None:
    if not str(item.get("time") or "").strip():
        item["time"] = "10:00"
    if not str(item.get("type") or "").strip():
        item["type"] = "activity"
    if not str(item.get("name") or "").strip():
        item["name"] = "Flexible repaired item"
    cost = item.get("estimatedCost")
    if isinstance(cost, dict):
        cost["currency"] = str(cost.get("currency") or currency).strip().upper() or currency


def _move_generation_item_after_transport(
    itinerary: dict,
    request: RepairGenerationOutputRequest,
    issue: GenerationValidationIssue,
) -> list[GenerationRepairChange]:
    if issue.day_number is None or issue.item_index is None:
        return []
    item = _item_at(itinerary, issue.day_number, issue.item_index)
    if item is None:
        return []
    leg = _generation_route_leg(request, issue)
    selected = leg.get("selectedTransportOption") if isinstance(leg, dict) else None
    arrival = selected.get("arrivalTime") if isinstance(selected, dict) else None
    new_start = _hhmm_add_minutes(str(arrival or "12:00"), 30)
    new_end = _hhmm_add_minutes(new_start, _generation_item_duration_minutes(item, 90))
    before = _compact_item(item)
    item["time"] = new_start
    item["endTime"] = new_end
    item["note"] = _append_note(
        item.get("note"),
        "AI repair moved this after the selected transport arrival.",
    )
    return [
        GenerationRepairChange(
            type="item_moved",
            description="Moved activity after selected transport arrival.",
            day_number=issue.day_number,
            item_index=issue.item_index,
            metadata={"issueId": issue.id, "before": before, "after": _compact_item(item)},
        )
    ]


def _add_generation_transfer_item(
    itinerary: dict,
    request: RepairGenerationOutputRequest,
    issue: GenerationValidationIssue,
    currency: str,
) -> list[GenerationRepairChange]:
    leg = _generation_route_leg(request, issue)
    if not isinstance(leg, dict):
        return []
    leg_id = str(leg.get("id") or issue.route_leg_id or "").strip()
    if not leg_id:
        return []
    target_day_number = issue.day_number or _generation_day_number_for_leg(itinerary, leg)
    if target_day_number is None:
        return []
    day = _generation_day_at(itinerary, target_day_number)
    if day is None:
        day = _generation_placeholder_day(target_day_number, currency)
        itinerary["days"].append(day)
    items = _day_items(day)
    if any(_item_has_transfer_leg(item, leg_id) for item in items):
        return []

    selected = leg.get("selectedTransportOption")
    selected = selected if isinstance(selected, dict) else {}
    departure_time = str(selected.get("departureTime") or "09:00")
    arrival_time = str(selected.get("arrivalTime") or "")
    mode = str(selected.get("mode") or leg.get("mode") or "other")
    from_name = str(leg.get("fromName") or selected.get("originName") or "origin")
    to_name = str(leg.get("toName") or selected.get("destinationName") or "destination")
    duration = leg.get("estimatedDurationMinutes") or selected.get("durationMinutes")
    amount = _generation_transport_cost_amount(selected, leg)
    transfer_item = {
        "time": departure_time,
        "type": "transport",
        "transportMode": mode,
        "name": f"Transfer from {from_name} to {to_name}",
        "note": "Added by AI repair to match selected route transport.",
        "estimatedCost": {
            "amount": amount,
            "currency": currency,
            "category": "transport",
            "confidence": "medium",
            "source": "ai",
        },
        "transfer": {
            "legId": leg_id,
            "from": from_name,
            "to": to_name,
            "mode": mode,
            "bookingRequired": bool(selected),
            "notes": "Verify provider details before travel.",
        },
    }
    if arrival_time:
        transfer_item["endTime"] = arrival_time
    if isinstance(duration, int) and duration > 0:
        transfer_item["durationMinutes"] = duration
        transfer_item["transfer"]["estimatedDurationMinutes"] = duration

    items.append(transfer_item)
    return [
        GenerationRepairChange(
            type="transfer_item_added",
            description="Added selected transport as an itinerary transfer item.",
            day_number=target_day_number,
            item_index=len(items) - 1,
            metadata={"issueId": issue.id, "routeLegId": leg_id},
        )
    ]


def _move_generation_item_to_opening_hours(
    itinerary: dict,
    issue: GenerationValidationIssue,
) -> list[GenerationRepairChange]:
    if issue.day_number is None or issue.item_index is None:
        return []
    item = _item_at(itinerary, issue.day_number, issue.item_index)
    if item is None:
        return []
    before = _compact_item(item)
    item["time"] = "10:00"
    if "endTime" in item:
        item["endTime"] = "11:30"
    item["note"] = _append_note(
        item.get("note"),
        "AI repair moved this to a safer daytime opening-hours window.",
    )
    return [
        GenerationRepairChange(
            type="item_moved",
            description="Moved item into a safer opening-hours window.",
            day_number=issue.day_number,
            item_index=issue.item_index,
            metadata={"issueId": issue.id, "before": before, "after": _compact_item(item)},
        )
    ]


def _reduce_generation_cost_risk(
    itinerary: dict,
    issue: GenerationValidationIssue,
    currency: str,
) -> list[GenerationRepairChange]:
    for day, item_index, item in _items_by_cost_desc(itinerary):
        amount = _repair_item_amount(item)
        if amount is None or amount <= 0:
            continue
        day_number = _day_number(day)
        before = _compact_item(item)
        reduced = max(Decimal("0"), (amount * Decimal("0.70")).quantize(Decimal("0.01")))
        _set_repair_item_amount(item, reduced, currency)
        item["name"] = _marked_name(item.get("name"), "lower-cost")
        item["note"] = _append_note(
            item.get("note"),
            "AI repair lowered this estimated cost for reliability validation.",
        )
        return [
            GenerationRepairChange(
                type="item_cost_reduced",
                description="Reduced estimated cost on the highest-cost item.",
                day_number=day_number,
                item_index=item_index,
                metadata={"issueId": issue.id, "before": before, "after": _compact_item(item)},
            )
        ]
    return []


def _generation_day_at(itinerary: dict, day_number: int) -> dict | None:
    for day in _generation_days(itinerary):
        if _day_number(day) == day_number:
            return day
    return None


def _generation_route_leg(
    request: RepairGenerationOutputRequest,
    issue: GenerationValidationIssue,
) -> dict | None:
    route = request.planning_context.route or {}
    legs = route.get("legs") if isinstance(route, dict) else None
    if not isinstance(legs, list):
        return None
    route_leg_id = issue.route_leg_id or _route_leg_id_from_issue(issue.id)
    for leg in legs:
        if isinstance(leg, dict) and str(leg.get("id") or "") == route_leg_id:
            return leg
    return None


def _route_leg_id_from_issue(issue_id: str) -> str:
    if ":" not in issue_id:
        return ""
    return issue_id.rsplit(":", 1)[-1].strip()


def _generation_day_number_for_leg(itinerary: dict, leg: dict) -> int | None:
    to_stop_id = str(leg.get("toStopId") or "")
    for day in _generation_days(itinerary):
        if to_stop_id and str(day.get("primaryStopId") or "") == to_stop_id:
            number = _day_number(day)
            if number is not None:
                return number
    days = _generation_days(itinerary)
    if days:
        return _day_number(days[0])
    return None


def _item_has_transfer_leg(item: dict, leg_id: str) -> bool:
    transfer = item.get("transfer")
    return isinstance(transfer, dict) and str(transfer.get("legId") or "") == leg_id


def _generation_transport_cost_amount(selected: dict, leg: dict) -> int | float:
    price = selected.get("estimatedPrice")
    if isinstance(price, dict) and price.get("amount") is not None:
        return price["amount"]
    cost = leg.get("estimatedCost")
    if isinstance(cost, dict) and cost.get("amount") is not None:
        return cost["amount"]
    return 0


def _generation_item_duration_minutes(item: dict, fallback: int) -> int:
    raw = item.get("durationMinutes")
    if isinstance(raw, int) and raw > 0:
        return raw
    start = _hhmm_to_minutes(str(item.get("time") or ""))
    end = _hhmm_to_minutes(str(item.get("endTime") or ""))
    if start is not None and end is not None and end > start:
        return end - start
    return fallback


def _hhmm_add_minutes(raw: str, minutes: int) -> str:
    base = _hhmm_to_minutes(raw)
    if base is None:
        base = 12 * 60
    value = min((24 * 60) - 1, max(0, base + minutes))
    return f"{value // 60:02d}:{value % 60:02d}"


def _hhmm_to_minutes(raw: str) -> int | None:
    if len(raw) < 5 or raw[2] != ":":
        return None
    try:
        hour = int(raw[:2])
        minute = int(raw[3:5])
    except ValueError:
        return None
    if hour < 0 or hour > 23 or minute < 0 or minute > 59:
        return None
    return hour * 60 + minute


def _sort_generation_day_items(day: dict) -> None:
    items = _day_items(day)

    def sort_key(item: dict) -> tuple[int, str]:
        minutes = _hhmm_to_minutes(str(item.get("time") or ""))
        return (minutes if minutes is not None else 24 * 60, str(item.get("name") or ""))

    items.sort(key=sort_key)


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
