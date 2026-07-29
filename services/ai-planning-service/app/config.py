import os
from functools import lru_cache
from urllib.parse import urlparse

from pydantic import BaseModel, Field, field_validator


class Settings(BaseModel):
    app_env: str = "local"
    http_host: str = "0.0.0.0"
    http_port: int = Field(default=8000, ge=1, le=65535)
    log_level: str = "INFO"
    ai_model_provider: str = "mock"
    ai_model_provider_fallback: str = "none"
    ai_model_provider_fallback_enabled: bool = True
    itinerary_generator_mode: str = "mock"
    copilot_enabled: bool = True
    copilot_mode: str = "mock"
    template_adaptation_enabled: bool = True
    template_adaptation_mode: str = "mock"
    template_adaptation_timeout_seconds: float = Field(default=120, gt=0)
    template_adaptation_fallback_enabled: bool = True
    trip_recap_enabled: bool = True
    trip_recap_mode: str = "mock"
    trip_recap_timeout_seconds: float = Field(default=30, gt=0)
    trip_recap_fallback_enabled: bool = True
    ollama_base_url: str = "http://ollama:11434"
    ollama_model: str = "llama3.1:8b"
    ollama_timeout_seconds: float = Field(default=60, gt=0)
    ollama_temperature: float = Field(default=0.2, ge=0)
    ollama_num_predict: int = 2048
    ollama_fallback_to_mock: bool = True
    ollama_repair_enabled: bool = True
    ollama_repair_attempts: int = Field(default=1, ge=0)
    log_llm_payloads: bool = False
    ai_fine_tuning_experiments_enabled: bool = False
    ai_adapter_inference_enabled: bool = False
    ai_adapter_staging_enabled: bool = False
    ai_adapter_enabled: bool = False
    ai_adapter_path: str = ""
    ai_adapter_key: str = ""
    ai_adapter_checksum: str = ""
    ai_adapter_experiment_key: str = ""
    ai_adapter_dataset_version: str = ""
    ai_adapter_fallback_to_base: bool = True
    ai_adapter_strict_load: bool = True
    ai_model_variant: str = "grounded_baseline"
    ai_model_load_timeout_seconds: int = Field(default=120, ge=1)
    ai_model_artifact_dir: str = "app/data/model-adapters"
    ai_prompt_logging_enabled: bool = False
    ai_prompt_logging_redacted_only: bool = True
    destination_context_enabled: bool = True
    destination_context_dir: str = "app/data/destinations"
    rag_enabled: bool = False
    rag_knowledge_dir: str = "app/data/knowledge"
    knowledge_curated_dir: str = "app/data/travel-knowledge"
    rag_chroma_dir: str = "app/data/chroma"
    rag_collection_name: str = "travel_knowledge"
    rag_top_k: int = 5
    rag_min_score: float = Field(default=0.0, ge=0)
    knowledge_indexing_enabled: bool = True
    knowledge_index_batch_size: int = Field(default=50, ge=1, le=250)
    knowledge_index_fail_open: bool = True
    chroma_anonymized_telemetry: bool = False
    ollama_embedding_model: str = "nomic-embed-text"
    ollama_embedding_timeout_seconds: float = Field(default=30, gt=0)
    openai_enabled: bool = False
    openai_api_key: str = ""
    openai_base_url: str = ""
    openai_organization: str = ""
    openai_project: str = ""
    openai_timeout_seconds: float = Field(default=90, gt=0, le=600)
    openai_connect_timeout_seconds: float = Field(default=10, gt=0, le=120)
    openai_max_retries: int = Field(default=2, ge=0, le=5)
    openai_retry_initial_delay_ms: int = Field(default=500, ge=0, le=60_000)
    openai_retry_max_delay_ms: int = Field(default=5_000, ge=0, le=120_000)
    openai_max_output_tokens: int | None = Field(default=None, gt=0)
    openai_store_responses: bool = False
    openai_store_responses_explicit: bool = False
    openai_model_default: str = ""
    openai_model_itinerary: str = ""
    openai_model_regeneration: str = ""
    openai_model_repair: str = ""
    openai_model_discovery: str = ""
    openai_model_route_alternatives: str = ""
    openai_model_budget_optimization: str = ""
    openai_model_checklist: str = ""
    openai_model_copilot: str = ""
    openai_model_recap: str = ""
    openai_model_evaluation: str = ""
    openai_usage_tracking_enabled: bool = True
    openai_cost_tracking_enabled: bool = True
    openai_daily_spend_limit_uah: float | None = Field(default=None, ge=0)
    openai_monthly_spend_limit_uah: float | None = Field(default=None, ge=0)
    openai_per_user_daily_generation_limit: int | None = Field(default=None, ge=1)
    openai_per_trip_daily_generation_limit: int | None = Field(default=None, ge=1)
    openai_per_user_copilot_requests_per_minute: int | None = Field(default=None, ge=1)
    openai_per_user_discovery_requests_per_day: int | None = Field(default=None, ge=1)
    openai_max_concurrent_requests: int | None = Field(default=None, ge=1)
    openai_max_input_tokens: int | None = Field(default=None, ge=1)
    openai_max_context_bytes: int | None = Field(default=None, ge=1)
    openai_max_grounding_places: int | None = Field(default=None, ge=1)
    openai_max_grounding_documents: int | None = Field(default=None, ge=1)
    openai_repair_enabled: bool = True
    openai_max_repair_attempts: int = Field(default=1, ge=0, le=1)
    openai_repair_timeout_seconds: float = Field(default=60, gt=0, le=300)
    openai_batch_enabled: bool = False
    openai_batch_model: str = ""
    openai_batch_max_requests: int | None = Field(default=None, ge=1)
    openai_batch_output_dir: str = ""

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

    def openai_model_for_operation(self, operation: str) -> str:
        normalized = operation.strip().lower()
        mapping = {
            "generate_itinerary": self.openai_model_itinerary,
            "itinerary": self.openai_model_itinerary,
            "regenerate": self.openai_model_regeneration,
            "regenerate_day": self.openai_model_regeneration,
            "regenerate_item": self.openai_model_regeneration,
            "repair": self.openai_model_repair,
            "repair_itinerary": self.openai_model_repair,
            "repair_generation_output": self.openai_model_repair,
            "suggest_destinations": self.openai_model_discovery,
            "discovery": self.openai_model_discovery,
            "suggest_route_alternatives": self.openai_model_route_alternatives,
            "route_alternatives": self.openai_model_route_alternatives,
            "optimize_budget_day": self.openai_model_budget_optimization,
            "budget_optimization": self.openai_model_budget_optimization,
            "generate_checklist": self.openai_model_checklist,
            "checklist": self.openai_model_checklist,
            "copilot_respond": self.openai_model_copilot,
            "copilot": self.openai_model_copilot,
            "generate_trip_recap": self.openai_model_recap,
            "recap": self.openai_model_recap,
            "evaluation": self.openai_model_evaluation,
        }
        return (mapping.get(normalized) or self.openai_model_default).strip()


