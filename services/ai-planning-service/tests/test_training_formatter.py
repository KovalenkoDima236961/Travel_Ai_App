from training.formatter import format_chat_example, format_sft_text


def test_format_chat_example_uses_grounding_without_raw_prompts() -> None:
    formatted = format_chat_example(
        {
            "task": "grounded_itinerary_generation",
            "language": "en",
            "input": {"destination": "Rome"},
            "grounding": {"places": [{"id": "curated:rome:0", "name": "Colosseum"}]},
            "labels": {},
            "output": {"days": [{"day": 1, "items": [{"name": "Colosseum"}]}]},
        }
    )

    assert [message["role"] for message in formatted["messages"]] == [
        "system",
        "user",
        "assistant",
    ]
    assert "system_prompt" not in format_sft_text(
        {
            "task": "grounded_itinerary_generation",
            "language": "en",
            "input": {"destination": "Rome"},
            "grounding": {},
            "labels": {},
            "output": {"days": []},
        }
    )
