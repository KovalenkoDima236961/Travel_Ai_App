from __future__ import annotations

import json
from typing import Any

SYSTEM_INSTRUCTION = (
    "Generate schema-valid grounded travel itineraries. Use only supplied "
    "grounding facts, preserve cited grounding IDs, and avoid adding private "
    "or unverifiable personal details."
)


def format_chat_example(example: dict[str, Any]) -> dict[str, list[dict[str, str]]]:
    prompt_payload = {
        "task": example["task"],
        "language": example.get("language", "en"),
        "input": example["input"],
        "grounding": example.get("grounding") or {},
        "labels": example.get("labels") or {},
    }
    return {
        "messages": [
            {"role": "system", "content": SYSTEM_INSTRUCTION},
            {"role": "user", "content": _compact_json(prompt_payload)},
            {"role": "assistant", "content": _compact_json(example["output"])},
        ]
    }


def format_sft_text(example: dict[str, Any]) -> str:
    messages = format_chat_example(example)["messages"]
    parts = [f"<|{message['role']}|>\n{message['content']}" for message in messages]
    return "\n".join(parts) + "\n<|end|>"


def _compact_json(value: Any) -> str:
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))
