# Release troubleshooting

| Problem | Checks and recovery |
| --- | --- |
| Image build failed | Run `./scripts/release/version-info.sh`, confirm Dockerfile build args and context, then retry the affected image with `build-images.sh --service <name>`. Do not use `latest` to mask a bad version tag. |
| Image push failed | Confirm `REGISTRY` is set and authenticated outside the script; rerun `push-images.sh`. The script never prints credentials. Verify both version and SHA tags exist. |
| Version metadata wrong | Compare `VERSION`, `GIT_SHA`, `BUILD_TIME`, and Docker build args. Rebuild the image; do not relabel a published image. Use `check-versions.sh` after deployment. |
| Migration failed | Stop rollout, preserve migration output, inspect dirty/version state, and use [migration safety](migration-safety.md). Prefer a reviewed forward fix; restore only from a verified backup. |
| Smoke test failed | Identify the first failed health/ready/version or business flow, inspect Compose logs, verify mock provider modes, and rerun the focused existing smoke suite after remediation. |
| Service unhealthy after release | Compare `/version` and `/ready`, dependency health, environment validation, database migrations, and RabbitMQ. Roll back only the affected compatible image when safe. |
| Frontend/backend contract mismatch | Run `scripts/contracts/validate-openapi.sh` and `check-generated.sh`; regenerate client types and update the contract changelog before release. |
| Stale generated client | Run `scripts/contracts/generate-web-client.sh`, inspect the diff, then typecheck the Web App. Commit generated output with the source OpenAPI change. |
| Missing environment variable | Use `scripts/validate-env.sh <target> --env-file <file>`. Add only non-secret examples to templates and mention operationally required values in release notes. |
| Registry authentication issue | Authenticate using the CI/registry-approved mechanism, verify repository permission and image name, then rerun explicit push. Never copy credentials into an env example, log, or release artifact. |
