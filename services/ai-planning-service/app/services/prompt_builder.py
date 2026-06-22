from app.schemas.itinerary import GenerateItineraryRequest

_ITEMS_PER_DAY_BY_PACE = {
    "relaxed": 3,
    "balanced": 4,
    "intensive": 5,
}


def build_itinerary_prompt(request: GenerateItineraryRequest) -> str:
    items_per_day = _ITEMS_PER_DAY_BY_PACE[request.pace]
    interests = ", ".join(request.interests) if request.interests else "general sightseeing"
    budget = (
        f"{request.budget_amount} {request.budget_currency}"
        if request.budget_amount is not None
        else "not provided"
    )
    start_date = request.start_date.isoformat() if request.start_date else "not provided"

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

Rules:
- Generate exactly {request.days} day objects.
- Each day must have exactly {items_per_day} items.
- Use realistic times in HH:MM 24-hour format.
- Use only these item types: place, food, activity, transport, rest.
- Include practical notes tailored to {request.destination}; avoid generic filler.
- Include estimatedCost as a number or null.
- Avoid hallucinated exact prices when uncertain; use reasonable estimates.
- Do not include fields outside the schema.
- Do not include any text outside the JSON.
""".strip()
