import json
from datetime import timedelta

from app.schemas.destination_context import DestinationContext
from app.schemas.destination_suggestion import DestinationSuggestionRequest
from app.schemas.itinerary import (
    GenerateItineraryRequest,
    OpeningHoursInterval,
    OptimizeBudgetDayRequest,
    RegenerateDayRequest,
    RegenerateItemRequest,
)
from app.schemas.knowledge import KnowledgeSearchResult
from app.schemas.repair import RepairItineraryRequest
from app.schemas.route_alternatives import RouteAlternativeRequest
from app.schemas.template_adaptation import TemplateAdaptationRequest


def build_destination_suggestion_prompt(request: DestinationSuggestionRequest) -> str:
    payload = request.model_dump(by_alias=True, exclude_none=True, mode="json")
    planning_constraints_section = _planning_constraints_section(request)
    return f"""
You are a careful destination recommendation assistant.

Return strict JSON only, matching this shape:
{{
  "sessionTitle": "string",
  "suggestions": [{{
    "id": "stable-slug",
    "suggestionType": "single_destination|route",
    "destination": "City, Country",
    "city": "string",
    "country": "string",
    "region": "string or null",
    "matchScore": 0,
    "recommendedDurationDays": 1,
    "bestFor": ["string"],
    "estimatedBudget": {{"amount": 0, "currency": "EUR", "confidence": "low|medium|high"}},
    "bestTimeToGo": "string",
    "whyItFits": "string",
    "possibleDownsides": ["string"],
    "tripPreview": {{"title": "string", "summary": "string", "sampleDay": ["string"]}},
    "tags": ["string"],
    "suggestedPromptForItinerary": "string",
    "route": {{
      "origin": {{"name": "string", "country": "string"}},
      "stops": [],
      "legs": [],
      "preferences": {{"preferredModes": ["train"], "tripStyles": ["train_trip"]}}
    }},
    "concerns": [{{"type": "string", "message": "string"}}]
  }}],
  "followUpQuestions": ["string"],
  "warnings": ["string"]
}}

Rules:
- Return 3 to {request.constraints.suggestion_count} plausible, distinct suggestions.
- You may return route suggestions when the user asks for multi-city, road trip,
  train trip, backpacking, hiking, camping, island hopping, or route-style travel.
- For route suggestions, set suggestionType="route", include route stops and transfer
  legs with approximate modes, durations, and costs, and never claim bookings or live schedules.
- Treat matchScore as an estimate and clamp it to 0-100.
- Never claim live prices, availability, booking, visa, legal, health, or safety guarantees.
- Budget values are rough estimates only.
- Consider preferences, previous-trip summaries, budget, season, origin, and workspace policy.
- When avoidPreviouslyVisited is true, do not repeat a previous destination.
- In refine mode, follow the refinement instruction and avoid rejected suggestions unless the
  user explicitly asks for similar places.
- Explain fit and downsides, and include suggestedPromptForItinerary.
- Keep JSON keys and enum values in English.
- Localize all user-facing text values to outputLanguage={request.output_language}.
- Do not suggest unsafe or illegal travel.

{planning_constraints_section}
Trusted sanitized request context:
{json.dumps(payload, ensure_ascii=False, indent=2)}
""".strip()


def build_route_alternatives_prompt(request: RouteAlternativeRequest) -> str:
    payload = request.model_dump(by_alias=True, exclude_none=True, mode="json")
    planning_constraints_section = _planning_constraints_section(request)
    return f"""
You are a route alternatives engine for a web-based AI travel planning app.

Return strict JSON only, matching this shape:
{{
  "sessionTitle": "string",
  "alternatives": [{{
    "id": "stable-slug",
    "title": "string",
    "summary": "string",
    "route": {{
      "origin": {{"name": "string", "country": "string", "coordinates": {{"lat": 0, "lng": 0}}}},
      "returnToOrigin": false,
      "stops": [{{
        "id": "stop_1",
        "destination": "string",
        "city": "string",
        "country": "string",
        "arrivalDate": "YYYY-MM-DD",
        "departureDate": "YYYY-MM-DD",
        "nights": 1,
        "coordinates": {{"lat": 0, "lng": 0}},
        "accommodationHint": "hotel|hostel|apartment|guesthouse|campsite|cabin|other",
        "notes": "string"
      }}],
      "legs": [{{
        "id": "leg_1",
        "fromStopId": "origin",
        "toStopId": "stop_1",
        "fromName": "string",
        "toName": "string",
        "mode": "walk|car|rental_car|train|bus|flight|boat|public_transport|other",
        "departureDate": "YYYY-MM-DD",
        "estimatedDurationMinutes": 0,
        "estimatedDistanceKm": 0,
        "estimatedCost": {{
          "amount": 0,
          "currency": "EUR",
          "category": "transport",
          "confidence": "low|medium|high",
          "source": "ai"
        }},
        "notes": "string"
      }}],
      "preferences": {{
        "preferredModes": ["train"],
        "avoidModes": ["flight"],
        "carAvailable": false,
        "maxTransferHoursPerDay": 4,
        "tripStyles": ["train_trip", "nature"]
      }}
    }},
    "scores": {{
      "overallFit": 0,
      "budgetFit": 0,
      "timeEfficiency": 0,
      "relaxation": 0,
      "nature": 0,
      "culture": 0,
      "transportSimplicity": 0,
      "policyCompliance": 0
    }},
    "estimatedBudget": {{"amount": 0, "currency": "EUR", "confidence": "medium"}},
    "estimatedTransferMinutes": 0,
    "estimatedTransferCost": {{"amount": 0, "currency": "EUR", "confidence": "medium"}},
    "difficulty": "relaxed|balanced|intense|rushed",
    "bestFor": ["string"],
    "pros": ["string"],
    "cons": ["string"],
    "warnings": ["string"],
    "suggestedItineraryPrompt": "string"
  }}],
  "comparisonSummary": {{
    "cheapestAlternativeId": "stable-slug",
    "mostRelaxedAlternativeId": "stable-slug",
    "bestNatureAlternativeId": "stable-slug",
    "bestOverallAlternativeId": "stable-slug"
  }},
  "followUpQuestions": ["string"],
  "warnings": ["string"]
}}

Rules:
- Generate route alternatives, not a detailed day-by-day itinerary.
- Return 1 to {request.suggestion_count} plausible, distinct alternatives.
- Respect duration, budget, origin, currentRoute, refinement, and planningConstraints.
- Prefer workspace policy over user preferences when conflicts exist.
- Suggest fewer stops if duration is short or budget is low.
- Avoid impossible routes and avoid disallowed transport modes.
- Include route stops and connecting legs with transport mode per leg.
- Use approximate estimates only; do not claim live schedules, live prices, bookings,
  reservations, ticket purchase, permits, or availability.
- Include clear pros, cons, and warnings for each alternative.
- Keep JSON keys and enum values in English.
- Localize user-facing text values to outputLanguage={request.output_language}.

{planning_constraints_section}
Trusted sanitized request context:
{json.dumps(payload, ensure_ascii=False, indent=2)}
""".strip()


