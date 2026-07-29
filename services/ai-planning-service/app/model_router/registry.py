from __future__ import annotations

from app.config import Settings
from app.model_router.models import ModelVariant, RuntimeDeploymentMetadata


def default_runtime_deployment(settings: Settings) -> RuntimeDeploymentMetadata:
    variant = settings.ai_model_variant.strip().lower()
    if variant == "adapter":
        model_variant = ModelVariant.FINE_TUNED_CANDIDATE
    else:
        model_variant = ModelVariant.GROUNDED_BASELINE
    return RuntimeDeploymentMetadata(
        deploymentKey="configured-runtime",
        modelVariant=model_variant,
    )