def _env_string(name: str, default: str) -> str:
    value = os.getenv(name)
    if value is None or value.strip() == "":
        return default
    return value.strip()


def _env_optional_int(name: str) -> int | None:
    raw_value = os.getenv(name)
    if raw_value is None or raw_value.strip() == "":
        return None
    return int(raw_value)


def _env_optional_float(name: str) -> float | None:
    raw_value = os.getenv(name)
    if raw_value is None or raw_value.strip() == "":
        return None
    return float(raw_value)


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
    default_provider = _env_string("AI_MODEL_PROVIDER", "mock")
    settings = Settings(
        app_env=_env_string("APP_ENV", "local"),
        http_host=_env_string("HTTP_HOST", "0.0.0.0"),
        http_port=_env_int("HTTP_PORT", 8000),
        log_level=_env_string("LOG_LEVEL", "INFO").upper(),
        ai_model_provider=default_provider,
        ai_model_provider_fallback=_env_string("AI_MODEL_PROVIDER_FALLBACK", "none"),
        ai_model_provider_fallback_enabled=_env_bool("AI_MODEL_PROVIDER_FALLBACK_ENABLED", True),
        itinerary_generator_mode=_env_string("ITINERARY_GENERATOR_MODE", default_provider),
        copilot_enabled=_env_bool("COPILOT_ENABLED", True),
        copilot_mode=_env_string(
            "COPILOT_MODE", _env_string("ITINERARY_GENERATOR_MODE", default_provider)
        ),
        template_adaptation_enabled=_env_bool("AI_TEMPLATE_ADAPTATION_ENABLED", True),
        template_adaptation_mode=_env_string("AI_TEMPLATE_ADAPTATION_MODE", "mock"),
        template_adaptation_timeout_seconds=_env_float(
            "AI_TEMPLATE_ADAPTATION_TIMEOUT_SECONDS", 120
        ),
        template_adaptation_fallback_enabled=_env_bool(
            "AI_TEMPLATE_ADAPTATION_FALLBACK_ENABLED", True
        ),
        trip_recap_enabled=_env_bool("TRIP_RECAP_ENABLED", True),
        trip_recap_mode=_env_string(
            "TRIP_RECAP_AI_MODE", _env_string("ITINERARY_GENERATOR_MODE", default_provider)
        ),
        trip_recap_timeout_seconds=_env_float("TRIP_RECAP_TIMEOUT_SECONDS", 30),
        trip_recap_fallback_enabled=_env_bool("TRIP_RECAP_FALLBACK_ENABLED", True),
        ollama_base_url=_env_string("OLLAMA_BASE_URL", "http://ollama:11434"),
        ollama_model=_env_string("OLLAMA_MODEL", "llama3.1:8b"),
        ollama_timeout_seconds=_env_float("OLLAMA_TIMEOUT_SECONDS", 60),
        ollama_temperature=_env_float("OLLAMA_TEMPERATURE", 0.2),
        ollama_num_predict=_env_int("OLLAMA_NUM_PREDICT", 2048),
        ollama_fallback_to_mock=_env_bool("OLLAMA_FALLBACK_TO_MOCK", True),
        ollama_repair_enabled=_env_bool("OLLAMA_REPAIR_ENABLED", True),
        ollama_repair_attempts=_env_int("OLLAMA_REPAIR_ATTEMPTS", 1),
        log_llm_payloads=_env_bool("LOG_LLM_PAYLOADS", False),
        ai_fine_tuning_experiments_enabled=_env_bool("AI_FINE_TUNING_EXPERIMENTS_ENABLED", False),
        ai_adapter_inference_enabled=_env_bool("AI_ADAPTER_INFERENCE_ENABLED", False),
        ai_adapter_staging_enabled=_env_bool("AI_ADAPTER_STAGING_ENABLED", False),
        ai_adapter_enabled=_env_bool("AI_ADAPTER_ENABLED", False),
        ai_adapter_path=_env_string("AI_ADAPTER_PATH", ""),
        ai_adapter_key=_env_string("AI_ADAPTER_KEY", ""),
        ai_adapter_checksum=_env_string("AI_ADAPTER_CHECKSUM", ""),
        ai_adapter_experiment_key=_env_string("AI_ADAPTER_EXPERIMENT_KEY", ""),
        ai_adapter_dataset_version=_env_string("AI_ADAPTER_DATASET_VERSION", ""),
        ai_adapter_fallback_to_base=_env_bool("AI_ADAPTER_FALLBACK_TO_BASE", True),
        ai_adapter_strict_load=_env_bool("AI_ADAPTER_STRICT_LOAD", True),
        ai_model_variant=_env_string("AI_MODEL_VARIANT", "grounded_baseline"),
        ai_model_load_timeout_seconds=_env_int("AI_MODEL_LOAD_TIMEOUT_SECONDS", 120),
        ai_model_artifact_dir=_env_string("AI_MODEL_ARTIFACT_DIR", "app/data/model-adapters"),
        ai_prompt_logging_enabled=_env_bool("AI_PROMPT_LOGGING_ENABLED", False),
        ai_prompt_logging_redacted_only=_env_bool("AI_PROMPT_LOGGING_REDACTED_ONLY", True),
        destination_context_enabled=_env_bool("DESTINATION_CONTEXT_ENABLED", True),
        destination_context_dir=_env_string("DESTINATION_CONTEXT_DIR", "app/data/destinations"),
        rag_enabled=_env_bool("RAG_ENABLED", False),
        rag_knowledge_dir=_env_string("RAG_KNOWLEDGE_DIR", "app/data/knowledge"),
        knowledge_curated_dir=_env_string("KNOWLEDGE_CURATED_DIR", "app/data/travel-knowledge"),
        rag_chroma_dir=_env_string("RAG_CHROMA_DIR", "app/data/chroma"),
        rag_collection_name=_env_string("RAG_COLLECTION_NAME", "travel_knowledge"),
        rag_top_k=_env_int("RAG_TOP_K", 5),
        rag_min_score=_env_float("RAG_MIN_SCORE", 0.0),
        knowledge_indexing_enabled=_env_bool("KNOWLEDGE_INDEXING_ENABLED", True),
        knowledge_index_batch_size=_env_int("KNOWLEDGE_INDEX_BATCH_SIZE", 50),
        knowledge_index_fail_open=_env_bool("KNOWLEDGE_INDEX_FAIL_OPEN", True),
        chroma_anonymized_telemetry=_env_bool("ANONYMIZED_TELEMETRY", False),
        ollama_embedding_model=_env_string("OLLAMA_EMBEDDING_MODEL", "nomic-embed-text"),
        ollama_embedding_timeout_seconds=_env_float("OLLAMA_EMBEDDING_TIMEOUT_SECONDS", 30),
        openai_enabled=_env_bool("OPENAI_ENABLED", False),
        openai_api_key=_env_string("OPENAI_API_KEY", ""),
        openai_base_url=_env_string("OPENAI_BASE_URL", ""),
        openai_organization=_env_string("OPENAI_ORGANIZATION", ""),
        openai_project=_env_string("OPENAI_PROJECT", ""),
        openai_timeout_seconds=_env_float("OPENAI_TIMEOUT_SECONDS", 90),
        openai_connect_timeout_seconds=_env_float("OPENAI_CONNECT_TIMEOUT_SECONDS", 10),
        openai_max_retries=_env_int("OPENAI_MAX_RETRIES", 2),
        openai_retry_initial_delay_ms=_env_int("OPENAI_RETRY_INITIAL_DELAY_MS", 500),
        openai_retry_max_delay_ms=_env_int("OPENAI_RETRY_MAX_DELAY_MS", 5_000),
        openai_max_output_tokens=_env_optional_int("OPENAI_MAX_OUTPUT_TOKENS"),
        openai_store_responses=_env_bool("OPENAI_STORE_RESPONSES", False),
        openai_store_responses_explicit=os.getenv("OPENAI_STORE_RESPONSES") is not None,
        openai_model_default=_env_string("OPENAI_MODEL_DEFAULT", ""),
        openai_model_itinerary=_env_string("OPENAI_MODEL_ITINERARY", ""),
        openai_model_regeneration=_env_string("OPENAI_MODEL_REGENERATION", ""),
        openai_model_repair=_env_string("OPENAI_MODEL_REPAIR", ""),
        openai_model_discovery=_env_string("OPENAI_MODEL_DISCOVERY", ""),
        openai_model_route_alternatives=_env_string("OPENAI_MODEL_ROUTE_ALTERNATIVES", ""),
        openai_model_budget_optimization=_env_string("OPENAI_MODEL_BUDGET_OPTIMIZATION", ""),
        openai_model_checklist=_env_string("OPENAI_MODEL_CHECKLIST", ""),
        openai_model_copilot=_env_string("OPENAI_MODEL_COPILOT", ""),
        openai_model_recap=_env_string("OPENAI_MODEL_RECAP", ""),
        openai_model_evaluation=_env_string("OPENAI_MODEL_EVALUATION", ""),
        openai_usage_tracking_enabled=_env_bool("OPENAI_USAGE_TRACKING_ENABLED", True),
        openai_cost_tracking_enabled=_env_bool("OPENAI_COST_TRACKING_ENABLED", True),
        openai_daily_spend_limit_uah=_env_optional_float("OPENAI_DAILY_SPEND_LIMIT_UAH"),
        openai_monthly_spend_limit_uah=_env_optional_float("OPENAI_MONTHLY_SPEND_LIMIT_UAH"),
        openai_per_user_daily_generation_limit=_env_optional_int(
            "OPENAI_PER_USER_DAILY_GENERATION_LIMIT"
        ),
        openai_per_trip_daily_generation_limit=_env_optional_int(
            "OPENAI_PER_TRIP_DAILY_GENERATION_LIMIT"
        ),
        openai_per_user_copilot_requests_per_minute=_env_optional_int(
            "OPENAI_PER_USER_COPILOT_REQUESTS_PER_MINUTE"
        ),
        openai_per_user_discovery_requests_per_day=_env_optional_int(
            "OPENAI_PER_USER_DISCOVERY_REQUESTS_PER_DAY"
        ),
        openai_max_concurrent_requests=_env_optional_int("OPENAI_MAX_CONCURRENT_REQUESTS"),
        openai_max_input_tokens=_env_optional_int("OPENAI_MAX_INPUT_TOKENS"),
        openai_max_context_bytes=_env_optional_int("OPENAI_MAX_CONTEXT_BYTES"),
        openai_max_grounding_places=_env_optional_int("OPENAI_MAX_GROUNDING_PLACES"),
        openai_max_grounding_documents=_env_optional_int("OPENAI_MAX_GROUNDING_DOCUMENTS"),
        openai_repair_enabled=_env_bool("OPENAI_REPAIR_ENABLED", True),
        openai_max_repair_attempts=_env_int("OPENAI_MAX_REPAIR_ATTEMPTS", 1),
        openai_repair_timeout_seconds=_env_float("OPENAI_REPAIR_TIMEOUT_SECONDS", 60),
        openai_batch_enabled=_env_bool("OPENAI_BATCH_ENABLED", False),
        openai_batch_model=_env_string("OPENAI_BATCH_MODEL", ""),
        openai_batch_max_requests=_env_optional_int("OPENAI_BATCH_MAX_REQUESTS"),
        openai_batch_output_dir=_env_string("OPENAI_BATCH_OUTPUT_DIR", ""),
    )
    _validate_startup_settings(settings)
    return settings


