from __future__ import annotations

from datetime import date, timedelta

from app.schemas.checklist import (
    ChecklistCategory,
    ChecklistItemType,
    ChecklistPriority,
    GenerateChecklistRequest,
    GeneratedChecklistItem,
    GeneratedChecklistResponse,
)


def generate_mock_checklist(request: GenerateChecklistRequest) -> GeneratedChecklistResponse:
    """Create a deterministic preparation checklist from trip context.

    This is intentionally practical but conservative: it suggests reviewable
    packing/preparation tasks and avoids claims about bookings, legal, medical,
    or shopping guarantees.
    """

    items: list[GeneratedChecklistItem] = []
    existing_keys = {
        _dedupe_key(item.title, item.category)
        for item in (request.existing_checklist.items if request.existing_checklist else [])
    }
    seen: set[str] = set()

    def add(
        title: str,
        category: ChecklistCategory,
        item_type: ChecklistItemType,
        priority: ChecklistPriority,
        description: str,
        reason: str,
        *,
        quantity: int | None = None,
        due_offset_days: int | None = None,
        related_day_number: int | None = None,
        related_item_index: int | None = None,
    ) -> None:
        if not _category_allowed(category, request):
            return
        key = _dedupe_key(title, category)
        if key in seen or key in existing_keys:
            return
        seen.add(key)
        due_date = None
        departure = _departure_date(request)
        if departure is not None and due_offset_days is not None:
            due_date = departure - timedelta(days=due_offset_days)
        items.append(
            GeneratedChecklistItem(
                title=title,
                description=description,
                category=category,
                itemType=item_type,
                priority=priority,
                quantity=quantity,
                dueDate=due_date,
                reason=reason,
                relatedDayNumber=related_day_number,
                relatedItemIndex=related_item_index,
                metadata={"generator": "mock"},
            )
        )

    add(
        "Passport, ID, or required travel documents",
        "documents",
        "document",
        "critical",
        "Confirm documents are valid for the whole trip and stored somewhere accessible.",
        "Core document check for any trip.",
        due_offset_days=1,
    )
    add(
        "Phone charger and power bank",
        "electronics",
        "packing",
        "high",
        "Pack chargers for daily navigation, bookings, and emergency contact.",
        "Electronics basics are important for modern travel.",
    )
    add(
        "Payment card and small cash reserve",
        "money",
        "packing",
        "high",
        "Carry at least one backup way to pay and a small local cash buffer where useful.",
        "Money access can be a trip blocker if one payment method fails.",
    )
    add(
        "Travel insurance and emergency contact details",
        "health_safety",
        "safety_check",
        "high",
        "Save policy details and emergency contacts offline before departure.",
        "Safety information should be reachable even without connectivity.",
        due_offset_days=1,
    )
    add(
        "Confirm accommodation address and check-in instructions",
        "accommodation",
        "booking_check",
        "high",
        "Save the address, check-in window, and host or reception contact offline.",
        "Arrival logistics are easier when accommodation details are ready.",
        due_offset_days=1,
    )
    add(
        "Offline maps and key booking references",
        "before_departure",
        "preparation",
        "medium",
        "Download maps and save booking references before leaving reliable Wi-Fi.",
        "Offline access reduces friction during transfers and arrival.",
    )

    modes = _transport_modes(request)
    if "flight" in modes:
        add(
            "Flight documents and airport timing check",
            "transport",
            "booking_check",
            "critical",
            "Review airline requirements, baggage allowance, terminal, and airport transfer time.",
            "The trip includes air travel.",
        )
    if modes & {"train", "bus", "public_transport"}:
        add(
            "Transit tickets, passes, and schedule backups",
            "transport",
            "booking_check",
            "high",
            "Save ticket QR codes, station names, and backup departure options.",
            "The route or itinerary includes public transport.",
        )
    if modes & {"car", "rental_car"}:
        add(
            "Driving documents and rental pickup check",
            "transport",
            "document",
            "high",
            "Check license, rental pickup rules, parking plan, tolls, and fuel or charging stops.",
            "The route uses a car or rental car.",
        )
    if modes & {"boat", "ferry"}:
        add(
            "Ferry or boat schedule verification",
            "transport",
            "booking_check",
            "high",
            "Verify departure port, seasonal schedule, luggage rules, and weather contingency.",
            "The route includes ferry or boat travel.",
        )

    styles = _trip_styles(request)
    if "hiking" in styles or "hiking" in modes:
        add(
            "Hiking layers and trail-ready footwear",
            "camping_hiking",
            "packing",
            "high",
            "Pack broken-in shoes, layers, socks, and a small daypack for trail days.",
            "The trip context includes hiking.",
        )
        add(
            "Trail safety basics",
            "health_safety",
            "safety_check",
            "high",
            "Share plans, carry water, and verify route difficulty and conditions locally.",
            "Hiking plans need conservative preparation.",
        )
    if styles & {"camping", "backpacking"} or _is_camping_accommodation(request):
        add(
            "Camping sleep setup",
            "camping_hiking",
            "packing",
            "high",
            "Pack or verify tent, sleeping bag, sleeping pad, and campsite lighting.",
            "Camping-style trips need sleep gear checks.",
        )
        add(
            "Campsite cooking and water plan",
            "food_water",
            "preparation",
            "medium",
            "Confirm whether cooking gear, safe water, and food storage are needed.",
            "Camping and backpacking require basic food and water planning.",
        )

    weather = _weather_flags(request)
    if weather["rain"]:
        add(
            "Rain jacket or compact umbrella",
            "weather",
            "packing",
            "medium",
            "Pack lightweight rain protection and a dry bag or pouch for electronics.",
            "Forecast or conditions suggest rain risk.",
        )
    if weather["hot"]:
        add(
            "Sun protection and refillable water bottle",
            "weather",
            "packing",
            "medium",
            "Pack sunscreen, sunglasses or hat, and a reusable bottle for hot days.",
            "Forecast suggests heat or strong sun exposure.",
        )
    if weather["cold"]:
        add(
            "Warm layers",
            "clothing",
            "packing",
            "medium",
            "Pack layers suitable for cold mornings or evenings.",
            "Forecast suggests low temperatures.",
        )

    for day_number, item_index, name in _activity_items(request):
        add(
            f"Prepare for {name}",
            "activities",
            "preparation",
            "medium",
            "Check timing, dress requirements, tickets, and weather suitability before that day.",
            "Generated from an itinerary activity.",
            related_day_number=day_number,
            related_item_index=item_index,
        )

    return GeneratedChecklistResponse(
        title="Packing & preparation checklist",
        summary=(
            f"Generated {len(items)} practical packing and preparation items for "
            f"{request.trip.destination}."
        ),
        items=items[:100],
        warnings=[
            "Checklist items are planning suggestions; verify bookings, local rules, "
            "and health needs independently."
        ],
    )