def build_route_alternatives_repair_prompt(
    request: RouteAlternativeRequest,
    invalid_response_text: str,
    validation_error: str,
) -> str:
    payload = request.model_dump(by_alias=True, exclude_none=True, mode="json")
    return f"""
Repair this invalid route alternatives JSON response.

Return strict JSON only. Keep the same response shape required by
build_route_alternatives_prompt. Do not add markdown.

Validation error:
{validation_error}

Original trusted request context:
{json.dumps(payload, ensure_ascii=False, indent=2)}

Invalid response:
{invalid_response_text}
""".strip()


_ITEMS_PER_DAY_BY_PACE = {
    "relaxed": 3,
    "balanced": 4,
    "intensive": 5,
}

_LANGUAGE_NAMES = {
    "en": "English",
    "es": "Spanish",
    "uk": "Ukrainian",
    "fr": "French",
}


def _output_language_section(request: object) -> str:
    code = getattr(request, "output_language", "en")
    language_name = _LANGUAGE_NAMES.get(code, "English")
    return f"""
OUTPUT LANGUAGE:
- Write every user-facing text value in {language_name}.
- This includes titles, names, descriptions, notes, summaries, warnings, reasons,
  recommendations, and tradeoffs.
- Keep all JSON keys and enum values in English.
- Keep currency codes unchanged.
- Keep proper nouns and place names in their common or local form when appropriate.
- Do not mix languages unless a proper noun is naturally written in another language.
""".strip()


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
    accommodation_context_section = _accommodation_context_section(request.accommodation)
    workspace_policy_section = _workspace_policy_section(request)
    planning_constraints_section = _planning_constraints_section(request)
    route_context_section = _route_context_section(request)
    instruction = request.instruction or "No extra user instruction provided."

    return f"""
You are generating an itinerary for a web-based travel planning application.

Return ONLY valid JSON. Do not include markdown, explanations, comments, or code fences.
The JSON must exactly match this schema and must not include any other fields:
{{
  "days": [
    {{
      "day": 1,
      "date": "2026-09-10",
      "title": "string",
      "primaryStopId": "stop_1",
      "locationName": "City",
      "transferDay": false,
      "items": [
        {{
          "time": "09:00",
          "endTime": "10:30",
          "type": "place",
          "name": "string",
          "note": "string",
          "transportMode": "train",
          "durationMinutes": 90,
          "transfer": null,
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
- User instruction: {instruction}
{_output_language_section(request)}
{planning_constraints_section}
{user_context_section}
{weather_context_section}
{accommodation_context_section}
{workspace_policy_section}
{route_context_section}
{destination_context_section}
{rag_context_section}

Rules:
- Generate exactly {request.days} day objects.
- Each day must have exactly {items_per_day} items.
- Use realistic times in HH:MM 24-hour format.
- Sort items inside each day by time ascending.
- Do not repeat the same time within a day.
- Do not repeat the same item name within a day.
- Use only these item types: place, food, activity, transport, transfer, rest.
- For multi-destination route trips, plan across all route stops, set primaryStopId,
  locationName and transferDay where relevant, and include one transfer item for each
  transfer day. Do not schedule dense sightseeing before or after long transfers.
- Transfer items must include transfer {{legId, from, to, mode, estimatedDurationMinutes,
  estimatedDistanceKm, estimatedCost, bookingRequired, notes, warnings}}.
- Respect route arrival/departure dates, nights per stop, preferredModes, avoidModes,
  maxTransferHoursPerDay, and tripStyles where possible.
- For camping trips, include campsite/accommodation-style notes without claiming
  reservations. For hiking trips, keep planning conservative and do not generate
  technical GPS routes. For ferry/boat/island hopping, say schedules are approximate
  and must be verified.
- Include practical notes tailored to {request.destination}; avoid generic filler.
- For each paid activity, museum/ticket, restaurant, cafe, transport, shopping, and
  accommodation item, include estimatedCost as an object with fields amount
  (non-negative number), currency (3-letter code), category
  (food|transport|ticket|activity|accommodation|shopping|other), confidence
  (low|medium|high), and source "ai". Use amount 0 for genuinely free stops.
- Prefer the requested budget currency or preferredCurrency for estimatedCost.currency.
  If a local currency is more natural and known, use a valid uppercase 3-letter code.
- Use approximate realistic costs only. Do not invent exact exchange rates, do not
  claim financial accuracy, and set estimatedCost to null when uncertain.
- Respect user profile and travel preferences where possible.
- Prefer activities matching travelStyles and interests.
- If preferences conflict with the explicit trip request, prioritize the trip request first.
- Follow the user instruction when provided, while preserving all safety and schema rules.
- Avoid preference items listed under Avoid unless necessary; if unavoidable, explain why
  in the item note.
- Keep walking-heavy days reasonable when maxWalkingKmPerDay is set.
- Prefer food recommendations matching foodPreferences and dietaryRestrictions.
- Use preferredCurrency for estimated costs when profile currency is available.
- Prefer indoor activities during rainy days, avoid long outdoor walks during high heat,
  and schedule parks/viewpoints/walking-heavy activities on better weather days.
- Add indoor backup suggestions when rain chance is high.
- Do not mention weather excessively unless relevant.
- Do not claim tickets, transport, accommodation, campsite, ferry, train, bus, or flight
  bookings are confirmed.
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
    accommodation_context_section = _accommodation_context_section(request.accommodation)
    workspace_policy_section = _workspace_policy_section(request)
    planning_constraints_section = _planning_constraints_section(request)
    route_context_section = _route_context_section(request)

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
{_output_language_section(request)}
{planning_constraints_section}
{user_context_section}
{weather_context_section}
{accommodation_context_section}
{workspace_policy_section}
{route_context_section}
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
      "date": "2026-09-10",
      "title": "string",
      "primaryStopId": "stop_1",
      "locationName": "City",
      "transferDay": false,
      "items": [
        {{
          "time": "09:00",
          "endTime": "10:30",
          "type": "place",
          "name": "string",
          "note": "string",
          "transportMode": "train",
          "durationMinutes": 90,
          "transfer": null,
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
- Use only these item types: place, food, activity, transport, transfer, rest.
- Preserve route stop assignment, transferDay metadata, and transfer items for
  multi-destination route trips. Transfer items must not claim confirmed bookings.
- Use HH:MM 24-hour times.
- Sort items inside each day by time ascending.
- Do not repeat the same time within a day.
- Do not repeat the same item name within a day.
- Make every day title, item name, and item note non-empty.
- Include estimatedCost as an object {{amount, currency, category, confidence, source}}
  for paid items (amount non-negative, currency a 3-letter code, category one of
  food|transport|ticket|activity|accommodation|shopping|other, source "ai"), or null
  when there is no cost or you are uncertain. Use amount 0 for free stops.
- Prefer the requested budget currency or preferredCurrency; local currency is acceptable
  only when natural and known. Do not invent exact exchange rates.
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
    planning_constraints_section = _planning_constraints_section(request)

    return f"""