def _validate_startup_settings(settings: Settings) -> None:
    provider = settings.ai_model_provider.strip().lower()
    if provider not in {"mock", "ollama", "openai"}:
        raise ValueError("AI_MODEL_PROVIDER must be mock, ollama, or openai")
    fallback_provider = settings.ai_model_provider_fallback.strip().lower()
    if fallback_provider not in {"none", "mock", "ollama"}:
        raise ValueError("AI_MODEL_PROVIDER_FALLBACK must be none, mock, or ollama")
    mode = settings.itinerary_generator_mode.strip().lower()
    if mode not in {"mock", "ollama", "openai"}:
        raise ValueError("ITINERARY_GENERATOR_MODE must be mock, ollama, or openai")
    copilot_mode = settings.copilot_mode.strip().lower()
    if copilot_mode not in {"mock", "ollama", "openai"}:
        raise ValueError("COPILOT_MODE must be mock, ollama, or openai")
    adaptation_mode = settings.template_adaptation_mode.strip().lower()
    if adaptation_mode not in {"mock", "ollama"}:
        raise ValueError("AI_TEMPLATE_ADAPTATION_MODE must be mock or ollama")
    if adaptation_mode == "ollama":
        _validate_http_url("OLLAMA_BASE_URL", settings.ollama_base_url)
    trip_recap_mode = settings.trip_recap_mode.strip().lower()
    if trip_recap_mode not in {"mock", "ollama", "openai"}:
        raise ValueError("TRIP_RECAP_AI_MODE must be mock, ollama, or openai")
    if trip_recap_mode == "ollama":
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
    if copilot_mode == "ollama":
        _validate_http_url("OLLAMA_BASE_URL", settings.ollama_base_url)
    if settings.rag_enabled and not settings.rag_chroma_dir.strip():
        raise ValueError("RAG_CHROMA_DIR is required when RAG_ENABLED=true")
    model_variant = settings.ai_model_variant.strip().lower()
    if model_variant not in {"base", "grounded_baseline", "adapter"}:
        raise ValueError("AI_MODEL_VARIANT must be base, grounded_baseline, or adapter")
    adapter_gate_enabled = settings.ai_adapter_inference_enabled or (
        settings.app_env in {"local", "development", "test", "staging"}
        and settings.ai_adapter_staging_enabled
    )
    if settings.ai_adapter_enabled and not adapter_gate_enabled:
        raise ValueError(
            "AI_ADAPTER_ENABLED requires AI_ADAPTER_INFERENCE_ENABLED=true or "
            "AI_ADAPTER_STAGING_ENABLED=true outside production"
        )
    if settings.app_env == "production" and settings.ai_adapter_enabled:
        if not settings.ai_adapter_inference_enabled:
            raise ValueError(
                "Production adapter inference requires AI_ADAPTER_INFERENCE_ENABLED=true"
            )
        if settings.ai_adapter_staging_enabled:
            raise ValueError("AI_ADAPTER_STAGING_ENABLED must be false in production")
    if settings.ai_adapter_enabled:
        if model_variant != "adapter":
            raise ValueError("AI_MODEL_VARIANT must be adapter when AI_ADAPTER_ENABLED=true")
        if not settings.ai_adapter_key.strip():
            raise ValueError("AI_ADAPTER_KEY is required when AI_ADAPTER_ENABLED=true")
        if not settings.ai_adapter_path.strip():
            raise ValueError("AI_ADAPTER_PATH is required when AI_ADAPTER_ENABLED=true")
        if settings.is_strict_env and not settings.ai_adapter_checksum.strip():
            raise ValueError("AI_ADAPTER_CHECKSUM is required in staging and production")
        if settings.ai_adapter_checksum.strip() and len(settings.ai_adapter_checksum.strip()) < 32:
            raise ValueError("AI_ADAPTER_CHECKSUM must be at least 32 characters")
    _validate_openai_settings(settings, mode, copilot_mode, trip_recap_mode)


