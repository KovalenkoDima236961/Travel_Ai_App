import os
from pathlib import Path
from typing import Any

from app.config import Settings


def create_persistent_chroma_client(settings: Settings, chroma_dir: Path) -> Any:
    os.environ["ANONYMIZED_TELEMETRY"] = "true" if settings.chroma_anonymized_telemetry else "false"

    import chromadb
    from chromadb.config import Settings as ChromaSettings

    chroma_settings = ChromaSettings(anonymized_telemetry=settings.chroma_anonymized_telemetry)
    return chromadb.PersistentClient(
        path=str(chroma_dir),
        settings=chroma_settings,
    )
