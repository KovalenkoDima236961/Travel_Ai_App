from datetime import timedelta

from app.schemas.destination_context import DestinationContext
from app.schemas.itinerary import (
    GenerateItineraryRequest,
    OpeningHoursInterval,
    RegenerateDayRequest,
    RegenerateItemRequest,
)
from app.schemas.knowledge import KnowledgeSearchResult

_ITEMS_PER_DAY_BY_PACE = {
    "relaxed": 3,
    "balanced": 4,
    "intensive": 5,
}


def build_itinerary_prompt(
    request: GenerateItineraryRequest,
    destination_context: DestinationContext | None = None,
    rag_chunks: list[KnowledgeSearchResult] | None = None,
) -> str:
    items_per_day = _items_per_day_for_pace(request.pace)
    interests = ", ".join(request.interests) if request.interests else "general sightseeing"
    budget = (
        f"{request.budget_amount} {request.budget_currency}"
        if request.budget_amount is not None
        else "not provided"
    )
    start_date = request.start_date.isoformat() if request.start_date else "not provided"
    destination_context_section = _destination_context_section(request, destination_context)
    rag_context_section = _rag_context_section(rag_chunks)
    user_context_section = _user_context_section(request)
    weather_context_section = _weather_context_section(request.weather_forecast)

    return f"""
You are generating an itinerary for a web-based travel planning application.

Return ONLY valid JSON. Do not include markdown, explanations, comments, or code fences.
The JSON must exactly match this schema and must not include any other fields:
{{
  "days": [
    {{
      "day": 1,
      "title": "string",
      "items": [
        {{
          "time": "09:00",
          "type": "place",
          "name": "string",
          "note": "string",
          "estimatedCost": {{
            "amount": 18,
            "currency": "EUR",
            "category": "ticket",
            "confidence": "medium",
            "source": "ai"
          }}
        }}
      ]
    }}
  ]
}}

Trip request:
- Destination: {request.destination}
- Start date: {start_date}
- Days: {request.days}
- Budget: {budget}
- Travelers: {request.travelers}
- Interests: {interests}
- Pace: {request.pace}
{user_context_section}
{weather_context_section}
{destination_context_section}
{rag_context_section}

Rules:
- Generate exactly {request.days} day objects.
- Each day must have exactly {items_per_day} items.
- Use realistic times in HH:MM 24-hour format.
- Sort items inside each day by time ascending.
- Do not repeat the same time within a day.
- Do not repeat the same item name within a day.
- Use only these item types: place, food, activity, transport, rest.
- Include practical notes tailored to {request.destination}; avoid generic filler.
- For each paid activity, museum/ticket, restaurant, cafe, transport, shopping, and
  accommodation item, include estimatedCost as an object with fields amount
  (non-negative number), currency (3-letter code), category
  (food|transport|ticket|activity|accommodation|shopping|other), confidence
  (low|medium|high), and source "ai". Use amount 0 for genuinely free stops.
- Use approximate realistic local costs in the requested currency. If uncertain,
  set estimatedCost to null instead of inventing an exact price.
- Avoid hallucinated exact prices when uncertain; use reasonable estimates.
- Respect user profile and travel preferences where possible.
- Prefer activities matching travelStyles and interests.
- If preferences conflict with the explicit trip request, prioritize the trip request first.
- Avoid preference items listed under Avoid unless necessary; if unavoidable, explain why
  in the item note.
- Keep walking-heavy days reasonable when maxWalkingKmPerDay is set.
- Prefer food recommendations matching foodPreferences and dietaryRestrictions.
- Use preferredCurrency for estimated costs when profile currency is available.
- Keep the response in English for now, but consider preferredLanguage as context.
- Prefer indoor activities during rainy days, avoid long outdoor walks during high heat,
  and schedule parks/viewpoints/walking-heavy activities on better weather days.
- Add indoor backup suggestions when rain chance is high.
- Do not mention weather excessively unless relevant.
- Do not include fields outside the schema.
- Do not include any text outside the JSON.
""".strip()