def _validate_openai_settings(
    settings: Settings, itinerary_mode: str, copilot_mode: str, trip_recap_mode: str
) -> None:
    selected_modes = {settings.ai_model_provider.strip().lower(), itinerary_mode}
    if settings.copilot_enabled:
        selected_modes.add(copilot_mode)
    if settings.trip_recap_enabled:
        selected_modes.add(trip_recap_mode)
    openai_selected = "openai" in selected_modes

    if settings.openai_base_url.strip():
        _validate_http_url("OPENAI_BASE_URL", settings.openai_base_url)
    if settings.openai_connect_timeout_seconds > settings.openai_timeout_seconds:
        raise ValueError("OPENAI_CONNECT_TIMEOUT_SECONDS cannot exceed OPENAI_TIMEOUT_SECONDS")
    if settings.openai_retry_initial_delay_ms > settings.openai_retry_max_delay_ms:
        raise ValueError("OPENAI_RETRY_INITIAL_DELAY_MS cannot exceed OPENAI_RETRY_MAX_DELAY_MS")
    if settings.openai_batch_enabled and not (
        settings.openai_batch_model.strip() or settings.openai_model_evaluation.strip()
    ):
        raise ValueError("OPENAI_BATCH_MODEL or OPENAI_MODEL_EVALUATION is required for Batch")

    if not openai_selected:
        return
    if not settings.openai_enabled:
        raise ValueError("OPENAI_ENABLED must be true when OpenAI is selected")
    if not settings.openai_api_key.strip():
        raise ValueError("OPENAI_API_KEY is required when OpenAI is selected")
    if settings.app_env == "production" and _looks_like_placeholder_key(settings.openai_api_key):
        raise ValueError("OPENAI_API_KEY must not be a placeholder in production")
    if settings.app_env == "production" and not settings.openai_store_responses_explicit:
        raise ValueError("OPENAI_STORE_RESPONSES must be explicit in production")

    required_operations: list[str] = []
    if itinerary_mode == "openai" or settings.ai_model_provider.strip().lower() == "openai":
        required_operations.extend(
            [
                "generate_itinerary",
                "regenerate_day",
                "regenerate_item",
                "repair_itinerary",
                "repair_generation_output",
                "suggest_destinations",
                "suggest_route_alternatives",
                "optimize_budget_day",
                "generate_checklist",
            ]
        )
    if copilot_mode == "openai":
        required_operations.append("copilot_respond")
    if trip_recap_mode == "openai":
        required_operations.append("generate_trip_recap")

    missing = [
        operation
        for operation in sorted(set(required_operations))
        if not settings.openai_model_for_operation(operation)
    ]
    if missing:
        raise ValueError(
            "OpenAI model configuration is missing for operation(s): " + ", ".join(missing)
        )


def _looks_like_placeholder_key(value: str) -> bool:
    normalized = value.strip().lower()
    return any(
        marker in normalized
        for marker in (
            "placeholder",
            "changeme",
            "change-me",
            "dummy",
            "example",
            "set_in_secret",
            "test-key",
        )
    )
