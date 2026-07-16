import os
from functools import lru_cache
from urllib.parse import urlparse

from pydantic import BaseModel, Field, field_validator


class Settings(BaseModel):
    app_env: str = "local"
    http_host: str = "0.0.0.0"
    http_port: int = Field(default=8000, ge=1, le=65535)
    log_level: str = "INFO"
    itinerary_generator_mode: str = "mock"
    template_adaptation_enabled: bool = True
    template_adaptation_mode: str = "mock"
    template_adaptation_timeout_seconds: float = Field(default=120, gt=0)
    template_adaptation_fallback_enabled: bool = True
    ollama_base_url: str = "http://ollama:11434"
    ollama_model: str = "llama3.1:8b"
    ollama_timeout_seconds: float = Field(default=60, gt=0)
    ollama_temperature: float = Field(default=0.2, ge=0)
    ollama_num_predict: int = 2048
    ollama_fallback_to_mock: bool = True
    ollama_repair_enabled: bool = True
    ollama_repair_attempts: int = Field(default=1, ge=0)
    log_llm_payloads: bool = False
    ai_prompt_logging_enabled: bool = False
    ai_prompt_logging_redacted_only: bool = True
    destination_context_enabled: bool = True
    destination_context_dir: str = "app/data/destinations"
    rag_enabled: bool = False
    rag_knowledge_dir: str = "app/data/knowledge"
    rag_chroma_dir: str = "app/data/chroma"
    rag_collection_name: str = "travel_knowledge"
    rag_top_k: int = 5
    rag_min_score: float = Field(default=0.0, ge=0)
    chroma_anonymized_telemetry: bool = False
    ollama_embedding_model: str = "nomic-embed-text"
    ollama_embedding_timeout_seconds: float = Field(default=30, gt=0)

    @field_validator("ollama_repair_attempts")
    @classmethod
    def clamp_ollama_repair_attempts(cls, value: int) -> int:
        return min(value, 1)

    @field_validator("app_env")
    @classmethod
    def normalize_app_env(cls, value: str) -> str:
        normalized = value.strip().lower()
        if normalized not in {"local", "staging", "production", "development", "test"}:
            raise ValueError("APP_ENV must be local, staging, or production")
        return normalized

    @field_validator("rag_top_k")
    @classmethod
    def clamp_rag_top_k(cls, value: int) -> int:
        return min(max(value, 1), 10)

    @property
    def allow_llm_payload_logging(self) -> bool:
        enabled = self.ai_prompt_logging_enabled or self.log_llm_payloads
        return (
            enabled
            and self.ai_prompt_logging_redacted_only
            and self.app_env in {"local", "development", "test"}
        )

    @property
    def is_strict_env(self) -> bool:
        return self.app_env in {"staging", "production"}


def _env_string(name: str, default: str) -> str:
    value = os.getenv(name)
    if value is None or value.strip() == "":
        return default
    return value.strip()


def _env_int(name: str, default: int) -> int:
    raw_value = os.getenv(name)
    if raw_value is None or raw_value.strip() == "":
        return default
    return int(raw_value)


def _env_float(name: str, default: float) -> float:
    raw_value = os.getenv(name)
    if raw_value is None or raw_value.strip() == "":
        return default
    return float(raw_value)


def _env_bool(name: str, default: bool) -> bool:
    raw_value = os.getenv(name)
    if raw_value is None or raw_value.strip() == "":
        return default

    normalized = raw_value.strip().lower()
    if normalized in {"1", "true", "yes", "on"}:
        return True
    if normalized in {"0", "false", "no", "off"}:
        return False

    raise ValueError(f"{name} must be a boolean value")


def _validate_http_url(name: str, value: str) -> None:
    parsed = urlparse(value.strip())
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise ValueError(f"{name} must be a valid http/https URL")