def build_repair_prompt(
    request: GenerateItineraryRequest,
    invalid_response_text: str,
    validation_error: str,
    destination_context: DestinationContext | None = None,
    rag_chunks: list[KnowledgeSearchResult] | None = None,
) -> str:
    items_per_day = _items_per_day_for_pace(request.pace)
    interests = ", ".join(request.interests) if request.interests else "general sightseeing"
    budget = (
        f"{request.budget_amount} {request.budget_currency}"
        if request.budget_amount is not None
        else "not provided"
    )
    start_date = request.start_date.isoformat() if request.start_date else "not provided"
    destination_context_section = _destination_context_section(request, destination_context)
    rag_context_section = _rag_context_section(rag_chunks)
    user_context_section = _user_context_section(request)
    weather_context_section = _weather_context_section(request.weather_forecast)

    return f"""
You previously generated an itinerary JSON response, but it was invalid.

Validation error:
{validation_error}

Original trip request:
- Destination: {request.destination}
- Start date: {start_date}
- Days: {request.days}
- Budget: {budget}
- Travelers: {request.travelers}
- Interests: {interests}
- Pace: {request.pace}
{user_context_section}
{weather_context_section}
{destination_context_section}
{rag_context_section}

Invalid previous response:
{invalid_response_text}

Return ONLY corrected JSON. Do not include markdown, explanations, comments, or code fences.
The corrected JSON must exactly match this schema and must not include any other fields:
{{
  "days": [
    {{
      "day": 1,
      "title": "string",
      "items": [
        {{
          "time": "09:00",
          "type": "place",
          "name": "string",
          "note": "string",
          "estimatedCost": {{
            "amount": 18,
            "currency": "EUR",
            "category": "ticket",
            "confidence": "medium",
            "source": "ai"
          }}
        }}
      ]
    }}
  ]
}}

Repair rules:
- Generate exactly {request.days} day objects.
- Day numbers must be 1 through {request.days} in order.
- Each day must have exactly {items_per_day} items.
- Use only these item types: place, food, activity, transport, rest.
- Use HH:MM 24-hour times.
- Sort items inside each day by time ascending.
- Do not repeat the same time within a day.
- Do not repeat the same item name within a day.
- Make every day title, item name, and item note non-empty.
- Include estimatedCost as an object {{amount, currency, category, confidence, source}}
  for paid items (amount non-negative, currency a 3-letter code, category one of
  food|transport|ticket|activity|accommodation|shopping|other, source "ai"), or null
  when there is no cost or you are uncertain. Use amount 0 for free stops.
- Keep total estimated costs reasonable for the requested budget when a budget is provided.
- Preserve personalization from the user profile and travel preferences where it fits the schema.
- Do not remove preference-aware details unless they caused the validation error or violate
  the schema.
- The corrected JSON should still use the destination context where relevant.
- The corrected JSON should still use the RAG context where relevant.
- Preserve weather-aware choices from the weather forecast where relevant.
- Do not include fields outside the schema.
- Do not include any text outside the JSON.
""".strip()


def build_regenerate_day_prompt(
    request: RegenerateDayRequest,
    destination_context: DestinationContext | None = None,
    rag_chunks: list[KnowledgeSearchResult] | None = None,
) -> str:
    selected_day = request.selected_day()
    selected_day_json = (
        selected_day.model_dump_json(by_alias=True, exclude_none=True) if selected_day else "{}"
    )
    instruction = request.instruction or "No extra user instruction provided."
    opening_hours_section = _attached_place_opening_hours_section(request)

    return f"""
You are regenerating exactly one itinerary day for a web-based travel planning application.

Return ONLY valid JSON. Do not include markdown, explanations, comments, or code fences.
The JSON must exactly match this schema and must not include any other fields:
{{
  "day": {{
    "day": {request.day_number},
    "title": "string",
    "items": [
      {{
        "time": "09:00",
        "type": "place",
        "name": "string",
        "note": "string",
        "estimatedCost": {{
          "amount": 18,
          "currency": "EUR",
          "category": "ticket",
          "confidence": "medium",
          "source": "ai"
        }}
      }}
    ]
  }}
}}

Trip request:
{_partial_trip_section(request)}
{_partial_user_context_section(request)}
{_weather_context_section(request.weather_forecast)}
{_partial_destination_context_section(destination_context)}
{_rag_context_section(rag_chunks)}
{opening_hours_section}

Current full itinerary JSON:
{request.current_itinerary.model_dump_json(by_alias=True, exclude_none=True)}

Selected day to replace:
{selected_day_json}

User instruction:
{instruction}

Rules:
- Replace only day {request.day_number}.
- The returned day.day must be {request.day_number}.
- Keep the new day consistent with the rest of the itinerary.
- Avoid duplicating activities already scheduled on other days.
- Respect user preferences and avoid list.
- Follow the user instruction if provided.
- Adapt the replacement day to the weather forecast when relevant.
- Include at least one item.
- Use realistic HH:MM 24-hour times.
- Sort items by time ascending.
- Use only these item types: place, food, activity, transport, rest.
- Include estimatedCost as an object {{amount, currency, category, confidence, source}}
  for paid items (amount non-negative, currency a 3-letter code, category one of
  food|transport|ticket|activity|accommodation|shopping|other, source "ai"), or null
  when there is no cost or you are uncertain. Use amount 0 for free stops.
- Do not include fields outside the schema.
- Do not include any text outside the JSON.
""".strip()