You are regenerating exactly one itinerary day for a web-based travel planning application.

{_output_language_section(request)}

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
{planning_constraints_section}
{_partial_user_context_section(request)}
{_weather_context_section(request.weather_forecast)}
{_accommodation_context_section(request.accommodation)}
{_workspace_policy_section(request)}
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
- Prefer the trip budget/preferred currency; local currency is acceptable only when
  natural and known. Do not invent exact exchange rates.
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
    planning_constraints_section = _planning_constraints_section(request)

    return f"""
You are regenerating exactly one itinerary item for a web-based travel planning application.

{_output_language_section(request)}

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
{planning_constraints_section}
{_partial_user_context_section(request)}
{_weather_context_section(request.weather_forecast)}
{_accommodation_context_section(request.accommodation)}
{_workspace_policy_section(request)}
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
- Prefer the trip budget/preferred currency; local currency is acceptable only when
  natural and known. Do not invent exact exchange rates.
- Do not include fields outside the schema.
- Do not include any text outside the JSON.
""".strip()


def build_optimize_budget_day_prompt(request: OptimizeBudgetDayRequest) -> str:
    instruction = request.instruction or "No extra user instruction provided."
    selected_day = request.selected_day()
    selected_day_json = (
        selected_day.model_dump_json(by_alias=True, exclude_none=True) if selected_day else "{}"
    )
    planning_constraints_section = _planning_constraints_section(request)

    return f"""
You are creating a reviewable budget optimization proposal for one itinerary day.

{_output_language_section(request)}

Return ONLY valid JSON. Do not include markdown, explanations, comments, or code fences.
The JSON must exactly match this schema and must not include any other fields:
{{
  "summary": "string",
  "scope": "day",
  "dayNumber": {request.day_number},
  "currency": "{request.budget_context.currency}",
  "baseDayEstimatedTotal": 185,
  "proposedDayEstimatedTotal": 113,
  "estimatedSavingsAmount": 72,
  "confidence": "medium",
  "changes": [
    {{
      "type": "replace_item",
      "oldItemIndex": 1,
      "oldItemName": "string",
      "newItemName": "string",
      "reason": "string",
      "estimatedSavingsAmount": 35,
      "currency": "{request.budget_context.currency}"
    }}
  ],
  "preservedItems": [
    {{
      "itemIndex": 0,
      "itemName": "string",
      "reason": "string"
    }}
  ],
  "tradeoffs": ["string"],
  "warnings": ["string"],
  "proposedDay": {{
    "day": {request.day_number},
    "title": "string",
    "items": [
      {{
        "time": "09:00",
        "type": "place",
        "name": "string",
        "note": "string",
        "estimatedCost": {{
          "amount": 10,
          "currency": "{request.budget_context.currency}",
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
{planning_constraints_section}
{_partial_user_context_section(request)}
{_weather_context_section(request.weather_forecast)}
{_accommodation_context_section(request.accommodation)}
{_workspace_policy_section(request)}

Budget context:
{request.budget_context.model_dump_json(by_alias=True, exclude_none=True)}

Optimization constraints:
{request.constraints.model_dump_json(by_alias=True, exclude_none=True)}

Current full itinerary JSON:
{request.current_itinerary.model_dump_json(by_alias=True, exclude_none=True)}

Selected day to optimize:
{selected_day_json}

User instruction:
{instruction}

Rules:
- Optimize only day {request.day_number}; do not propose full-trip changes.
- The returned proposedDay.day must be {request.day_number}.
- Reduce estimated cost while preserving trip quality, core interests, meals, and rest.
- Prefer free or lower-cost alternatives where reasonable.
- Do not remove every paid/high-value attraction.
- Preserve must-see or high-value items when they appear important.
- Avoid replacing manually priced items when avoidReplacingManualCosts is true.
- Keep route and walking realistic around accommodation when accommodation context is present.
- Respect weather and opening-hours context included in item/place data.
- Explain tradeoffs and approximate savings.
- Use currency {request.budget_context.currency} where possible.
- Use estimatedCost with non-negative amount, 3-letter currency, category,
  confidence, and source "ai".
- Do not claim exact financial accuracy; warnings may mention approximate ticket prices.
- Use change types only: replace_item, remove_item, add_item, modify_item_cost,
  reorder_item, keep_item.
- Do not include fields outside the schema.
- Do not include any text outside the JSON.
""".strip()


