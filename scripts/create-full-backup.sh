#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

SERVICE_NAME="${SERVICE_NAME:-langschool}"
BACKUPCTL_BIN="${BACKUPCTL_BIN:-/usr/local/bin/langschool-backupctl}"

docker compose run --rm --no-deps --entrypoint "$BACKUPCTL_BIN" "$SERVICE_NAME" create-full
