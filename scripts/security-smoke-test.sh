#!/usr/bin/env bash
set -euo pipefail

# The canonical smoke flow already creates owner, viewer, and unrelated users
# and owns the evolving request schemas. This dedicated entry point runs that
# flow as the security gate instead of duplicating stale payload builders.
#
# Covered assertions include:
# - pending/viewer/removed/random-user access boundaries;
# - viewer edit/share-management denial;
# - password share lock/unlock, sanitized output, private-route denial, disable;
# - valid receipt upload/download/delete and invalid/unauthenticated access;
# - internal provider endpoints without and with a service token.
#
# Set the same environment variables documented by scripts/smoke-test.sh.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "Running Security & Privacy Hardening v1 smoke assertions..."
exec "${SCRIPT_DIR}/smoke-test.sh"

