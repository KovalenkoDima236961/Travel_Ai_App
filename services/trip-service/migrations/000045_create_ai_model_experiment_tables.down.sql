DROP INDEX IF EXISTS idx_ai_model_promotion_decisions_experiment_created;
DROP INDEX IF EXISTS idx_ai_model_evaluations_experiment_split;
DROP INDEX IF EXISTS idx_ai_model_adapters_status;
DROP INDEX IF EXISTS idx_ai_model_adapters_experiment;
DROP INDEX IF EXISTS idx_ai_training_experiments_task_status;
DROP INDEX IF EXISTS idx_ai_training_experiments_dataset_status;
DROP INDEX IF EXISTS idx_ai_models_status;

DROP TABLE IF EXISTS ai_model_promotion_decisions;
DROP TABLE IF EXISTS ai_model_evaluations;
DROP TABLE IF EXISTS ai_model_adapters;
DROP TABLE IF EXISTS ai_training_experiments;
DROP TABLE IF EXISTS ai_models;

ALTER TABLE ai_training_examples
    DROP CONSTRAINT IF EXISTS ai_training_examples_task_type_check;
ALTER TABLE ai_training_examples
    ADD CONSTRAINT ai_training_examples_task_type_check CHECK (task_type IN (
        'itinerary_generation',
        'day_regeneration',
        'item_regeneration',
        'place_replacement',
        'policy_repair',
        'budget_optimization',
        'route_alternatives',
        'checklist_generation',
        'copilot_response',
        'recap_generation'
    ));

ALTER TABLE ai_dataset_projects
    DROP CONSTRAINT IF EXISTS ai_dataset_projects_task_type_check;
ALTER TABLE ai_dataset_projects
    ADD CONSTRAINT ai_dataset_projects_task_type_check CHECK (task_type IN (
        'itinerary_generation',
        'day_regeneration',
        'item_regeneration',
        'place_replacement',
        'policy_repair',
        'budget_optimization',
        'route_alternatives',
        'checklist_generation',
        'copilot_response',
        'recap_generation'
    ));
