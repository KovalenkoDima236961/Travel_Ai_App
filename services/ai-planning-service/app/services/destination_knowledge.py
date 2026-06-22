import json
import logging
from pathlib import Path
from typing import Protocol

from pydantic import ValidationError

from app.schemas.destination_context import DestinationContext, DestinationContextSummary

logger = logging.getLogger(__name__)


class DestinationKnowledgeProvider(Protocol):
    def get_context(self, destination: str) -> DestinationContext | None: ...

    def list_contexts(self) -> list[DestinationContextSummary]: ...


class FileDestinationKnowledgeProvider:
    def __init__(self, data_dir: Path):
        self._data_dir = data_dir
        self._contexts: list[DestinationContext] | None = None

    def get_context(self, destination: str) -> DestinationContext | None:
        normalized_destination = self._normalize(destination)
        if not normalized_destination:
            return None

        for context in self._load_contexts():
            if self._matches(context, normalized_destination):
                return context

        return None

    def list_contexts(self) -> list[DestinationContextSummary]:
        summaries = [
            DestinationContextSummary(
                destination=context.destination,
                aliases=context.aliases,
                source="file",
            )
            for context in self._load_contexts()
        ]
        return sorted(summaries, key=lambda summary: summary.destination.casefold())

    def _load_contexts(self) -> list[DestinationContext]:
        if self._contexts is not None:
            return self._contexts

        if not self._data_dir.exists() or not self._data_dir.is_dir():
            logger.warning(
                "Destination context directory is missing or invalid",
                extra={"destination_context_dir": str(self._data_dir)},
            )
            self._contexts = []
            return self._contexts

        contexts: list[DestinationContext] = []
        for path in sorted(self._data_dir.glob("*.json")):
            context = self._load_context_file(path)
            if context is not None:
                contexts.append(context)

        self._contexts = contexts
        return self._contexts

    def _load_context_file(self, path: Path) -> DestinationContext | None:
        try:
            with path.open(encoding="utf-8") as file:
                payload = json.load(file)
        except json.JSONDecodeError:
            logger.warning(
                "Skipping invalid destination context JSON",
                extra={"destination_context_file": str(path)},
                exc_info=True,
            )
            return None
        except OSError:
            logger.warning(
                "Could not read destination context file",
                extra={"destination_context_file": str(path)},
                exc_info=True,
            )
            return None

        try:
            return DestinationContext.model_validate(payload)
        except ValidationError:
            logger.warning(
                "Skipping invalid destination context data",
                extra={"destination_context_file": str(path)},
                exc_info=True,
            )
            return None

    def _matches(self, context: DestinationContext, normalized_destination: str) -> bool:
        names = [context.destination, *context.aliases]
        return any(self._normalize(name) == normalized_destination for name in names)

    def _normalize(self, value: str) -> str:
        return value.strip().casefold()
