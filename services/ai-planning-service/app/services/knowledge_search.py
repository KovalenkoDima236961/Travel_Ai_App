import logging
from pathlib import Path
from typing import Any, Protocol

from app.config import Settings
from app.schemas.knowledge import KnowledgeSearchResult
from app.services.chroma_client import create_persistent_chroma_client
from app.services.ollama_embedding_client import OllamaEmbeddingClient

logger = logging.getLogger(__name__)


class EmbeddingClient(Protocol):
    def embed(self, text: str) -> list[float]: ...


class KnowledgeSearchService:
    def __init__(
        self,
        settings: Settings,
        embedding_client: EmbeddingClient | None = None,
        collection: Any | None = None,
    ) -> None:
        self._settings = settings
        self._embedding_client = embedding_client or OllamaEmbeddingClient(settings)
        self._collection = collection
        self._collection_lookup_failed = False
        self.last_search_text: str | None = None
        self.last_search_failed = False

    def search(
        self,
        destination: str,
        interests: list[str],
        query: str | None,
        top_k: int,
    ) -> list[KnowledgeSearchResult]:
        if not self._settings.rag_enabled:
            return []

        normalized_destination = _normalize_destination(destination)
        if not normalized_destination:
            return []

        effective_top_k = min(max(top_k, 1), 10)
        search_text = self._build_search_text(destination, interests, query)
        self.last_search_text = search_text
        self.last_search_failed = False

        try:
            embedding = self._embedding_client.embed(search_text)
        except Exception:
            self.last_search_failed = True
            logger.warning(
                "RAG search embedding failed; continuing without RAG context",
                extra={"destination": destination},
                exc_info=True,
            )
            return []

        collection = self._get_collection()
        if collection is None:
            self.last_search_failed = True
            return []

        try:
            raw_results = collection.query(
                query_embeddings=[embedding],
                n_results=effective_top_k,
                where={"destination": normalized_destination},
                include=["documents", "metadatas", "distances"],
            )
        except Exception:
            self.last_search_failed = True
            logger.warning(
                "RAG ChromaDB query failed; continuing without RAG context",
                extra={"destination": destination},
                exc_info=True,
            )
            return []

        return self._map_results(raw_results, normalized_destination, effective_top_k)

    def _build_search_text(
        self,
        destination: str,
        interests: list[str],
        query: str | None,
    ) -> str:
        parts = [destination.strip(), "travel itinerary"]
        normalized_interests = [interest.strip() for interest in interests if interest.strip()]
        if normalized_interests:
            parts.append("interests: " + ", ".join(normalized_interests))
        if query and query.strip():
            parts.append(query.strip())
        return " | ".join(parts)

    def _get_collection(self) -> Any | None:
        if self._collection is not None:
            return self._collection
        if self._collection_lookup_failed:
            return None

        try:
            client = create_persistent_chroma_client(
                self._settings,
                _resolve_service_path(self._settings.rag_chroma_dir),
            )
            self._collection = client.get_collection(
                self._settings.rag_collection_name,
                embedding_function=None,
            )
        except ImportError:
            logger.warning(
                "ChromaDB is not installed; continuing without RAG context",
                extra={"rag_collection_name": self._settings.rag_collection_name},
            )
            self._collection_lookup_failed = True
            return None
        except Exception:
            logger.warning(
                "RAG ChromaDB collection is missing; continuing without RAG context",
                extra={"rag_collection_name": self._settings.rag_collection_name},
                exc_info=True,
            )
            self._collection_lookup_failed = True
            return None

        return self._collection

    def _map_results(
        self,
        raw_results: Any,
        normalized_destination: str,
        top_k: int,
    ) -> list[KnowledgeSearchResult]:
        ids = _first_result_list(raw_results, "ids")
        documents = _first_result_list(raw_results, "documents")
        metadatas = _first_result_list(raw_results, "metadatas")
        distances = _first_result_list(raw_results, "distances")

        items: list[KnowledgeSearchResult] = []
        for index, item_id in enumerate(ids):
            content = documents[index] if index < len(documents) else ""
            metadata = _clean_metadata(metadatas[index] if index < len(metadatas) else {})
            distance = distances[index] if index < len(distances) else None

            if not isinstance(item_id, str) or not isinstance(content, str) or not content.strip():
                continue

            item_destination = str(metadata.get("destination", normalized_destination)).strip()
            if _normalize_destination(item_destination) != normalized_destination:
                continue

            score = _distance_to_similarity(distance)
            if score is not None and score < self._settings.rag_min_score:
                continue
            if score is None and self._settings.rag_min_score > 0:
                continue

            source = metadata.get("source", "unknown")
            items.append(
                KnowledgeSearchResult(
                    id=item_id,
                    destination=normalized_destination,
                    source=str(source),
                    content=content.strip(),
                    score=score,
                    metadata=metadata,
                )
            )
            if len(items) >= top_k:
                break

        return items


def _first_result_list(raw_results: Any, key: str) -> list[Any]:
    if not isinstance(raw_results, dict):
        return []
    value = raw_results.get(key)
    if not isinstance(value, list) or not value:
        return []
    first = value[0]
    return first if isinstance(first, list) else []


def _distance_to_similarity(distance: Any) -> float | None:
    if not isinstance(distance, int | float):
        return None
    return 1 / (1 + float(distance))


def _clean_metadata(metadata: Any) -> dict[str, str | int | float | bool]:
    if not isinstance(metadata, dict):
        return {}

    cleaned: dict[str, str | int | float | bool] = {}
    for key, value in metadata.items():
        if not isinstance(key, str):
            continue
        if isinstance(value, str | int | float | bool):
            cleaned[key] = value
        elif value is not None:
            cleaned[key] = str(value)
    return cleaned


def _normalize_destination(destination: str) -> str:
    return destination.strip().casefold()


def _resolve_service_path(raw_path: str) -> Path:
    path = Path(raw_path)
    if path.is_absolute():
        return path

    cwd_path = Path.cwd() / path
    if cwd_path.exists():
        return cwd_path

    service_root = Path(__file__).resolve().parents[2]
    return service_root / path
