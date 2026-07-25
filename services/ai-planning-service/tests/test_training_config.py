import pytest

from training.config import TrainingConfig


def test_training_config_rejects_holdout_training() -> None:
    with pytest.raises(ValueError, match="allowHoldoutTraining"):
        TrainingConfig(
            experimentKey="exp-v1",
            taskType="grounded_itinerary_generation",
            method="qlora",
            baseModelName="local/test-model",
            datasetPath="/tmp/export",
            datasetVersion="v1",
            allowHoldoutTraining=True,
        )


def test_training_config_accepts_grounded_itinerary_task() -> None:
    config = TrainingConfig(
        experimentKey="exp-v1",
        taskType="grounded_itinerary_generation",
        method="lora",
        baseModelName="local/test-model",
        datasetPath="/tmp/export",
        datasetVersion="v1",
    )

    assert config.task_type == "grounded_itinerary_generation"
    assert config.method == "lora"
