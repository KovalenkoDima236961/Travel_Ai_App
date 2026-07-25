from __future__ import annotations

import hashlib
import json
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from training.config import TrainingConfig
from training.dataset_loader import TrainingDatasetBundle


def experiment_dir(config: TrainingConfig) -> Path:
    root = Path(config.output_dir).expanduser()
    if not root.is_absolute():
        root = Path.cwd() / root
    path = (root / config.experiment_key).resolve()
    path.mkdir(parents=True, exist_ok=True)
    return path


def write_experiment_manifest(
    config: TrainingConfig,
    bundle: TrainingDatasetBundle,
    *,
    status: str,
    extra: dict[str, Any] | None = None,
) -> Path:
    target_dir = experiment_dir(config)
    manifest = {
        "experimentKey": config.experiment_key,
        "status": status,
        "taskType": config.task_type,
        "method": config.method,
        "baseModelName": config.base_model_name,
        "baseModelRevision": config.base_model_revision,
        "datasetVersion": config.dataset_version,
        "datasetExportChecksum": bundle.export_checksum,
        "distribution": bundle.distribution,
        "createdAt": datetime.now(UTC).isoformat(),
        "config": config.model_dump(by_alias=True, exclude={"dataset_path"}),
    }
    if extra:
        manifest.update(extra)
    path = target_dir / "experiment-manifest.json"
    path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n")
    return path


def sha256_path(path: Path) -> str:
    digest = hashlib.sha256()
    if path.is_file():
        _hash_file(digest, path, path.name)
        return digest.hexdigest()
    for file_path in sorted(item for item in path.rglob("*") if item.is_file()):
        _hash_file(digest, file_path, file_path.relative_to(path).as_posix())
    return digest.hexdigest()


def _hash_file(digest: hashlib._Hash, file_path: Path, relative_name: str) -> None:
    digest.update(relative_name.encode("utf-8"))
    with file_path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
