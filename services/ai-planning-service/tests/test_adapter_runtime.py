from pathlib import Path

from app.config import Settings, get_settings
from app.services.adapter_runtime import _checksum_path, validate_adapter_runtime


def test_adapter_runtime_disabled_has_no_path_metadata() -> None:
    status = validate_adapter_runtime(Settings())

    assert status.is_ready is True
    assert status.safe_metadata()["adapterEnabled"] is False
    assert "adapterPath" not in status.safe_metadata()


def test_adapter_runtime_verifies_checksum(tmp_path: Path) -> None:
    adapter_dir = tmp_path / "adapters" / "candidate"
    adapter_dir.mkdir(parents=True)
    (adapter_dir / "adapter_config.json").write_text('{"peft_type":"LORA"}\n')
    checksum = _checksum_path(adapter_dir)
    strict_status = validate_adapter_runtime(
        Settings(
            ai_adapter_enabled=True,
            ai_adapter_key="candidate-v1",
            ai_adapter_path=str(adapter_dir),
            ai_adapter_checksum=checksum,
            ai_model_artifact_dir=str(tmp_path / "adapters"),
        )
    )

    assert strict_status.is_ready is True
    assert strict_status.adapter_checksum_verified is True
    assert strict_status.safe_metadata()["adapterKey"] == "candidate-v1"


def test_get_settings_rejects_adapter_without_gate(monkeypatch) -> None:
    get_settings.cache_clear()
    monkeypatch.setenv("AI_ADAPTER_ENABLED", "true")
    monkeypatch.setenv("AI_MODEL_VARIANT", "adapter")
    monkeypatch.setenv("AI_ADAPTER_KEY", "candidate-v1")
    monkeypatch.setenv("AI_ADAPTER_PATH", "app/data/model-adapters/candidate-v1")

    try:
        try:
            get_settings()
        except ValueError as exc:
            assert "AI_ADAPTER_ENABLED requires" in str(exc)
        else:
            raise AssertionError("expected adapter gate validation to fail")
    finally:
        get_settings.cache_clear()
