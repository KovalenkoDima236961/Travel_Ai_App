from __future__ import annotations

from app.model_router.models import InferenceMode, ModelRoutingRequestMetadata


def routing_metadata_from_request(request: object) -> dict[str, object]:
    if hasattr(request, "model_dump"):
        data = request.model_dump(by_alias=True)
    else:
        data = request
    metadata = ModelRoutingRequestMetadata.model_validate(data)
    payload: dict[str, object] = {
        "inferenceMode": (metadata.inference_mode or InferenceMode.PRIMARY).value,
    }
    if metadata.deployment_key:
        payload["deploymentKey"] = metadata.deployment_key
    if metadata.request_assignment_id:
        payload["requestAssignmentId"] = str(metadata.request_assignment_id)
    return payload
