# Add a training example type

1. Add the task type constant and allowed-task entry in `services/trip-service/internal/aidataset/types.go`.
2. Add the task type to migration CHECK constraints before applying the migration in a shared environment.
3. Add a manual example directory under `data/ai-training/manual/<task-with-dashes>/`.
4. Add scoring rules in `services/trip-service/internal/aidataset/scorer.go` when the task needs custom validation.
5. Add sanitizer/scorer/split tests in `services/trip-service/internal/aidataset`.
6. Document accepted input/output shape in `data/ai-training/schemas` or a task-specific README.
7. Run:

```bash
scripts/ai/validate-training-examples.sh
go test ./internal/aidataset
```