def build_repair_itinerary_prompt(request: RepairItineraryRequest) -> str:
    mode = request.constraints.repair_mode
    selected = (
        ", ".join(request.constraints.selected_issue_types)
        if request.constraints.selected_issue_types
        else "none selected"
    )
    special_instructions = request.constraints.special_instructions or "No extra instruction."
    trip_context = request.trip_context.model_dump_json(by_alias=True, exclude_none=True)
    policy = json.dumps(request.policy or {}, ensure_ascii=False)
    policy_evaluation = json.dumps(request.policy_evaluation or {}, ensure_ascii=False)
    approval_risk = json.dumps(request.approval_risk or {}, ensure_ascii=False)
    issues = json.dumps(
        [issue.model_dump(by_alias=True, exclude_none=True) for issue in request.issues],
        ensure_ascii=False,
    )
    constraints = request.constraints.model_dump_json(by_alias=True, exclude_none=True)
    context = (
        request.context.model_dump_json(by_alias=True, exclude_none=True)
        if hasattr(request.context, "model_dump_json")
        else json.dumps(request.context or {}, ensure_ascii=False)
    )
    planning_constraints_section = _planning_constraints_section(request, repair_targets=True)

    return f"""
You are an itinerary repair engine for a web-based travel planning application.

{_output_language_section(request)}

Return ONLY valid JSON. Do not include markdown, explanations, comments, or code fences.
The JSON must exactly match this schema and must not include any top-level fields other
than repairedItinerary, repairSummary, and changes:
{{
  "repairedItinerary": {{
    "destination": "string",
    "summary": "string",
    "travelers": 2,
    "pace": "balanced",
    "currency": "EUR",
    "totalBudget": 700,
    "days": [
      {{
        "day": 1,
        "title": "string",
        "items": [
          {{
            "time": "09:00",
            "type": "activity",
            "name": "string",
            "note": "string",
            "estimatedCost": {{
              "amount": 20,
              "currency": "EUR",
              "category": "activity",
              "confidence": "medium",
              "source": "ai"
            }}
          }}
        ]
      }}
    ]
  }},
  "repairSummary": {{
    "repairMode": "{mode}",
    "changedItemCount": 1,
    "addedItemCount": 0,
    "removedItemCount": 0,
    "movedItemCount": 0,
    "estimatedCostBefore": {{"amount": 920, "currency": "EUR"}},
    "estimatedCostAfter": {{"amount": 690, "currency": "EUR"}},
    "majorChanges": ["string"],
    "issuesAddressed": ["maxTripBudget"],
    "issuesRemaining": ["availability_unchecked"],
    "warnings": ["Availability must be checked again after repair."]
  }},
  "changes": [
    {{
      "type": "item_modified",
      "dayNumber": 1,
      "itemIndex": 0,
      "before": {{"name": "string"}},
      "after": {{"name": "string"}},
      "reason": "string"
    }}
  ]
}}

Role:
- Repair the itinerary for policy/risk review.
- Propose changes only; never claim the repair is approved, booked, paid, or available.

Current itinerary JSON:
{json.dumps(request.itinerary, ensure_ascii=False)}

Trip context:
{trip_context}

Workspace policy:
{policy}

Policy evaluation:
{policy_evaluation}

Approval risk:
{approval_risk}

Selected issues and policy/risk factors:
{issues}

Repair mode:
- mode: {mode}
- selectedIssueTypes: {selected}

Preservation constraints:
{constraints}

Additional context:
{context}

{planning_constraints_section}
Special instructions:
{special_instructions}

Rules:
- Address the selected issues first.
- Minimize changes when minimizeChanges is true.
- Preserve confirmed items when preserveConfirmedItems is true.
- Preserve user-edited items when preserveUserEditedItems is true.
- Do not change accommodation when doNotChangeAccommodation is true.
- Do not change dates, day count, or day numbers when doNotChangeDates is true.
- Keep the destination stable.
- Keep trip comments, collaborators, shares, calendar sync, and approval metadata out of the output.
- Preserve useful item metadata if the item is essentially the same.
- Keep costs as estimates and use source "ai" for AI-estimated costs.
- Do not claim booking, payment, legal compliance, or availability.
- Include warnings for uncertain costs, required availability recheck, partial repairs,
  or major changes.
- Use realistic HH:MM 24-hour times for itinerary item time/endTime fields.
- Return JSON only.
""".strip()


