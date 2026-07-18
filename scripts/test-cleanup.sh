#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cache_root="$root/.cache/go-build-cleanup"
(cd "$root/services/worker-service" && env GOCACHE="$cache_root/worker" go test ./internal/cleanup ./internal/config ./internal/httpserver)
(cd "$root/services/auth-service" && env GOCACHE="$cache_root/auth" go test ./internal/cleanup ./internal/config ./internal/httpserver/...)
(cd "$root/services/notification-service" && env GOCACHE="$cache_root/notification" go test ./internal/cleanup ./internal/config ./internal/httpserver/...)
(cd "$root/services/external-integrations-service" && env GOCACHE="$cache_root/external" go test ./internal/cleanup ./internal/config ./internal/httpserver/...)
