from __future__ import annotations

import json
from pathlib import Path
from typing import Literal

from pydantic import BaseModel, Field, field_validator, model_validator

TaskType = Literal["grounded_itinerary_generation"]
TrainingMethod = Literal["lora", "qlora"]


class PromotionGates(BaseModel):
    min_schema_valid_rate: float = Field(default=0.99, ge=0, le=1)
    min_factual_precision: float = Field(default=0.92, ge=0, le=1)
    min_grounding_citation_rate: float = Field(default=0.95, ge=0, le=1)
    min_no_pii_rate: float = Field(default=1.0, ge=0, le=1)
    max_cost_regression_pct: float = Field(default=10, ge=0)
    max_latency_regression_pct: float = Field(default=25, ge=0)


class TrainingConfig(BaseModel):
    experiment_key: str = Field(alias="experimentKey", min_length=3)
    task_type: TaskType = Field(default="grounded_itinerary_generation", alias="taskType")
    method: TrainingMethod = "qlora"
    base_model_name: str = Field(alias="baseModelName", min_length=1)
    base_model_revision: str = Field(default="main", alias="baseModelRevision")
    base_model_family: str = Field(default="llama", alias="baseModelFamily")
    base_model_license: str = Field(default="record-before-training", alias="baseModelLicense")
    dataset_path: str = Field(alias="datasetPath", min_length=1)
    dataset_version: str = Field(alias="datasetVersion", min_length=1)
    output_dir: str = Field(default="artifacts/ai-training", alias="outputDir")
    seed: int = Field(default=42, ge=0)
    max_seq_length: int = Field(default=4096, alias="maxSeqLength", ge=128)
    train_batch_size: int = Field(default=1, alias="trainBatchSize", ge=1)
    eval_batch_size: int = Field(default=1, alias="evalBatchSize", ge=1)
    gradient_accumulation_steps: int = Field(default=16, alias="gradientAccumulationSteps", ge=1)
    num_train_epochs: float = Field(default=1.0, alias="numTrainEpochs", gt=0)
    learning_rate: float = Field(default=2e-4, alias="learningRate", gt=0)
    warmup_ratio: float = Field(default=0.03, alias="warmupRatio", ge=0, le=1)
    weight_decay: float = Field(default=0.0, alias="weightDecay", ge=0)
    lora_r: int = Field(default=16, alias="loraR", ge=1)
    lora_alpha: int = Field(default=32, alias="loraAlpha", ge=1)
    lora_dropout: float = Field(default=0.05, alias="loraDropout", ge=0, le=1)
    target_modules: list[str] = Field(
        default_factory=lambda: [
            "q_proj",
            "k_proj",
            "v_proj",
            "o_proj",
            "gate_proj",
            "up_proj",
            "down_proj",
        ],
        alias="targetModules",
    )
    quantization_bits: int = Field(default=4, alias="quantizationBits")
    gradient_checkpointing: bool = Field(default=True, alias="gradientCheckpointing")
    allow_holdout_training: bool = Field(default=False, alias="allowHoldoutTraining")
    dry_run: bool = Field(default=False, alias="dryRun")
    cpu_smoke: bool = Field(default=False, alias="cpuSmoke")
    promotion_gates: PromotionGates = Field(default_factory=PromotionGates, alias="promotionGates")

    @field_validator("experiment_key")
    @classmethod
    def normalize_experiment_key(cls, value: str) -> str:
        normalized = value.strip()
        if not normalized:
            raise ValueError("experimentKey is required")
        if any(ch for ch in normalized if not (ch.isalnum() or ch in {"-", "_", "."})):
            raise ValueError("experimentKey may contain only letters, numbers, '-', '_', '.'")
        return normalized

    @field_validator("method")
    @classmethod
    def normalize_method(cls, value: str) -> str:
        return value.strip().lower()

    @field_validator("target_modules")
    @classmethod
    def require_target_modules(cls, value: list[str]) -> list[str]:
        cleaned = [item.strip() for item in value if item.strip()]
        if not cleaned:
            raise ValueError("targetModules must not be empty")
        return cleaned

    @model_validator(mode="after")
    def validate_method(self) -> TrainingConfig:
        if self.method == "qlora" and self.quantization_bits not in {4, 8}:
            raise ValueError("QLoRA quantizationBits must be 4 or 8")
        if self.allow_holdout_training:
            raise ValueError("allowHoldoutTraining must remain false")
        return self


def load_config(path: str | Path) -> TrainingConfig:
    config_path = Path(path)
    payload = json.loads(config_path.read_text())
    return TrainingConfig.model_validate(payload)


def write_default_config(path: str | Path) -> None:
    config = TrainingConfig(
        experimentKey="grounded-itinerary-local-v1",
        baseModelName="meta-llama/Meta-Llama-3.1-8B-Instruct",
        datasetPath="../../data/ai-datasets/example-export",
        datasetVersion="v1",
        dryRun=True,
    )
    Path(path).write_text(config.model_dump_json(by_alias=True, indent=2) + "\n")