_TEMPLATE_ADAPTATION_OUTPUT_SCHEMA = """{
  "itinerary": {
    "title": "string",
    "destination": "string",
    "startDate": "YYYY-MM-DD",
    "days": [
      {
        "date": "YYYY-MM-DD",
        "title": "string",
        "items": [
          {
            "name": "string",
            "type": "place",
            "description": "string",
            "startTime": "09:00",
            "endTime": "10:30",
            "place": {"name": "string", "category": "string"},
            "estimatedCost": {
              "amount": 18,
              "currency": "EUR",
              "category": "ticket",
              "confidence": "medium",
              "source": "ai"
            },
            "notes": "string"
          }
        ]
      }
    ]
  },
  "adaptationSummary": {
    "sourceDurationDays": 3,
    "targetDurationDays": 3,
    "preservedStructure": true,
    "changedDestination": true,
    "majorChanges": ["string"],
    "warnings": ["string"]
  }
}"""


def build_template_adaptation_prompt(request: TemplateAdaptationRequest) -> str:
    """Build the strict-JSON prompt for adapting a template to a new target."""
    target = request.target
    constraints = request.constraints
    planning_constraints_section = _planning_constraints_section(request)

    return f"""
You are a travel itinerary adaptation engine for a web-based travel planning
application. You take an existing reusable trip template and adapt it to a new
destination and constraints while preserving the template's planning structure
and rhythm.

{_output_language_section(request)}

Return ONLY valid JSON. Do not include markdown, explanations, comments, or code
fences. The JSON must exactly match this schema and must not include any other
top-level fields:
{_TEMPLATE_ADAPTATION_OUTPUT_SCHEMA}

SOURCE TEMPLATE SUMMARY:
{_template_summary_section(request)}

TARGET TRIP REQUIREMENTS:
{_target_requirements_section(request)}

PRESERVATION RULES (keep from the template):
- Preserve the day rhythm and the morning/afternoon/evening structure.
- Preserve the meal and rest structure{_flag(constraints.preserve_meal_structure)}.
- Preserve the number of activities per day proportional to the pace and the
  template's density{_flag(constraints.preserve_activity_density)}.
- Preserve the category mix (sightseeing, food, culture, transport, rest).
- Preserve the overall budget level and the traveler-friendliness of the plan.
- Preserve the template's intent and pacing{_flag(constraints.preserve_structure)}.

ADAPTATION RULES (change for the target):
- Adapt place names, local attractions, and destination-specific context to
  {target.destination}.
- Adapt local transport assumptions to {target.destination}.
- Adapt cost estimates to the target destination and budget{_flag(constraints.adapt_costs)}.
- Respect the target pace, interests, and avoid list.
- Keep realistic time windows in HH:MM 24-hour format and keep items time-ordered.
- Adapt activity order where it makes the day flow better.

DURATION ADAPTATION RULES:
- If the target duration equals the template duration, preserve the day count.
- If the target duration is shorter, compress/trim lower-priority days while
  preserving must-do structure; do not overload the remaining days.
- If the target duration is longer, extend with additional destination-relevant
  days that preserve the original rhythm; do not duplicate identical activities.
- The output MUST contain exactly {target.duration_days} day object(s), dated from
  {target.start_date.isoformat()} onward (one calendar day per day object).

OUTPUT REQUIREMENTS:
- Each item must include name, type, and a helpful note tailored to
  {target.destination}.
- Use only these item types: place, food, activity, transport, rest.
- Include estimatedCost as an object {{amount, currency, category, confidence,
  source "ai"}} where a cost is reasonable, or null for free/uncertain items.
- Prefer the target budget currency for estimatedCost.currency.
- Include a concise adaptationSummary describing the major changes and warnings.

SAFETY AND PRODUCT CONSTRAINTS:
- Do not claim any booking is confirmed and do not guarantee availability.
- Do not invent exact provider prices; mark all costs as estimates.
- Do not assume places are closed or unavailable unless the context says so.
- Avoid unsafe or illegal activities and respect the avoid list.
- Prices are estimates that the user must verify; availability must be checked.

{_workspace_policy_section(request)}
{planning_constraints_section}
{_special_instructions_section(constraints)}
Return ONLY the JSON described above. Do not include any text outside the JSON.
""".strip()