@lru_cache
def get_settings() -> Settings:
    settings = Settings(
        app_env=_env_string("APP_ENV", "local"),
        http_host=_env_string("HTTP_HOST", "0.0.0.0"),
        http_port=_env_int("HTTP_PORT", 8000),
        log_level=_env_string("LOG_LEVEL", "INFO").upper(),
        itinerary_generator_mode=_env_string("ITINERARY_GENERATOR_MODE", "mock"),
        template_adaptation_enabled=_env_bool("AI_TEMPLATE_ADAPTATION_ENABLED", True),
        template_adaptation_mode=_env_string(
            "AI_TEMPLATE_ADAPTATION_MODE",
            _env_string("ITINERARY_GENERATOR_MODE", "mock"),
        ),
        template_adaptation_timeout_seconds=_env_float(
            "AI_TEMPLATE_ADAPTATION_TIMEOUT_SECONDS", 120
        ),
        template_adaptation_fallback_enabled=_env_bool(
            "AI_TEMPLATE_ADAPTATION_FALLBACK_ENABLED", True
        ),
        ollama_base_url=_env_string("OLLAMA_BASE_URL", "http://ollama:11434"),
        ollama_model=_env_string("OLLAMA_MODEL", "llama3.1:8b"),
        ollama_timeout_seconds=_env_float("OLLAMA_TIMEOUT_SECONDS", 60),
        ollama_temperature=_env_float("OLLAMA_TEMPERATURE", 0.2),
        ollama_num_predict=_env_int("OLLAMA_NUM_PREDICT", 2048),
        ollama_fallback_to_mock=_env_bool("OLLAMA_FALLBACK_TO_MOCK", True),
        ollama_repair_enabled=_env_bool("OLLAMA_REPAIR_ENABLED", True),
        ollama_repair_attempts=_env_int("OLLAMA_REPAIR_ATTEMPTS", 1),
        log_llm_payloads=_env_bool("LOG_LLM_PAYLOADS", False),
        ai_prompt_logging_enabled=_env_bool("AI_PROMPT_LOGGING_ENABLED", False),
        ai_prompt_logging_redacted_only=_env_bool("AI_PROMPT_LOGGING_REDACTED_ONLY", True),
        destination_context_enabled=_env_bool("DESTINATION_CONTEXT_ENABLED", True),
        destination_context_dir=_env_string("DESTINATION_CONTEXT_DIR", "app/data/destinations"),
        rag_enabled=_env_bool("RAG_ENABLED", False),
        rag_knowledge_dir=_env_string("RAG_KNOWLEDGE_DIR", "app/data/knowledge"),
        rag_chroma_dir=_env_string("RAG_CHROMA_DIR", "app/data/chroma"),
        rag_collection_name=_env_string("RAG_COLLECTION_NAME", "travel_knowledge"),
        rag_top_k=_env_int("RAG_TOP_K", 5),
        rag_min_score=_env_float("RAG_MIN_SCORE", 0.0),
        chroma_anonymized_telemetry=_env_bool("ANONYMIZED_TELEMETRY", False),
        ollama_embedding_model=_env_string("OLLAMA_EMBEDDING_MODEL", "nomic-embed-text"),
        ollama_embedding_timeout_seconds=_env_float("OLLAMA_EMBEDDING_TIMEOUT_SECONDS", 30),
    )
    _validate_startup_settings(settings)
    return settings


def _validate_startup_settings(settings: Settings) -> None:
    mode = settings.itinerary_generator_mode.strip().lower()
    if mode not in {"mock", "ollama"}:
        raise ValueError("ITINERARY_GENERATOR_MODE must be mock or ollama")
    adaptation_mode = settings.template_adaptation_mode.strip().lower()
    if adaptation_mode not in {"mock", "ollama"}:
        raise ValueError("AI_TEMPLATE_ADAPTATION_MODE must be mock or ollama")
    if adaptation_mode == "ollama":
        _validate_http_url("OLLAMA_BASE_URL", settings.ollama_base_url)
    if settings.is_strict_env and (settings.log_llm_payloads or settings.ai_prompt_logging_enabled):
        raise ValueError("AI prompt logging must be false in staging or production")
    if (
        settings.ai_prompt_logging_enabled or settings.log_llm_payloads
    ) and not settings.ai_prompt_logging_redacted_only:
        raise ValueError(
            "AI_PROMPT_LOGGING_REDACTED_ONLY must be true when prompt logging is enabled"
        )
    if mode == "ollama":
        _validate_http_url("OLLAMA_BASE_URL", settings.ollama_base_url)
    if settings.rag_enabled and not settings.rag_chroma_dir.strip():
        raise ValueError("RAG_CHROMA_DIR is required when RAG_ENABLED=true")
