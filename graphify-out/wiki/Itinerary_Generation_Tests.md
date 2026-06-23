# Itinerary Generation Tests

> 26 nodes · cohesion 0.10

## Key Concepts

- **test_generate_itinerary.py** (25 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **partial_payload()** (7 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_regenerate_day_instruction_too_long_returns_400()** (2 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_regenerate_day_invalid_day_number_returns_400()** (2 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_regenerate_day_success_returns_replacement_day_only()** (2 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_regenerate_item_accepts_optional_user_context()** (2 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_regenerate_item_invalid_item_index_returns_400()** (2 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_regenerate_item_success_returns_replacement_item_only()** (2 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_days_greater_than_thirty_returns_validation_error()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_days_less_than_one_returns_validation_error()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_empty_budget_currency_defaults_to_eur()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_empty_destination_returns_validation_error()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_generate_itinerary_accepts_optional_user_context()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_generate_itinerary_success()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_generated_itinerary_has_requested_number_of_days()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_generated_itinerary_includes_destination_in_title_or_note()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_health_endpoint_returns_ok()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_invalid_trip_id_returns_validation_error()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_missing_destination_returns_validation_error()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_mock_generator_uses_hidden_gems_and_local_food_preferences()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_negative_budget_amount_returns_validation_error()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_ready_endpoint_returns_ready_in_mock_mode()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_relaxed_pace_produces_fewer_or_equal_items_than_intensive_pace()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_travelers_less_than_one_returns_validation_error()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- **test_unexpected_generator_error_returns_clean_generation_error()** (1 connections) — `services/ai-planning-service/tests/test_generate_itinerary.py`
- *... and 1 more nodes in this community*

## Relationships

- No strong cross-community connections detected

## Source Files

- `services/ai-planning-service/tests/test_generate_itinerary.py`

## Audit Trail

- EXTRACTED: 62 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*