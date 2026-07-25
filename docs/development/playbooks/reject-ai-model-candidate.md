# Reject AI Model Candidate

1. Record the reason and failed gates in the experiment decision.
2. Keep the adapter disabled and unpromoted.
3. Preserve the experiment manifest, logs, metrics, and adapter checksum for audit.
4. If private data or holdout leakage is involved, invalidate the dataset version
   and follow the AI dataset incident response steps.
5. If the issue is quality only, decide whether to collect more curated examples
   or retrain with revised hyperparameters.
