from app.schemas.destination_context import DestinationContext
from app.schemas.itinerary import GenerateItineraryRequest

_ITEMS_PER_DAY_BY_PACE = {
    "relaxed": 3,
    "balanced": 4,
    "intensive": 5,
}


def build_itinerary_prompt(
    request: GenerateItineraryRequest,
    destination_context: DestinationContext | None = None,
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
          "estimatedCost": 18
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
{destination_context_section}

Rules:
- Generate exactly {request.days} day objects.
- Each day must have exactly {items_per_day} items.
- Use realistic times in HH:MM 24-hour format.
- Sort items inside each day by time ascending.
- Do not repeat the same time within a day.
- Do not repeat the same item name within a day.
- Use only these item types: place, food, activity, transport, rest.
- Include practical notes tailored to {request.destination}; avoid generic filler.
- Include estimatedCost as a number or null.
- Avoid hallucinated exact prices when uncertain; use reasonable estimates.
- Do not include fields outside the schema.
- Do not include any text outside the JSON.
""".strip()


def build_repair_prompt(
    request: GenerateItineraryRequest,
    invalid_response_text: str,
    validation_error: str,
    destination_context: DestinationContext | None = None,
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
{destination_context_section}

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
          "estimatedCost": 18
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
- Include estimatedCost as a non-negative number or null.
- Keep total estimated costs reasonable for the requested budget when a budget is provided.
- The corrected JSON should still use the destination context where relevant.
- Do not include fields outside the schema.
- Do not include any text outside the JSON.
""".strip()


def _items_per_day_for_pace(pace: str) -> int:
    return _ITEMS_PER_DAY_BY_PACE.get(pace, _ITEMS_PER_DAY_BY_PACE["balanced"])


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


def _should_include_hidden_gems(request: GenerateItineraryRequest) -> bool:
    if not request.interests:
        return True
    normalized_interests = {_normalize_interest(interest) for interest in request.interests}
    return "hidden_gems" in request.interests or "hidden gems" in normalized_interests


def _should_include_food_tips(request: GenerateItineraryRequest) -> bool:
    if not request.interests:
        return True
    return any(_normalize_interest(interest) == "food" for interest in request.interests)


def _normalize_interest(value: str) -> str:
    return value.strip().lower().replace("_", " ")
