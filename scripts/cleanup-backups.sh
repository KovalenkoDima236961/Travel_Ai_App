#!/usr/bin/env bash
set -euo pipefail

dry_run=true
confirmed=false
for arg in "$@"; do
  case "$arg" in
    --dry-run) dry_run=true ;;
    --yes) dry_run=false; confirmed=true ;;
    *) echo "usage: $0 [--dry-run|--yes]" >&2; exit 2 ;;
  esac
done

backup_dir="${BACKUP_DIR:-}"
retention_days="${RETENTION_LOCAL_BACKUPS_DAYS:-30}"
app_env="${APP_ENV:-local}"
if [[ -z "$backup_dir" || ! "$retention_days" =~ ^[1-9][0-9]*$ ]]; then
  echo "BACKUP_DIR and a positive RETENTION_LOCAL_BACKUPS_DAYS are required." >&2
  exit 2
fi
if [[ "$dry_run" == false && "$confirmed" != true ]]; then
  echo "Destructive cleanup requires --yes." >&2
  exit 2
fi
case "$app_env" in
  local|development|test) ;;
  *)
    if [[ "${ALLOW_PRODUCTION_BACKUP_CLEANUP:-false}" != "true" ]]; then
      echo "Backup cleanup is local/dev/test only. Set ALLOW_PRODUCTION_BACKUP_CLEANUP=true only after following the production backup runbook." >&2
      exit 2
    fi
    ;;
esac
if [[ ! -d "$backup_dir" ]]; then
  echo "BACKUP_DIR does not exist or is not a directory." >&2
  exit 2
fi

backup_abs="$(cd "$backup_dir" && pwd -P)"
case "$backup_abs" in
  /|/tmp|/private/tmp|/var|/private/var|"")
    echo "Refusing unsafe BACKUP_DIR." >&2
    exit 2
    ;;
esac

files=0
bytes=0
while IFS= read -r -d '' file; do
  # find is rooted at the resolved configured directory. Do not recurse through
  # symlinks and do not remove anything outside this directory.
  size="$(stat -f '%z' "$file")"
  files=$((files + 1))
  bytes=$((bytes + size))
  if [[ "$dry_run" == false ]]; then
    rm -- "$file"
  fi
done < <(find -P "$backup_abs" -type f -mtime "+$retention_days" -print0)

mode="dry-run"
[[ "$dry_run" == false ]] && mode="deleted"
echo "Backup cleanup $mode: files=$files bytesFreed=$bytes retentionDays=$retention_days"