def build_template_adaptation_repair_prompt(
    request: TemplateAdaptationRequest,
    invalid_response_text: str,
    validation_error: str,
) -> str:
    planning_constraints_section = _planning_constraints_section(request)
    return f"""
You previously generated a template adaptation JSON response, but it was invalid.

{_output_language_section(request)}

Validation error:
{validation_error}

Required schema:
{_TEMPLATE_ADAPTATION_OUTPUT_SCHEMA}

SOURCE TEMPLATE SUMMARY:
{_template_summary_section(request)}

TARGET TRIP REQUIREMENTS:
{_target_requirements_section(request)}

{_workspace_policy_section(request)}
{planning_constraints_section}
Invalid previous response:
{invalid_response_text}

Return ONLY corrected JSON. The itinerary MUST contain exactly
{request.target.duration_days} day object(s). Do not include markdown,
explanations, comments, code fences, or fields outside the schema. Preserve the
template's rhythm and adapt places/costs to {request.target.destination}.
Prices remain estimates; do not claim bookings or guaranteed availability.
""".strip()


def _template_summary_section(request: TemplateAdaptationRequest) -> str:
    template = request.template
    # Only sanitized structure is included. Private metadata (source trip IDs,
    # summary, tags) is intentionally excluded from the prompt.
    days_payload = [
        {
            "dayOffset": day.day_offset,
            "title": day.title,
            "items": [
                {
                    key: value
                    for key, value in {
                        "name": item.name,
                        "type": item.type,
                        "startTime": item.start_time or item.time,
                        "endTime": item.end_time,
                        "category": item.place.category if item.place else None,
                        "estimatedCost": (
                            _compact_cost(item.estimated_cost)
                            if item.estimated_cost is not None
                            else None
                        ),
                        "notes": item.notes or item.description,
                    }.items()
                    if value is not None
                }
                for item in day.items
            ],
        }
        for day in template.days
    ]
    lines = [
        f"- Template duration: {template.duration_days} day(s)",
        f"- Template days and items (JSON): {json.dumps(days_payload, ensure_ascii=False)}",
    ]
    return "\n".join(lines)


def _target_requirements_section(request: TemplateAdaptationRequest) -> str:
    target = request.target
    budget = (
        f"{target.budget.amount} {target.budget.currency}"
        if target.budget is not None
        else "not provided"
    )
    interests = ", ".join(target.interests) if target.interests else "general sightseeing"
    avoid = ", ".join(target.avoid) if target.avoid else "none"
    lines = [
        f"- Destination: {target.destination}",
        f"- Start date: {target.start_date.isoformat()}",
        f"- Duration: {target.duration_days} day(s)",
        f"- Budget: {budget}",
        f"- Travelers: {target.travelers}",
        f"- Pace: {target.pace}",
        f"- Interests: {interests}",
        f"- Avoid: {avoid}",
    ]
    context = request.context
    if context is not None and context.destination_context:
        lines.append(
            "- Destination context (JSON): "
            + json.dumps(context.destination_context, ensure_ascii=False)[:1500]
        )
    if context is not None and context.weather_context:
        lines.append(
            "- Weather context (JSON): "
            + json.dumps(context.weather_context, ensure_ascii=False)[:1500]
        )
    return "\n".join(lines)


def _special_instructions_section(constraints: object) -> str:
    special = getattr(constraints, "special_instructions", None)
    if not special:
        return ""
    return f"SPECIAL INSTRUCTIONS:\n- {special}\n\n"


def _flag(enabled: bool) -> str:
    return "" if enabled else " (relaxed for this request)"


def _compact_cost(cost: object) -> dict:
    amount = getattr(cost, "amount", None)
    if amount is not None:
        amount = int(amount) if amount == amount.to_integral_value() else float(amount)
    return {
        key: value
        for key, value in {
            "amount": amount,
            "currency": getattr(cost, "currency", None),
            "category": getattr(cost, "category", None),
        }.items()
        if value is not None
    }


def build_regenerate_day_repair_prompt(
    request: RegenerateDayRequest,
    invalid_response_text: str,
    validation_error: str,
    destination_context: DestinationContext | None = None,
    rag_chunks: list[KnowledgeSearchResult] | None = None,
) -> str:
    planning_constraints_section = _planning_constraints_section(request)
    return f"""
You previously generated a replacement itinerary day JSON response, but it was invalid.

{_output_language_section(request)}

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
{planning_constraints_section}
{_partial_user_context_section(request)}
{_weather_context_section(request.weather_forecast)}
{_accommodation_context_section(request.accommodation)}
{_workspace_policy_section(request)}
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
    planning_constraints_section = _planning_constraints_section(request)
    return f"""
You previously generated a replacement itinerary item JSON response, but it was invalid.

{_output_language_section(request)}

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
{planning_constraints_section}
{_partial_user_context_section(request)}
{_weather_context_section(request.weather_forecast)}
{_accommodation_context_section(request.accommodation)}
{_workspace_policy_section(request)}
{_partial_destination_context_section(destination_context)}
{_rag_context_section(rag_chunks)}

Invalid previous response:
{invalid_response_text}

