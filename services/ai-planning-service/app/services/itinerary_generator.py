from decimal import Decimal

from app.schemas.itinerary import (
    GenerateItineraryRequest,
    ItineraryDay,
    ItineraryItem,
    ItineraryResponse,
)


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

    def _title_for_day(self, request: GenerateItineraryRequest, day_number: int) -> str:
        interests = set(request.interests)
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
        interests = set(request.interests)
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
        if "food" in interests:
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

        if "hidden_gems" in interests:
            return ItineraryItem(
                time=time,
                type="activity",
                name="Local neighborhood walk",
                note=(
                    f"Explore a quieter {destination} neighborhood with independent "
                    "shops and small cafes."
                ),
                estimated_cost=Decimal("5"),
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
