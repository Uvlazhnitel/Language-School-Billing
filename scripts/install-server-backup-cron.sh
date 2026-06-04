#!/usr/bin/env bash
set -euo pipefail

CRON_EXPR="${CRON_EXPR:-15 3 * * *}"
ROOT_DIR="${ROOT_DIR:-/home/ilya/langschool}"
LOG_FILE="${LOG_FILE:-/home/ilya/langschool-data/backups/create-full-backup.log}"
TMP_FILE="$(mktemp)"
trap 'rm -f "$TMP_FILE"' EXIT

mkdir -p "$(dirname "$LOG_FILE")"

CRON_CMD="cd ${ROOT_DIR} && ./scripts/create-full-backup.sh >> ${LOG_FILE} 2>&1"
CRON_LINE="${CRON_EXPR} ${CRON_CMD}"

crontab -l 2>/dev/null | grep -vF "$CRON_CMD" >"$TMP_FILE" || true
printf '%s\n' "$CRON_LINE" >>"$TMP_FILE"
crontab "$TMP_FILE"

echo "Installed cron entry:"
echo "$CRON_LINE"
