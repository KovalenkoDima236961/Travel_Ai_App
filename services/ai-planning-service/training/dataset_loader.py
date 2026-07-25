from __future__ import annotations

import hashlib
import json
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from training.config import TrainingConfig

REQUIRED_EXPORT_FILES = {
    "train.jsonl",
    "validation.jsonl",
    "test.jsonl",
    "holdout.jsonl",
    "manifest.json",
    "checksums.txt",
    "README.md",
}
PRIVATE_TEXT_PATTERN = re.compile(
    r"(\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b|"
    r"\b(?:\+?\d{1,3}[\s.-]?)?(?:\(?\d{3}\)?[\s.-]?)\d{3}[\s.-]?\d{4}\b|"
    r"password|api[_-]?key|access[_-]?token|refresh[_-]?token|"
    r"system[_-]?prompt|chain[_-]?of[_-]?thought|receiptocr)",
    re.IGNORECASE,
)
PRIVATE_KEYS = {
    "userid",
    "user_id",
    "tripid",
    "trip_id",
    "email",
    "phone",
    "password",
    "apikey",
    "api_key",
    "accesstoken",
    "access_token",
    "refreshtoken",
    "refresh_token",
    "systemprompt",
    "system_prompt",
    "chainofthought",
    "chain_of_thought",
    "receiptocr",
    "receipt_ocr",
}


@dataclass(frozen=True)
class TrainingDatasetBundle:
    root: Path
    manifest: dict[str, Any]
    train_examples: list[dict[str, Any]]
    validation_examples: list[dict[str, Any]]
    test_examples: list[dict[str, Any]]
    export_checksum: str
    distribution: dict[str, Any]


def load_training_dataset(config: TrainingConfig) -> TrainingDatasetBundle:
    root = Path(config.dataset_path).expanduser()
    if not root.is_absolute():
        root = Path.cwd() / root
    root = root.resolve()
    _validate_export_shape(root)
    export_checksum = verify_checksums(root)
    manifest = json.loads((root / "manifest.json").read_text())
    _validate_manifest(manifest, config)

    train_examples = _load_split(root, "train", config)
    validation_examples = _load_split(root, "validation", config)
    test_examples = _load_split(root, "test", config)
    holdout_examples = _load_split(root, "holdout", config, allow_empty=True)
    _assert_disjoint_splits(
        {
            "train": train_examples,
            "validation": validation_examples,
            "test": test_examples,
            "holdout": holdout_examples,
        }
    )
    distribution = _distribution(train_examples, validation_examples, test_examples, manifest)
    return TrainingDatasetBundle(
        root=root,
        manifest=manifest,
        train_examples=train_examples,
        validation_examples=validation_examples,
        test_examples=test_examples,
        export_checksum=export_checksum,
        distribution=distribution,
    )


def verify_checksums(root: Path) -> str:
    expected: dict[str, str] = {}
    for line in (root / "checksums.txt").read_text().splitlines():
        if not line.strip():
            continue
        checksum, name = line.split(None, 1)
        expected[name.strip()] = checksum.strip()

    errors: list[str] = []
    digest = hashlib.sha256()
    for name in sorted(expected):
        path = root / name
        if not path.exists():
            errors.append(f"checksum references missing file {name}")
            continue
        actual = _sha256_file(path)
        if actual != expected[name]:
            errors.append(f"{name} checksum mismatch")
        digest.update(f"{name} {actual}\n".encode())
    if errors:
        raise ValueError("; ".join(errors))
    return digest.hexdigest()


def _validate_export_shape(root: Path) -> None:
    if not root.is_dir():
        raise ValueError(f"dataset export does not exist: {root}")
    missing = sorted(name for name in REQUIRED_EXPORT_FILES if not (root / name).exists())
    if missing:
        raise ValueError(f"dataset export missing files: {', '.join(missing)}")


def _validate_manifest(manifest: dict[str, Any], config: TrainingConfig) -> None:
    if manifest.get("schemaVersion") != "ai_dataset_v1":
        raise ValueError("manifest schemaVersion must be ai_dataset_v1")
    split_counts = manifest.get("splitCounts")
    if not isinstance(split_counts, dict):
        raise ValueError("manifest splitCounts is required")
    if int(split_counts.get("train", 0)) <= 0:
        raise ValueError("training split must contain at least one example")
    if int(split_counts.get("validation", 0)) <= 0:
        raise ValueError("validation split must contain at least one example")
    task_distribution = manifest.get("taskDistribution") or {}
    unexpected_tasks = set(task_distribution) - {config.task_type}
    if unexpected_tasks:
        raise ValueError(f"dataset contains unsupported tasks: {sorted(unexpected_tasks)}")


def _load_split(
    root: Path,
    split: str,
    config: TrainingConfig,
    *,
    allow_empty: bool = False,
) -> list[dict[str, Any]]:
    examples: list[dict[str, Any]] = []
    for line_number, line in enumerate((root / f"{split}.jsonl").read_text().splitlines(), 1):
        if not line.strip():
            continue
        payload = json.loads(line)
        _validate_example(payload, split, line_number, config)
        examples.append(payload)
    if not examples and not allow_empty and split in {"train", "validation"}:
        raise ValueError(f"{split} split is empty")
    return examples


def _validate_example(
    payload: dict[str, Any],
    split: str,
    line_number: int,
    config: TrainingConfig,
) -> None:
    required = {"id", "task", "language", "schemaVersion", "input", "grounding", "output", "labels"}
    missing = required - payload.keys()
    if missing:
        raise ValueError(f"{split}.jsonl:{line_number}: missing {sorted(missing)}")
    if payload.get("task") != config.task_type:
        raise ValueError(f"{split}.jsonl:{line_number}: task must be {config.task_type}")
    _scan_private_data(payload, f"{split}.jsonl:{line_number}")


def _scan_private_data(value: Any, path: str) -> None:
    if isinstance(value, dict):
        for key, item in value.items():
            normalized_key = re.sub(r"[^a-z0-9_]", "", str(key).lower())
            if normalized_key in PRIVATE_KEYS:
                raise ValueError(f"{path}: private identifier key {key!r} is not allowed")
            _scan_private_data(item, f"{path}.{key}")
    elif isinstance(value, list):
        for index, item in enumerate(value):
            _scan_private_data(item, f"{path}[{index}]")
    elif isinstance(value, str) and PRIVATE_TEXT_PATTERN.search(value):
        raise ValueError(f"{path}: private text marker detected")


def _assert_disjoint_splits(splits: dict[str, list[dict[str, Any]]]) -> None:
    owner_by_id: dict[str, str] = {}
    for split, examples in splits.items():
        for example in examples:
            example_id = str(example.get("id"))
            if example_id in owner_by_id:
                raise ValueError(
                    f"example {example_id} appears in both {owner_by_id[example_id]} and {split}"
                )
            owner_by_id[example_id] = split


def _distribution(
    train_examples: list[dict[str, Any]],
    validation_examples: list[dict[str, Any]],
    test_examples: list[dict[str, Any]],
    manifest: dict[str, Any],
) -> dict[str, Any]:
    return {
        "trainCount": len(train_examples),
        "validationCount": len(validation_examples),
        "testCount": len(test_examples),
        "taskDistribution": manifest.get("taskDistribution", {}),
        "languageDistribution": manifest.get("languageDistribution", {}),
        "sourceDistribution": manifest.get("sourceDistribution", {}),
        "qualityScore": manifest.get("qualityScore", {}),
    }


def _sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()
