from __future__ import annotations

import hashlib
import json
from pathlib import Path

import pytest

from training.config import TrainingConfig
from training.dataset_loader import load_training_dataset


def test_load_training_dataset_excludes_holdout(tmp_path: Path) -> None:
    root = write_export(tmp_path)
    config = config_for(root)

    bundle = load_training_dataset(config)

    assert len(bundle.train_examples) == 1
    assert len(bundle.validation_examples) == 1
    assert len(bundle.test_examples) == 1
    assert bundle.distribution["trainCount"] == 1


def test_load_training_dataset_rejects_private_identifiers(tmp_path: Path) -> None:
    root = write_export(tmp_path, train_input={"destination": "Rome", "userId": "raw-user"})
    config = config_for(root)

    with pytest.raises(ValueError, match="private identifier"):
        load_training_dataset(config)


def test_load_training_dataset_detects_holdout_leak(tmp_path: Path) -> None:
    root = write_export(tmp_path, holdout_id="example-train")
    config = config_for(root)

    with pytest.raises(ValueError, match="appears in both train and holdout"):
        load_training_dataset(config)


def config_for(root: Path) -> TrainingConfig:
    return TrainingConfig(
        experimentKey="exp-v1",
        taskType="grounded_itinerary_generation",
        method="lora",
        baseModelName="local/test-model",
        datasetPath=str(root),
        datasetVersion="v1",
        outputDir=str(root / "artifacts"),
    )


def write_export(
    tmp_path: Path,
    *,
    train_input: dict[str, object] | None = None,
    holdout_id: str = "example-holdout",
) -> Path:
    root = tmp_path / "export"
    root.mkdir()
    train = example("example-train", train_input or {"destination": "Rome"})
    validation = example("example-validation", {"destination": "Vienna"})
    test = example("example-test", {"destination": "Paris"})
    holdout = example(holdout_id, {"destination": "Lisbon"})
    _write_jsonl(root / "train.jsonl", [train])
    _write_jsonl(root / "validation.jsonl", [validation])
    _write_jsonl(root / "test.jsonl", [test])
    _write_jsonl(root / "holdout.jsonl", [holdout])
    manifest = {
        "schemaVersion": "ai_dataset_v1",
        "version": "v1",
        "exampleCount": 4,
        "splitCounts": {"train": 1, "validation": 1, "test": 1, "holdout": 1},
        "taskDistribution": {"grounded_itinerary_generation": 4},
        "languageDistribution": {"en": 4},
        "sourceDistribution": {"synthetic": 4},
        "qualityScore": {"average": 0.95, "min": 0.9, "max": 1.0},
    }
    (root / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    (root / "README.md").write_text("sanitized export\n")
    checksums = {}
    for name in [
        "README.md",
        "holdout.jsonl",
        "manifest.json",
        "test.jsonl",
        "train.jsonl",
        "validation.jsonl",
    ]:
        checksums[name] = hashlib.sha256((root / name).read_bytes()).hexdigest()
    (root / "checksums.txt").write_text(
        "".join(f"{checksum}  {name}\n" for name, checksum in sorted(checksums.items()))
    )
    return root


def example(example_id: str, input_payload: dict[str, object]) -> dict[str, object]:
    return {
        "id": example_id,
        "task": "grounded_itinerary_generation",
        "language": "en",
        "schemaVersion": "ai_dataset_v1",
        "input": input_payload,
        "grounding": {"places": [{"id": "curated:rome:0", "name": "Colosseum"}]},
        "output": {"days": [{"day": 1, "items": [{"name": "Colosseum"}]}]},
        "labels": {"quality": "approved"},
        "metadata": {"sourceType": "synthetic", "qualityScore": 0.95},
    }


def _write_jsonl(path: Path, rows: list[dict[str, object]]) -> None:
    path.write_text("".join(json.dumps(row, sort_keys=True) + "\n" for row in rows))
