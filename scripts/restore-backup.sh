#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 full-YYYYMMDD-HHMMSS.tar.gz|/var/lib/langschool/backups/full-....tar.gz" >&2
  exit 2
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

SERVICE_NAME="${SERVICE_NAME:-langschool}"
BACKUPCTL_BIN="${BACKUPCTL_BIN:-/usr/local/bin/langschool-backupctl}"
HEALTHCHECK_URL="${HEALTHCHECK_URL:-http://127.0.0.1:8082/healthz}"
ARCHIVE_ARG="$1"

if [[ "$ARCHIVE_ARG" == */* ]]; then
  CONTAINER_ARCHIVE_PATH="$ARCHIVE_ARG"
else
  CONTAINER_ARCHIVE_PATH="/var/lib/langschool/backups/$ARCHIVE_ARG"
fi

docker compose stop "$SERVICE_NAME"
docker compose run --rm --no-deps --entrypoint "$BACKUPCTL_BIN" "$SERVICE_NAME" restore-full --archive "$CONTAINER_ARCHIVE_PATH"
docker compose up -d "$SERVICE_NAME"

for _ in {1..15}; do
  if curl --fail --silent --show-error "$HEALTHCHECK_URL" >/dev/null; then
    echo "Restore completed and health check passed: $HEALTHCHECK_URL"
    exit 0
  fi
  sleep 1
done

echo "Restore completed but health check did not pass in time: $HEALTHCHECK_URL" >&2
exit 1
