#!/usr/bin/env bash
set -euo pipefail

SOURCE_HOST="${SOURCE_HOST:-home-java}"
SOURCE_DIR="${SOURCE_DIR:-/home/ilya/langschool-data/backups/}"
DEST_DIR="${DEST_DIR:-$HOME/Backups/langschool}"
LOG_FILE="${LOG_FILE:-$DEST_DIR/pull.log}"

mkdir -p "$DEST_DIR"

{
  printf '[%s] Starting backup pull from %s:%s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$SOURCE_HOST" "$SOURCE_DIR"
  rsync -av --prune-empty-dirs \
    --include='full-*.tar.gz' \
    --exclude='*' \
    "${SOURCE_HOST}:${SOURCE_DIR}" \
    "${DEST_DIR}/"
  printf '[%s] Backup pull finished\n' "$(date '+%Y-%m-%d %H:%M:%S')"
} | tee -a "$LOG_FILE"
