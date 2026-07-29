DROP INDEX IF EXISTS idx_ai_model_feedback_trip_created;
DROP INDEX IF EXISTS idx_ai_model_feedback_deployment_created;
DROP TABLE IF EXISTS ai_model_feedback;

DROP INDEX IF EXISTS idx_ai_shadow_input_snapshots_expires;
DROP TABLE IF EXISTS ai_shadow_input_snapshots;

DROP INDEX IF EXISTS idx_ai_model_rollout_windows_deployment_start;
DROP TABLE IF EXISTS ai_model_rollout_windows;

DROP INDEX IF EXISTS idx_ai_model_online_comparisons_guardrail_created;
DROP INDEX IF EXISTS idx_ai_model_online_comparisons_candidate_created;
DROP TABLE IF EXISTS ai_model_online_comparisons;

DROP INDEX IF EXISTS idx_ai_model_request_assignments_trip_created;
DROP INDEX IF EXISTS idx_ai_model_request_assignments_user_created;
DROP TABLE IF EXISTS ai_model_request_assignments;

DROP INDEX IF EXISTS idx_ai_model_deployment_events_deployment_created;
DROP TABLE IF EXISTS ai_model_deployment_events;

DROP INDEX IF EXISTS idx_ai_model_deployments_environment_task_status;
DROP INDEX IF EXISTS idx_ai_model_deployments_one_active_per_env_task;
DROP TABLE IF EXISTS ai_model_deployments;
