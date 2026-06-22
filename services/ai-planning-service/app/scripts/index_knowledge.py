import logging
from pathlib import Path
from typing import Any

from app.config import Settings, get_settings
from app.services.chroma_client import create_persistent_chroma_client
from app.services.knowledge_chunker import chunk_text
from app.services.ollama_embedding_client import OllamaEmbeddingClient

logger = logging.getLogger(__name__)


def main() -> None:
    logging.basicConfig(level=logging.INFO)
    settings = get_settings()
    knowledge_dir = _resolve_service_path(settings.rag_knowledge_dir)
    chroma_dir = _resolve_service_path(settings.rag_chroma_dir)

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


def _resolve_service_path(raw_path: str) -> Path:
    path = Path(raw_path)
    if path.is_absolute():
        return path

    cwd_path = Path.cwd() / path
    if cwd_path.exists():
        return cwd_path

    service_root = Path(__file__).resolve().parents[2]
    return service_root / path


if __name__ == "__main__":
    main()