def _category_allowed(category: str, request: GenerateChecklistRequest) -> bool:
    options = request.generation_options
    if options.mode != "category":
        return True
    selected = set(options.categories)
    return category in selected


def _dedupe_key(title: str, category: str) -> str:
    return f"{category}:{' '.join(title.casefold().split())}"


def _transport_modes(request: GenerateChecklistRequest) -> set[str]:
    modes: set[str] = set()
    route = request.route
    if route is not None:
        modes.update(route.preferences.preferred_modes)
        modes.update(leg.mode for leg in route.legs)
    if request.user_preferences is not None:
        modes.update(request.user_preferences.preferred_transport)
    constraints = request.planning_constraints
    if constraints is not None:
        modes.update(getattr(constraints.transport, "preferred_modes", []) or [])

    if request.itinerary is not None:
        for day in request.itinerary.days:
            for item in day.items:
                if item.transport_mode:
                    modes.add(item.transport_mode)
                if item.transfer is not None:
                    modes.add(item.transfer.mode)
                text = f"{item.type} {item.name} {item.note or ''}".casefold()
                for mode in (
                    "flight",
                    "train",
                    "bus",
                    "car",
                    "rental_car",
                    "boat",
                    "ferry",
                    "hiking",
                ):
                    if mode in text:
                        modes.add(mode)
    return {mode.strip().lower() for mode in modes if mode}


def _departure_date(request: GenerateChecklistRequest):
    constraints = request.planning_constraints
    if constraints is not None and constraints.dates.start_date:
        return date.fromisoformat(constraints.dates.start_date)
    return request.trip.start_date


def _trip_styles(request: GenerateChecklistRequest) -> set[str]:
    styles = {item.strip().lower() for item in request.trip.interests if item.strip()}
    route = request.route
    if route is not None:
        styles.update(route.preferences.trip_styles)
    if request.user_preferences is not None:
        styles.update(request.user_preferences.travel_styles)
    constraints = request.planning_constraints
    if constraints is not None:
        styles.update(constraints.trip_styles)
        styles.update(constraints.interests)
    instructions = request.generation_options.instructions or ""
    for token in ("hiking", "camping", "backpacking", "nature", "adventure"):
        if token in instructions.casefold():
            styles.add(token)
    return styles


def _is_camping_accommodation(request: GenerateChecklistRequest) -> bool:
    accommodation = request.accommodation
    if accommodation is None:
        return False
    return accommodation.type in {"campsite", "cabin", "campervan"}


def _weather_flags(request: GenerateChecklistRequest) -> dict[str, bool]:
    result = {"rain": False, "hot": False, "cold": False}
    if request.weather is None:
        return result
    for day in request.weather.days:
        text = f"{day.condition} {day.summary} {' '.join(day.warnings)}".casefold()
        if day.precipitation_chance >= 50 or "rain" in text or "storm" in text:
            result["rain"] = True
        if day.temperature_max_c >= 30 or "heat" in text or "sun" in text:
            result["hot"] = True
        if day.temperature_min_c <= 5 or "cold" in text or "snow" in text:
            result["cold"] = True
    return result


def _activity_items(request: GenerateChecklistRequest) -> list[tuple[int, int, str]]:
    if request.itinerary is None:
        return []
    selected: list[tuple[int, int, str]] = []
    for day in request.itinerary.days:
        for index, item in enumerate(day.items):
            if item.type not in {"activity", "place", "transport"}:
                continue
            text = f"{item.name} {item.note or ''}".casefold()
            if any(
                token in text for token in ("museum", "ticket", "tour", "hike", "boat", "ferry")
            ):
                selected.append((day.day, index, item.name))
            if len(selected) >= 6:
                return selected
    return selected