def build_regenerate_item_prompt(
    request: RegenerateItemRequest,
    destination_context: DestinationContext | None = None,
    rag_chunks: list[KnowledgeSearchResult] | None = None,
) -> str:
    selected_day = request.selected_day()
    selected_item = request.selected_item()
    selected_day_json = (
        selected_day.model_dump_json(by_alias=True, exclude_none=True) if selected_day else "{}"
    )
    selected_item_json = (
        selected_item.model_dump_json(by_alias=True, exclude_none=True) if selected_item else "{}"
    )
    instruction = request.instruction or "No extra user instruction provided."
    opening_hours_section = _attached_place_opening_hours_section(request)

    return f"""
You are regenerating exactly one itinerary item for a web-based travel planning application.

Return ONLY valid JSON. Do not include markdown, explanations, comments, or code fences.
The JSON must exactly match this schema and must not include any other fields:
{{
  "item": {{
    "time": "12:30",
    "type": "food",
    "name": "string",
    "note": "string",
    "estimatedCost": {{
      "amount": 15,
      "currency": "EUR",
      "category": "food",
      "confidence": "medium",
      "source": "ai"
    }}
  }}
}}

Trip request:
{_partial_trip_section(request)}
{_partial_user_context_section(request)}
{_weather_context_section(request.weather_forecast)}
{_partial_destination_context_section(destination_context)}
{_rag_context_section(rag_chunks)}
{opening_hours_section}

Current full itinerary JSON:
{request.current_itinerary.model_dump_json(by_alias=True, exclude_none=True)}

Selected day:
{selected_day_json}

Selected item to replace, zero-based itemIndex {request.item_index}:
{selected_item_json}

User instruction:
{instruction}

Rules:
- Replace only item index {request.item_index} in day {request.day_number}.
- Keep timing reasonable relative to neighboring items.
- Avoid duplicating existing itinerary items.
- Respect user preferences and avoid list.
- Follow the user instruction if provided.
- Adapt the replacement item to the weather forecast when relevant.
- Use only these item types: place, food, activity, transport, rest.
- Include estimatedCost as an object {{amount, currency, category, confidence, source}}
  for paid items (amount non-negative, currency a 3-letter code, category one of
  food|transport|ticket|activity|accommodation|shopping|other, source "ai"), or null
  when there is no cost or you are uncertain. Use amount 0 for free stops.
- Do not include fields outside the schema.
- Do not include any text outside the JSON.
""".strip()


def build_regenerate_day_repair_prompt(
    request: RegenerateDayRequest,
    invalid_response_text: str,
    validation_error: str,
    destination_context: DestinationContext | None = None,
    rag_chunks: list[KnowledgeSearchResult] | None = None,
) -> str:
    return f"""
You previously generated a replacement itinerary day JSON response, but it was invalid.

Validation error:
{validation_error}

Required schema:
{{
  "day": {{
    "day": {request.day_number},
    "title": "string",
    "items": [
      {{
        "time": "09:00",
        "type": "place",
        "name": "string",
        "note": "string",
        "estimatedCost": {{
          "amount": 18,
          "currency": "EUR",
          "category": "ticket",
          "confidence": "medium",
          "source": "ai"
        }}
      }}
    ]
  }}
}}

Trip request:
{_partial_trip_section(request)}
{_partial_user_context_section(request)}
{_weather_context_section(request.weather_forecast)}
{_partial_destination_context_section(destination_context)}
{_rag_context_section(rag_chunks)}

Invalid previous response:
{invalid_response_text}

Return ONLY corrected JSON for day {request.day_number}. Do not include markdown,
explanations, comments, code fences, or fields outside the schema.
""".strip()


