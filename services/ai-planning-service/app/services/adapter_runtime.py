from __future__ import annotations

import hashlib
from dataclasses import dataclass
from pathlib import Path

from app.config import Settings
from app.core.paths import resolve_service_path


@dataclass(frozen=True)
class AdapterRuntimeStatus:
    is_ready: bool
    check_status: str
    model_variant: str
    adapter_enabled: bool
    adapter_loaded: bool
    adapter_key: str | None = None
    adapter_checksum: str | None = None
    adapter_checksum_verified: bool = False
    experiment_key: str | None = None
    dataset_version: str | None = None
    fallback_to_base: bool = False

    def safe_metadata(self) -> dict[str, object]:
        metadata: dict[str, object] = {
            "modelVariant": self.model_variant,
            "adapterEnabled": self.adapter_enabled,
            "adapterLoaded": self.adapter_loaded,
            "adapterChecksumVerified": self.adapter_checksum_verified,
            "fallbackToBase": self.fallback_to_base,
        }
        optional_values = {
            "adapterKey": self.adapter_key,
            "adapterChecksum": self.adapter_checksum,
            "experimentKey": self.experiment_key,
            "datasetVersion": self.dataset_version,
        }
        for key, value in optional_values.items():
            if value:
                metadata[key] = value
        return metadata


def adapter_response_metadata(settings: Settings) -> dict[str, object]:
    return validate_adapter_runtime(settings).safe_metadata()


def validate_adapter_runtime(settings: Settings) -> AdapterRuntimeStatus:
    model_variant = settings.ai_model_variant.strip().lower()
    adapter_key = settings.ai_adapter_key.strip() or None
    configured_checksum = settings.ai_adapter_checksum.strip() or None
    experiment_key = settings.ai_adapter_experiment_key.strip() or None
    dataset_version = settings.ai_adapter_dataset_version.strip() or None

    if not settings.ai_adapter_enabled:
        return AdapterRuntimeStatus(
            is_ready=True,
            check_status="disabled",
            model_variant=model_variant,
            adapter_enabled=False,
            adapter_loaded=False,
            fallback_to_base=False,
        )

    adapter_path = resolve_service_path(settings.ai_adapter_path)
    artifact_dir = resolve_service_path(settings.ai_model_artifact_dir)
    status = _validate_adapter_path(adapter_path, artifact_dir)
    if status is not None:
        return _invalid_status(
            settings,
            status,
            model_variant,
            adapter_key,
            configured_checksum,
            experiment_key,
            dataset_version,
        )

    actual_checksum: str | None = None
    if configured_checksum:
        actual_checksum = _checksum_path(adapter_path)
        if actual_checksum.lower() != configured_checksum.lower():
            return _invalid_status(
                settings,
                "checksum_mismatch",
                model_variant,
                adapter_key,
                configured_checksum,
                experiment_key,
                dataset_version,
            )

    return AdapterRuntimeStatus(
        is_ready=True,
        check_status="ok",
        model_variant=model_variant,
        adapter_enabled=True,
        adapter_loaded=True,
        adapter_key=adapter_key,
        adapter_checksum=configured_checksum or actual_checksum,
        adapter_checksum_verified=bool(configured_checksum),
        experiment_key=experiment_key,
        dataset_version=dataset_version,
        fallback_to_base=False,
    )


def _invalid_status(
    settings: Settings,
    check_status: str,
    model_variant: str,
    adapter_key: str | None,
    adapter_checksum: str | None,
    experiment_key: str | None,
    dataset_version: str | None,
) -> AdapterRuntimeStatus:
    can_fallback = settings.ai_adapter_fallback_to_base and not settings.ai_adapter_strict_load
    return AdapterRuntimeStatus(
        is_ready=can_fallback,
        check_status=f"fallback_base:{check_status}" if can_fallback else check_status,
        model_variant=model_variant,
        adapter_enabled=True,
        adapter_loaded=False,
        adapter_key=adapter_key,
        adapter_checksum=adapter_checksum,
        adapter_checksum_verified=False,
        experiment_key=experiment_key,
        dataset_version=dataset_version,
        fallback_to_base=can_fallback,
    )


def _validate_adapter_path(adapter_path: Path, artifact_dir: Path) -> str | None:
    if not adapter_path.exists():
        return "missing"
    if not adapter_path.is_file() and not adapter_path.is_dir():
        return "invalid_path"
    try:
        adapter_path.resolve().relative_to(artifact_dir.resolve())
    except ValueError:
        return "outside_artifact_dir"
    return None


def _checksum_path(path: Path) -> str:
    digest = hashlib.sha256()
    if path.is_file():
        _hash_file(digest, path, path.name)
        return digest.hexdigest()

    for file_path in sorted(item for item in path.rglob("*") if item.is_file()):
        if file_path.name == ".DS_Store":
            continue
        relative_name = file_path.relative_to(path).as_posix()
        _hash_file(digest, file_path, relative_name)
    return digest.hexdigest()


def _hash_file(digest: hashlib._Hash, file_path: Path, relative_name: str) -> None:
    digest.update(relative_name.encode("utf-8"))
    with file_path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
