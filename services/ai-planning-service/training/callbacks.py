from __future__ import annotations

import json
from pathlib import Path
from typing import Any


class JsonlMetricsCallback:
    """Minimal callback compatible with Transformers Trainer log events."""

    def __init__(self, output_path: str | Path) -> None:
        self.output_path = Path(output_path)
        self.output_path.parent.mkdir(parents=True, exist_ok=True)

    def on_log(
        self,
        _args: Any,
        _state: Any,
        _control: Any,
        logs: dict[str, Any] | None = None,
        **_: Any,
    ) -> None:
        if not logs:
            return
        with self.output_path.open("a") as handle:
            handle.write(json.dumps(logs, sort_keys=True) + "\n")