def build_regenerate_item_repair_prompt(
    request: RegenerateItemRequest,
    invalid_response_text: str,
    validation_error: str,
    destination_context: DestinationContext | None = None,
    rag_chunks: list[KnowledgeSearchResult] | None = None,
) -> str:
    return f"""
You previously generated a replacement itinerary item JSON response, but it was invalid.

Validation error:
{validation_error}

Required schema:
{{
  "item": {{
    "time": "12:30",
    "type": "food",
    "name": "string",
    "note": "string",
    "estimatedCost": {{
      "amount": 15,
      "currency": "EUR",
      "category": "food",
      "confidence": "medium",
      "source": "ai"
    }}
  }}
}}

Trip request:
{_partial_trip_section(request)}
{_partial_user_context_section(request)}
{_weather_context_section(request.weather_forecast)}
{_partial_destination_context_section(destination_context)}
{_rag_context_section(rag_chunks)}

Invalid previous response:
{invalid_response_text}

Return ONLY corrected JSON for item index {request.item_index} in day {request.day_number}.
Do not include markdown, explanations, comments, code fences, or fields outside the schema.
""".strip()


def _items_per_day_for_pace(pace: str) -> int:
    return _ITEMS_PER_DAY_BY_PACE.get(pace, _ITEMS_PER_DAY_BY_PACE["balanced"])


def _partial_trip_section(request: RegenerateDayRequest) -> str:
    trip = request.trip
    budget = (
        f"{trip.budget_amount} {trip.budget_currency}"
        if trip.budget_amount is not None
        else "not provided"
    )
    start_date = trip.start_date.isoformat() if trip.start_date else "not provided"
    interests = ", ".join(trip.interests) if trip.interests else "general sightseeing"
    return "\n".join(
        [
            f"- Trip ID: {trip.id}",
            f"- Destination: {trip.destination}",
            f"- Start date: {start_date}",
            f"- Days: {trip.days}",
            f"- Budget: {budget}",
            f"- Travelers: {trip.travelers}",
            f"- Interests: {interests}",
            f"- Pace: {trip.pace}",
        ]
    )


def _partial_user_context_section(request: RegenerateDayRequest) -> str:
    profile = request.user_profile
    preferences = request.user_preferences
    if profile is None and preferences is None:
        return ""

    sections: list[str] = []
    if profile is not None:
        profile_lines = ["USER PROFILE:"]
        _append_optional_line(profile_lines, "Home city", profile.home_city)
        _append_optional_line(profile_lines, "Home country", profile.home_country)
        _append_optional_line(profile_lines, "Preferred currency", profile.preferred_currency)
        _append_optional_line(profile_lines, "Preferred language", profile.preferred_language)
        if len(profile_lines) > 1:
            sections.append("\n".join(profile_lines))

    if preferences is not None:
        preference_lines = ["USER TRAVEL PREFERENCES:"]
        _append_optional_line(
            preference_lines, "Travel styles", _display_list(preferences.travel_styles)
        )
        _append_optional_line(preference_lines, "Preferred pace", preferences.pace)
        if preferences.max_walking_km_per_day is not None:
            _append_optional_line(
                preference_lines,
                "Max walking distance per day",
                f"{preferences.max_walking_km_per_day:g} km",
            )
        _append_optional_line(
            preference_lines, "Food preferences", _display_list(preferences.food_preferences)
        )
        _append_optional_line(preference_lines, "Avoid", _display_list(preferences.avoid))
        _append_optional_line(
            preference_lines, "Preferred transport", _display_list(preferences.preferred_transport)
        )
        _append_optional_line(
            preference_lines, "Accommodation style", _display_list(preferences.accommodation_style)
        )
        _append_optional_line(
            preference_lines,
            "Dietary restrictions",
            _display_list(preferences.dietary_restrictions) or "none",
        )
        if len(preference_lines) > 1:
            sections.append("\n".join(preference_lines))

    if not sections:
        return ""
    return "\n" + "\n".join(sections)


def _partial_destination_context_section(destination_context: DestinationContext | None) -> str:
    if destination_context is None:
        return ""

    lines = ["DESTINATION CONTEXT:", f"- Destination: {destination_context.destination}"]
    sections: list[tuple[str, list[str]]] = [
        ("Local tips", destination_context.localTips),
        ("Hidden gems", destination_context.hiddenGems),
        ("Food tips", destination_context.foodTips),
        ("Avoid", destination_context.avoid),
        ("Transport tips", destination_context.transportTips),
        ("Budget tips", destination_context.budgetTips),
    ]
    for label, items in sections:
        trimmed_items = [item.strip() for item in items if item.strip()][:5]
        if not trimmed_items:
            continue
        lines.append(f"- {label}:")
        lines.extend(f"  - {item}" for item in trimmed_items)

    if len(lines) == 1:
        return ""
    return "\n" + "\n".join(lines)


