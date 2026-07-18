# Cleanup failed

1. Open `GET /ops/cleanup/runs` as an ops admin and identify the task/status/request ID.
2. Check Worker Service logs for the aggregate `cleanup_run` entry and the owning-service logs for the same request ID. Do not request or log row contents, tokens, OCR, or filenames.
3. Confirm `CLEANUP_JOBS_ENABLED`, `WORKER_SCHEDULED_JOBS_ENABLED`, the internal service token, and the owning service URL.
4. Check `cleanup_runs` for a running row or an expired lock; do not force-delete the row. Lock expiry allows a safe later retry.
5. Run the task manually as a dry run: `POST /ops/cleanup/run` with `dryRun: true`, a small batch, and a small batch count.
6. Only after reviewing aggregate eligibility and backup posture, run a bounded destructive cleanup.

Do not rerun an audit cleanup, receipt cleanup, or production backup cleanup solely to reduce storage. Escalate if a task reports an unsafe path, persistent internal-auth failure, or unexpected protected-record eligibility.