Return ONLY corrected JSON for item index {request.item_index} in day {request.day_number}.
Do not include markdown, explanations, comments, code fences, or fields outside the schema.
""".strip()


def _planning_constraints_section(request: object, repair_targets: bool = False) -> str:
    constraints = getattr(request, "planning_constraints", None)
    if constraints is None:
        return ""

    lines = [
        "PLANNING CONSTRAINTS:",
        "- Respect these normalized constraints consistently across the response.",
        "- If constraints conflict, prioritize workspace policy and explicit trip/request fields.",
        (
            "- Treat group preferences as soft constraints unless they also appear in explicit "
            "request fields."
        ),
        (
            "- Do not claim group consensus when the group preference summary says decisions are "
            "still open or unclear."
        ),
        "- Keep JSON keys and enum values in English; localize only user-facing text.",
        (
            "- Do not claim live booking, live availability, legal compliance, "
            "medical/accessibility guarantees, or exact prices."
        ),
    ]
    if repair_targets:
        lines.append(
            "- Treat blockers as repair targets rather than a reason to refuse the repair."
        )
    else:
        lines.append("- Treat blockers as hard constraints.")

    language = getattr(constraints, "language", None)
    if language:
        lines.append(f"- Output language: {_LANGUAGE_NAMES.get(language, language)}.")
    budget = getattr(constraints, "budget", None)
    if budget is not None:
        amount = getattr(budget, "amount", None)
        currency = getattr(budget, "currency", None)
        strictness = getattr(budget, "strictness", None)
        if amount is not None and currency:
            lines.append(f"- Budget: {amount} {currency}, strictness: {strictness or 'target'}.")
        elif currency:
            lines.append(f"- Budget currency: {currency}, strictness: {strictness or 'loose'}.")
    pace = getattr(constraints, "pace", None)
    if pace:
        lines.append(f"- Pace: {pace}.")
    walking = getattr(constraints, "walking", None)
    if walking is not None:
        max_km = getattr(walking, "max_km_per_day", None)
        if max_km is not None:
            lines.append(f"- Max walking: {max_km:g} km/day.")
        if getattr(walking, "allow_long_hikes", True) is False:
            lines.append("- Long hikes are not allowed.")
    transport = getattr(constraints, "transport", None)
    if transport is not None:
        _append_optional_line(
            lines,
            "Preferred transport",
            _display_list(getattr(transport, "preferred_modes", [])),
        )
        _append_optional_line(
            lines,
            "Avoid transport",
            _display_list(getattr(transport, "avoid_modes", [])),
        )
        _append_optional_line(
            lines,
            "Disallowed transport",
            _display_list(getattr(transport, "disallowed_modes", [])),
        )
        max_transfer = getattr(transport, "max_transfer_hours_per_day", None)
        if max_transfer is not None:
            lines.append(f"- Max transfer hours per day: {max_transfer}.")
    trip_styles = getattr(constraints, "trip_styles", [])
    _append_optional_line(lines, "Trip styles", _display_list(trip_styles))
    _append_optional_line(lines, "Interests", _display_list(getattr(constraints, "interests", [])))
    _append_optional_line(lines, "Avoid", _display_list(getattr(constraints, "avoid", [])))
    _append_optional_line(lines, "Must have", _display_list(getattr(constraints, "must_have", [])))

    accommodation = getattr(constraints, "accommodation", None)
    if accommodation is not None:
        _append_optional_line(
            lines,
            "Accommodation preferred types",
            _display_list(getattr(accommodation, "preferred_types", [])),
        )
        _append_optional_line(
            lines,
            "Accommodation avoid types",
            _display_list(getattr(accommodation, "avoid_types", [])),
        )
    food = getattr(constraints, "food", None)
    if food is not None:
        _append_optional_line(
            lines,
            "Food preferences",
            _display_list(getattr(food, "preferences", [])),
        )
        _append_optional_line(
            lines,
            "Dietary restrictions",
            _display_list(getattr(food, "dietary_restrictions", [])),
        )

    workspace_policy = getattr(constraints, "workspace_policy", None)
    if workspace_policy is not None and getattr(workspace_policy, "summary", None):
        lines.append("- Workspace policy summary:")
        lines.extend(
            f"  - {line}" for line in str(workspace_policy.summary).splitlines() if line.strip()
        )

    group_preferences = getattr(constraints, "group_preferences", None)
    if group_preferences is not None:
        summary = getattr(group_preferences, "summary", "")
        if summary:
            lines.append("- Group preference summary:")
            lines.extend(f"  - {line}" for line in str(summary).splitlines() if line.strip())
        _append_optional_line(
            lines,
            "Group preferred destinations",
            _display_list(getattr(group_preferences, "preferred_destinations", [])),
        )
        _append_optional_line(
            lines,
            "Group preferred transport",
            _display_list(getattr(group_preferences, "preferred_transport_modes", [])),
        )
        _append_optional_line(
            lines,
            "Group preferred dates",
            _display_list(getattr(group_preferences, "preferred_dates", [])),
        )
        must_have_names = _group_preference_item_names(
            getattr(group_preferences, "must_have_items", []),
        )
        skip_names = _group_preference_item_names(
            getattr(group_preferences, "skip_candidates", []),
        )
        _append_optional_line(lines, "Group must-have activities", _display_list(must_have_names))
        _append_optional_line(lines, "Group skip candidates", _display_list(skip_names))
        open_count = getattr(group_preferences, "open_decision_count", 0)
        if open_count:
            lines.append(
                f"- {open_count} group decision(s) remain open; avoid overstating consensus."
            )
        if must_have_names:
            lines.append("- Preserve group must-have activities where possible.")
        if skip_names:
            lines.append("- Prefer replacing high-skip activities before removing must-have items.")
        lines.append("- Workspace policy overrides group preferences.")

    route = getattr(constraints, "route", None)
    if route:
        route_payload = route if isinstance(route, dict) else {}
        stops = route_payload.get("stops") if isinstance(route_payload, dict) else None
        if isinstance(stops, list) and stops:
            stop_names = [
                str(stop.get("city") or stop.get("destination") or stop.get("name"))
                for stop in stops[:8]
                if isinstance(stop, dict)
            ]
            _append_optional_line(lines, "Route", " -> ".join(name for name in stop_names if name))

    warnings = getattr(constraints, "warnings", []) or []
    blockers = getattr(constraints, "blockers", []) or []
    if warnings:
        lines.append("Warnings:")
        for issue in warnings[:8]:
            message = getattr(issue, "message", "")
            if message:
                lines.append(f"- {message}")
    if blockers:
        lines.append("Blockers:")
        for issue in blockers[:8]:
            message = getattr(issue, "message", "")
            if message:
                lines.append(f"- {message}")

    return "\n" + "\n".join(lines) + "\n"


def _group_preference_item_names(items: list[object]) -> list[str]:
    names: list[str] = []
    for item in items[:6]:
        name = getattr(item, "name", "")
        if isinstance(name, str) and name.strip():
            names.append(name.strip())
    return names


def _accommodation_context_section(accommodation: object | None) -> str:
    if accommodation is None:
        return ""

    lines = [
        "ACCOMMODATION CONTEXT:",
        "Use this optional stay location to make the day routes practical.",
    ]
    _append_optional_line(lines, "Name", getattr(accommodation, "name", None))
    _append_optional_line(lines, "Type", getattr(accommodation, "type", None))
    _append_optional_line(lines, "Address", getattr(accommodation, "address", None))

    place = getattr(accommodation, "place", None)
    if place is not None:
        _append_optional_line(lines, "Place name", getattr(place, "name", None))
        _append_optional_line(lines, "Place address", getattr(place, "address", None))
        latitude = getattr(place, "latitude", None)
        longitude = getattr(place, "longitude", None)
        if latitude is not None and longitude is not None:
            lines.append(f"- Coordinates: {latitude:g}, {longitude:g}")

    check_in = getattr(accommodation, "check_in_date", None)
    if check_in is not None:
        lines.append(f"- Check-in: {check_in.isoformat()}")
    check_out = getattr(accommodation, "check_out_date", None)
    if check_out is not None:
        lines.append(f"- Check-out: {check_out.isoformat()}")

    estimated_cost = getattr(accommodation, "estimated_cost", None)
    if isinstance(estimated_cost, dict):
        amount = estimated_cost.get("amount")
        currency = estimated_cost.get("currency")
        if amount is not None and currency:
            lines.append(f"- Estimated stay cost: {amount} {currency}")

    _append_optional_line(lines, "Notes", getattr(accommodation, "notes", None))
    lines.extend(
        [
            "Accommodation instructions:",
            "- Plan each day to start and end near this accommodation when practical.",
            "- Avoid unnecessary zig-zag routes far from the accommodation.",
            "- For early or late activities, account for travel time from or to the accommodation.",
            "- Do not add accommodation booking suggestions unless the user asks for them.",
        ]
    )
    return "\n" + "\n".join(lines)


def _workspace_policy_section(request: object) -> str:
    constraints = getattr(request, "workspace_policy_constraints", None)
    if constraints is None:
        return ""
    summary = getattr(constraints, "summary", "").strip()
    if not summary:
        return ""
    return (
        "\nWORKSPACE PLANNING POLICY (guidance; backend evaluation is authoritative):\n"
        f"{summary}\n"
        "- Do not claim policy compliance in the output.\n"
        "- Do not omit required JSON fields to satisfy a policy.\n"
    )


def _route_context_section(request: GenerateItineraryRequest) -> str:
    route = getattr(request, "route", None)
    if route is None or not getattr(route, "stops", None):
        return ""

    route_json = route.model_dump(by_alias=True, exclude_none=True, mode="json")
    transport_preferences = getattr(request, "transport_preferences", None)
    trip_styles = getattr(request, "trip_styles", [])
    lines = [
        "\nROUTE CONTEXT:",
        json.dumps(route_json, ensure_ascii=False, indent=2),
        "Route planning instructions:",
        "- Treat this as a multi-destination route when it has more than one stop.",
        "- Use route legs for transfer items and keep costs/durations approximate.",
        (
            "- Respect avoidModes and do not use disallowed modes unless the route explicitly "
            "requires it."
        ),
        (
            "- Do not claim live schedules, ticket purchase, accommodation booking, permits, "
            "or reservations."
        ),
    ]
    if transport_preferences is not None:
        lines.append(
            "Transport preferences: "
            + transport_preferences.model_dump_json(by_alias=True, exclude_none=True)
        )
    if trip_styles:
        lines.append("Trip styles: " + ", ".join(trip_styles))
    return "\n".join(lines) + "\n"


def _items_per_day_for_pace(pace: str) -> int:
    return _ITEMS_PER_DAY_BY_PACE.get(pace, _ITEMS_PER_DAY_BY_PACE["balanced"])


def _partial_trip_section(request: RegenerateDayRequest | OptimizeBudgetDayRequest) -> str:
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


def _partial_user_context_section(
    request: RegenerateDayRequest | OptimizeBudgetDayRequest,
) -> str:
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
