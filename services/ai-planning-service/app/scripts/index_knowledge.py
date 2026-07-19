import json
import logging
from pathlib import Path
from typing import Any

from app.config import Settings, get_settings
from app.core.paths import resolve_service_path
from app.services.chroma_client import create_persistent_chroma_client
from app.services.knowledge_chunker import chunk_text
from app.services.ollama_embedding_client import OllamaEmbeddingClient

logger = logging.getLogger(__name__)


def main() -> None:
    logging.basicConfig(level=logging.INFO)
    settings = get_settings()
    knowledge_dir = resolve_service_path(settings.rag_knowledge_dir)
    curated_dir = resolve_service_path(settings.knowledge_curated_dir)
    chroma_dir = resolve_service_path(settings.rag_chroma_dir)

    if not knowledge_dir.exists() or not knowledge_dir.is_dir():
        raise SystemExit(f"Knowledge directory does not exist: {knowledge_dir}")

    collection = _get_or_create_collection(
        chroma_dir=chroma_dir,
        collection_name=settings.rag_collection_name,
        settings=settings,
    )
    embedding_client = OllamaEmbeddingClient(settings=settings)

    indexed_files = 0
    indexed_chunks = 0
    for path in _iter_knowledge_files(knowledge_dir):
        destination = path.parent.name.strip().casefold()
        if not destination:
            continue

        text = path.read_text(encoding="utf-8")
        chunks = chunk_text(text)
        if not chunks:
            continue

        ids: list[str] = []
        documents: list[str] = []
        embeddings: list[list[float]] = []
        metadatas: list[dict[str, str | int | float | bool]] = []
        source = path.name

        for chunk_index, content in enumerate(chunks):
            ids.append(f"{destination}:{source}:{chunk_index}")
            documents.append(content)
            embeddings.append(embedding_client.embed(content))
            metadatas.append(
                {
                    "destination": destination,
                    "source": source,
                    "sourcePath": str(path.relative_to(knowledge_dir)),
                    "chunkIndex": chunk_index,
                }
            )

        collection.upsert(
            ids=ids,
            documents=documents,
            embeddings=embeddings,
            metadatas=metadatas,
        )
        indexed_files += 1
        indexed_chunks += len(chunks)

    curated_files, curated_chunks = _index_curated_knowledge(
        collection=collection,
        embedding_client=embedding_client,
        curated_dir=curated_dir,
    )
    indexed_files += curated_files
    indexed_chunks += curated_chunks

    logger.info(
        "Knowledge indexing completed",
        extra={"files": indexed_files, "chunks": indexed_chunks},
    )
    print(
        "Indexed "
        f"{indexed_files} knowledge file(s) and {indexed_chunks} chunk(s) "
        f"into collection {settings.rag_collection_name!r}."
    )


def _get_or_create_collection(
    chroma_dir: Path,
    collection_name: str,
    settings: Settings,
) -> Any:
    chroma_dir.mkdir(parents=True, exist_ok=True)
    try:
        client = create_persistent_chroma_client(settings, chroma_dir)
    except ImportError as exc:
        raise SystemExit("chromadb is required to index local knowledge files") from exc

    return client.get_or_create_collection(
        collection_name,
        embedding_function=None,
    )


def _iter_knowledge_files(knowledge_dir: Path) -> list[Path]:
    supported_suffixes = {".md", ".txt"}
    return sorted(
        path
        for path in knowledge_dir.rglob("*")
        if path.is_file() and path.suffix.lower() in supported_suffixes
    )


def _index_curated_knowledge(
    collection: Any,
    embedding_client: OllamaEmbeddingClient,
    curated_dir: Path,
) -> tuple[int, int]:
    """Index approved structured places and concise documents into the existing collection."""
    if not curated_dir.is_dir():
        logger.info(
            "Curated knowledge directory is unavailable; skipping",
            extra={"path": str(curated_dir)},
        )
        return 0, 0

    indexed_files = indexed_chunks = 0
    for path in sorted((curated_dir / "documents").glob("*.en.md")):
        destination = path.name.removesuffix(".en.md").casefold()
        chunks = chunk_text(path.read_text(encoding="utf-8"))
        if not chunks:
            continue
        _upsert_chunks(
            collection,
            embedding_client,
            ids=[f"curated:document:{destination}:{index}" for index in range(len(chunks))],
            documents=chunks,
            metadatas=[
                {
                    "destination": destination,
                    "source": "manual_curated",
                    "sourcePath": str(path.relative_to(curated_dir)),
                    "recordType": "document",
                    "chunkIndex": index,
                }
                for index in range(len(chunks))
            ],
        )
        indexed_files += 1
        indexed_chunks += len(chunks)

    for path in sorted((curated_dir / "destinations").glob("*.json")):
        try:
            payload = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            logger.warning("Skipping invalid curated destination", extra={"path": str(path)})
            continue
        destination = str(payload.get("canonicalName", "")).strip().casefold()
        places = payload.get("places")
        if not destination or not isinstance(places, list):
            continue
        documents: list[str] = []
        metadatas: list[dict[str, str | int | float | bool]] = []
        for index, place in enumerate(places):
            if not isinstance(place, dict):
                continue
            name = str(place.get("name", "")).strip()
            category = str(place.get("category", "")).strip()
            confidence = place.get("confidence")
            if not name or not category or not isinstance(confidence, int | float):
                continue
            tags = ", ".join(str(tag) for tag in place.get("tags", []) if isinstance(tag, str))
            duration = place.get("typicalDurationMinutes")
            details = [f"{name} is a {category} in {payload.get('canonicalName')}."]
            if tags:
                details.append(f"Tags: {tags}.")
            if isinstance(duration, int):
                details.append(f"Typical duration: {duration} minutes.")
            if place.get("outdoor") is True:
                details.append("Outdoor.")
            if place.get("rainFriendly") is True:
                details.append("Rain-friendly.")
            documents.append(" ".join(details))
            metadatas.append(
                {
                    "destination": destination,
                    "source": str(place.get("sourceKey", "manual_curated")),
                    "sourcePath": str(path.relative_to(curated_dir)),
                    "recordType": "place",
                    "placeName": name,
                    "category": category,
                    "confidence": float(confidence),
                    "chunkIndex": index,
                }
            )
        if documents:
            _upsert_chunks(
                collection,
                embedding_client,
                ids=[f"curated:place:{destination}:{index}" for index in range(len(documents))],
                documents=documents,
                metadatas=metadatas,
            )
            indexed_files += 1
            indexed_chunks += len(documents)
    return indexed_files, indexed_chunks


def _upsert_chunks(
    collection: Any,
    embedding_client: OllamaEmbeddingClient,
    ids: list[str],
    documents: list[str],
    metadatas: list[dict[str, str | int | float | bool]],
) -> None:
    collection.upsert(
        ids=ids,
        documents=documents,
        embeddings=[embedding_client.embed(document) for document in documents],
        metadatas=metadatas,
    )


if __name__ == "__main__":
    main()
