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
    --include='app-*.sqlite' \
    --include='full-*.tar.gz' \
    --exclude='*' \
    "${SOURCE_HOST}:${SOURCE_DIR}" \
    "${DEST_DIR}/"
  shopt -s nullglob
  app_files=( "${DEST_DIR}"/app-*.sqlite )
  if ((${#app_files[@]} > 30)); then
    ls -1t "${app_files[@]}" | tail -n +31 | while IFS= read -r f; do rm -f -- "$f"; done
  fi
  full_files=( "${DEST_DIR}"/full-*.tar.gz )
  if ((${#full_files[@]} > 8)); then
    ls -1t "${full_files[@]}" | tail -n +9 | while IFS= read -r f; do rm -f -- "$f"; done
  fi
  printf '[%s] Backup pull finished\n' "$(date '+%Y-%m-%d %H:%M:%S')"
} | tee -a "$LOG_FILE"