def _weather_context_section(weather_forecast: object | None) -> str:
    if weather_forecast is None:
        return ""

    days = getattr(weather_forecast, "days", [])
    if not days:
        return ""

    lines = [
        "WEATHER FORECAST:",
        "Use this optional context to keep the itinerary realistic for weather conditions.",
    ]
    for day in days:
        warnings = getattr(day, "warnings", []) or []
        lines.append(
            "- "
            f"{day.date}: {day.summary}, "
            f"{day.temperature_min_c:g}-{day.temperature_max_c:g}C, "
            f"rain chance {day.precipitation_chance}%, "
            f"wind {day.wind_speed_kph:g} kph"
        )
        if warnings:
            lines.append(f"  Warnings: {_display_list(warnings)}")

    lines.extend(
        [
            "Weather instructions:",
            "- Prefer indoor activities during rainy periods or days.",
            "- Avoid long outdoor walks during high heat.",
            "- Schedule parks, viewpoints, and walking-heavy activities on better weather days.",
            "- Add indoor backup suggestions when rain chance is high.",
            "- If weather conflicts with user interests, preserve user goals but adapt timing "
            "or activity type.",
        ]
    )

    return "\n" + "\n".join(lines)


def _attached_place_opening_hours_section(request: RegenerateDayRequest) -> str:
    lines: list[str] = []
    for day in request.current_itinerary.days:
        for item in day.items:
            place = item.place
            if place is None or not place.opening_hours:
                continue

            place_name = place.name or item.name
            hours = _format_opening_hours_for_trip_day(request, day.day, place.opening_hours)
            lines.append(f"- Day {day.day}, {item.time}, {place_name}: {hours}")

    if not lines:
        return ""

    section = [
        "ATTACHED PLACE OPENING HOURS:",
        *lines[:20],
        "Opening hours instructions:",
        "- If keeping an attached place, do not schedule it outside its opening hours.",
        "- If replacing an item, prefer a realistic time for that place type.",
        "- If a place appears closed at the scheduled time, adjust the time or suggest an "
        "alternative.",
    ]
    return "\n" + "\n".join(section)


def _format_opening_hours_for_trip_day(
    request: RegenerateDayRequest,
    day_number: int,
    opening_hours: list[OpeningHoursInterval],
) -> str:
    if request.trip.start_date is None:
        return "opening hours available; trip start date not provided"

    trip_day = request.trip.start_date + timedelta(days=day_number - 1)
    day_of_week = trip_day.isoweekday()
    day_name = _format_day_of_week(day_of_week)
    intervals = [
        interval
        for interval in opening_hours
        if getattr(interval, "day_of_week", None) == day_of_week
    ]
    if not intervals:
        return f"{day_name} closed"
    return f"{day_name} {', '.join(_format_opening_interval(interval) for interval in intervals)}"


def _format_opening_interval(interval: OpeningHoursInterval) -> str:
    return f"{interval.open}\u2013{interval.close}"


def _format_day_of_week(day_of_week: int) -> str:
    return {
        1: "Monday",
        2: "Tuesday",
        3: "Wednesday",
        4: "Thursday",
        5: "Friday",
        6: "Saturday",
        7: "Sunday",
    }.get(day_of_week, "Unknown day")


def _destination_context_section(
    request: GenerateItineraryRequest,
    destination_context: DestinationContext | None,
) -> str:
    if destination_context is None:
        return ""

    sections: list[tuple[str, list[str]]] = [
        ("Local tips", destination_context.localTips),
    ]

    if _should_include_hidden_gems(request):
        sections.append(("Hidden gems", destination_context.hiddenGems))

    if _should_include_food_tips(request):
        sections.append(("Food tips", destination_context.foodTips))

    sections.extend(
        [
            ("Avoid", destination_context.avoid),
            ("Transport tips", destination_context.transportTips),
            ("Budget tips", destination_context.budgetTips),
        ]
    )

    lines = ["DESTINATION CONTEXT:", f"- Destination: {destination_context.destination}"]
    for label, items in sections:
        trimmed_items = [item.strip() for item in items if item.strip()][:5]
        if not trimmed_items:
            continue
        lines.append(f"- {label}:")
        lines.extend(f"  - {item}" for item in trimmed_items)

    if len(lines) == 1:
        return ""

    return "\n" + "\n".join(lines)


