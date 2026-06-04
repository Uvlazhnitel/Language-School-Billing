#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${ROOT_DIR:-/home/ilya/langschool}"
DB_CRON_EXPR="${DB_CRON_EXPR:-15 3 * * *}"
FULL_CRON_EXPR="${FULL_CRON_EXPR:-0 4 * * 0}"
DB_LOG_FILE="${DB_LOG_FILE:-/home/ilya/langschool-data/backups/create-db-backup.log}"
FULL_LOG_FILE="${FULL_LOG_FILE:-/home/ilya/langschool-data/backups/create-full-backup.log}"
TMP_FILE="$(mktemp)"
trap 'rm -f "$TMP_FILE"' EXIT

mkdir -p "$(dirname "$DB_LOG_FILE")" "$(dirname "$FULL_LOG_FILE")"

DB_CRON_CMD="cd ${ROOT_DIR} && ./scripts/create-db-backup.sh >> ${DB_LOG_FILE} 2>&1"
DB_CRON_LINE="${DB_CRON_EXPR} ${DB_CRON_CMD}"
FULL_CRON_CMD="cd ${ROOT_DIR} && ./scripts/create-full-backup.sh >> ${FULL_LOG_FILE} 2>&1"
FULL_CRON_LINE="${FULL_CRON_EXPR} ${FULL_CRON_CMD}"

crontab -l 2>/dev/null | grep -vF "$DB_CRON_CMD" | grep -vF "$FULL_CRON_CMD" >"$TMP_FILE" || true
printf '%s\n' "$DB_CRON_LINE" >>"$TMP_FILE"
printf '%s\n' "$FULL_CRON_LINE" >>"$TMP_FILE"
crontab "$TMP_FILE"

echo "Installed cron entries:"
echo "$DB_CRON_LINE"
echo "$FULL_CRON_LINE"
