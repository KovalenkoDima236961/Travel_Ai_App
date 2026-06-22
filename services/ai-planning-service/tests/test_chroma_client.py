import os
import sys
from types import ModuleType
from typing import Any

from app.config import Settings
from app.services.chroma_client import create_persistent_chroma_client


class FakeChromaSettings:
    def __init__(self, **kwargs: Any) -> None:
        self.kwargs = kwargs


def _install_fake_chromadb(monkeypatch) -> dict[str, Any]:
    calls: dict[str, Any] = {}
    chromadb = ModuleType("chromadb")
    config = ModuleType("chromadb.config")

    def persistent_client(**kwargs: Any) -> object:
        calls.update(kwargs)
        return object()

    chromadb.PersistentClient = persistent_client  # type: ignore[attr-defined]
    config.Settings = FakeChromaSettings  # type: ignore[attr-defined]
    monkeypatch.setitem(sys.modules, "chromadb", chromadb)
    monkeypatch.setitem(sys.modules, "chromadb.config", config)
    return calls


def test_create_persistent_chroma_client_disables_telemetry_by_default(
    monkeypatch,
    tmp_path,
) -> None:
    calls = _install_fake_chromadb(monkeypatch)
    monkeypatch.delenv("ANONYMIZED_TELEMETRY", raising=False)

    create_persistent_chroma_client(Settings(), tmp_path)

    assert os.environ["ANONYMIZED_TELEMETRY"] == "false"
    assert calls["path"] == str(tmp_path)
    assert calls["settings"].kwargs == {"anonymized_telemetry": False}


def test_create_persistent_chroma_client_can_enable_telemetry(
    monkeypatch,
    tmp_path,
) -> None:
    calls = _install_fake_chromadb(monkeypatch)
    monkeypatch.delenv("ANONYMIZED_TELEMETRY", raising=False)

    create_persistent_chroma_client(
        Settings(chroma_anonymized_telemetry=True),
        tmp_path,
    )

    assert os.environ["ANONYMIZED_TELEMETRY"] == "true"
    assert calls["settings"].kwargs == {"anonymized_telemetry": True}