def _rag_context_section(rag_chunks: list[KnowledgeSearchResult] | None) -> str:
    if not rag_chunks:
        return ""

    lines = [
        "RAG CONTEXT:",
        "Use these retrieved local travel notes when relevant.",
        "Do not copy them blindly.",
        "Prefer them over generic assumptions.",
        "If a note conflicts with the request, follow the request.",
    ]
    for chunk in rag_chunks:
        content = _compact_content(chunk.content)
        if not content:
            continue
        lines.extend(
            [
                f"- Source: {chunk.source}",
                f"  Content: {content}",
            ]
        )

    if len(lines) == 5:
        return ""

    return "\n" + "\n".join(lines)


def _user_context_section(request: GenerateItineraryRequest) -> str:
    profile = request.user_profile
    preferences = request.user_preferences
    if profile is None and preferences is None:
        return ""

    sections: list[str] = []
    if profile is not None:
        profile_lines = ["USER PROFILE:"]
        _append_optional_line(profile_lines, "Home city", profile.home_city)
        _append_optional_line(profile_lines, "Home country", profile.home_country)
        _append_optional_line(profile_lines, "Preferred currency", profile.preferred_currency)
        _append_optional_line(profile_lines, "Preferred language", profile.preferred_language)
        if len(profile_lines) > 1:
            sections.append("\n".join(profile_lines))

    if preferences is not None:
        preference_lines = ["USER TRAVEL PREFERENCES:"]
        _append_optional_line(
            preference_lines,
            "Travel styles",
            _display_list(preferences.travel_styles),
        )
        _append_optional_line(preference_lines, "Preferred pace", preferences.pace)
        if preferences.max_walking_km_per_day is not None:
            _append_optional_line(
                preference_lines,
                "Max walking distance per day",
                f"{preferences.max_walking_km_per_day:g} km",
            )
        _append_optional_line(
            preference_lines,
            "Food preferences",
            _display_list(preferences.food_preferences),
        )
        _append_optional_line(preference_lines, "Avoid", _display_list(preferences.avoid))
        _append_optional_line(
            preference_lines,
            "Preferred transport",
            _display_list(preferences.preferred_transport),
        )
        _append_optional_line(
            preference_lines,
            "Accommodation style",
            _display_list(preferences.accommodation_style),
        )
        _append_optional_line(
            preference_lines,
            "Dietary restrictions",
            _display_list(preferences.dietary_restrictions) or "none",
        )
        if len(preference_lines) > 1:
            sections.append("\n".join(preference_lines))

    if not sections:
        return ""

    return "\n" + "\n".join(sections)


def _append_optional_line(lines: list[str], label: str, value: str | None) -> None:
    if value:
        lines.append(f"- {label}: {value}")


def _display_list(values: list[str]) -> str | None:
    cleaned = [value.strip().replace("_", " ") for value in values if value.strip()]
    if not cleaned:
        return None
    return ", ".join(cleaned)


def _compact_content(content: str, max_chars: int = 700) -> str:
    compacted = " ".join(content.split())
    if len(compacted) <= max_chars:
        return compacted
    return compacted[: max_chars - 3].rstrip() + "..."


def _should_include_hidden_gems(request: GenerateItineraryRequest) -> bool:
    preference_styles = request.user_preferences.travel_styles if request.user_preferences else []
    all_interests = [*request.interests, *preference_styles]
    if not all_interests:
        return True
    normalized_interests = {_normalize_interest(interest) for interest in all_interests}
    return "hidden gems" in normalized_interests


def _should_include_food_tips(request: GenerateItineraryRequest) -> bool:
    preference_styles = request.user_preferences.travel_styles if request.user_preferences else []
    food_preferences = request.user_preferences.food_preferences if request.user_preferences else []
    all_interests = [*request.interests, *preference_styles, *food_preferences]
    if not all_interests:
        return True
    normalized_interests = {_normalize_interest(interest) for interest in all_interests}
    return bool({"food", "local"} & normalized_interests)


def _normalize_interest(value: str) -> str:
    return value.strip().lower().replace("_", " ")
