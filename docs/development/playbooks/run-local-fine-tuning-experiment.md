# Run Local Fine-Tuning Experiment

1. Confirm the task is `grounded_itinerary_generation`.
2. Export a frozen curated dataset version from Trip Service.
3. Run `scripts/ai/validate-dataset-export.sh <export-dir>`.
4. Create a training config using `services/ai-planning-service/training/config.schema.json`.
5. Run `scripts/ai/validate-training-run.sh --config <config> --dataset-path <export-dir>`.
6. On the local training host, set `AI_FINE_TUNING_EXPERIMENTS_ENABLED=true`.
7. Run `scripts/ai/run-local-fine-tuning.sh --config <config> --evaluate-after`.
8. Record the experiment manifest and adapter checksum.

Do not run full training in CI or against raw production data.
