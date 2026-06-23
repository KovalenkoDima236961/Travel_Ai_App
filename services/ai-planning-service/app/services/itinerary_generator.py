from decimal import Decimal
from typing import Protocol

from app.schemas.itinerary import (
    GenerateItineraryRequest,
    ItineraryDay,
    ItineraryItem,
    ItineraryResponse,
    RegenerateDayRequest,
    RegenerateDayResponse,
    RegenerateItemRequest,
    RegenerateItemResponse,
)


class ItineraryGenerator(Protocol):
    def generate(self, request: GenerateItineraryRequest) -> ItineraryResponse: ...
    def regenerate_day(self, request: RegenerateDayRequest) -> RegenerateDayResponse: ...
    def regenerate_item(self, request: RegenerateItemRequest) -> RegenerateItemResponse: ...


class MockItineraryGenerator:
    def generate(self, request: GenerateItineraryRequest) -> ItineraryResponse:
        days = [
            ItineraryDay(
                day=day_number,
                title=self._title_for_day(request, day_number),
                items=self._items_for_day(request, day_number),
            )
            for day_number in range(1, request.days + 1)
        ]
        return ItineraryResponse(days=days)

    def regenerate_day(self, request: RegenerateDayRequest) -> RegenerateDayResponse:
        destination = request.trip.destination
        cheap = _mentions(request.instruction, "cheap", "cheaper", "budget")
        relaxed = _mentions(request.instruction, "relaxed", "slow", "easy")
        first_time = "10:00" if relaxed else "09:30"
        lunch_cost = Decimal("10") if cheap else Decimal("16")

        return RegenerateDayResponse(
            day=ItineraryDay(
                day=request.day_number,
                title=f"Day {request.day_number}: refreshed {destination} plan",
                items=[
                    ItineraryItem(
                        time=first_time,
                        type="activity",
                        name=f"Updated {destination} neighborhood walk",
                        note="A focused replacement day that keeps the rest of the trip intact.",
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
                        type="rest" if relaxed else "place",
                        name="Quiet cafe break" if relaxed else "Updated signature stop",
                        note="Keeps timing realistic alongside the unchanged itinerary days.",
                        estimated_cost=Decimal("6"),
                    ),
                ],
            )
        )

    def regenerate_item(self, request: RegenerateItemRequest) -> RegenerateItemResponse:
        cheap = _mentions(request.instruction, "cheap", "cheaper", "budget")
        return RegenerateItemResponse(
            item=ItineraryItem(
                time="12:30",
                type="food",
                name="Budget local food option" if cheap else "Updated local option",
                note=(
                    f"Mock replacement for zero-based item index {request.item_index} "
                    f"on day {request.day_number}."
                ),
                estimated_cost=Decimal("9") if cheap else Decimal("15"),
            )
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
            return self._relaxed_items(destination, interests, day_number)
        if request.pace == "intensive":
            return self._intensive_items(destination, interests, day_number)
        return self._balanced_items(destination, interests, day_number)

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


def _normalize_interest(value: str) -> str:
    return value.strip().lower().replace(" ", "_")


def _mentions(value: str | None, *terms: str) -> bool:
    if not value:
        return False
    normalized = value.casefold()
    return any(term in normalized for term in terms)
