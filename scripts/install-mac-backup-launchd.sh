#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLIST_TEMPLATE="${ROOT_DIR}/ops/com.langschool.pull-backups.plist"
PLIST_DEST="${HOME}/Library/LaunchAgents/com.langschool.pull-backups.plist"
BACKUP_DIR="${HOME}/Backups/langschool"

mkdir -p "${HOME}/Library/LaunchAgents" "$BACKUP_DIR"
cp "$PLIST_TEMPLATE" "$PLIST_DEST"

launchctl unload "$PLIST_DEST" >/dev/null 2>&1 || true
launchctl load "$PLIST_DEST"

echo "Installed launchd job at $PLIST_DEST"
echo "Logs will be written under $BACKUP_DIR"
