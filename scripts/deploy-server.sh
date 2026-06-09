#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: deploy-server.sh [--dry-run]

Environment overrides:
  DEPLOY_HOST            SSH host to deploy to (default: home-java)
  DEPLOY_USER            SSH user (default: ilya)
  DEPLOY_PATH            Target directory on the server (default: /home/ilya/langschool)
  DEPLOY_HEALTHCHECK_URL Health check URL on the server (default: http://127.0.0.1:8082/healthz)
  DEPLOY_HEALTHCHECK_ATTEMPTS Number of health check attempts (default: 30)
  DEPLOY_HEALTHCHECK_DELAY Delay between attempts in seconds (default: 2)
EOF
}

DRY_RUN=0
if [[ $# -gt 1 ]]; then
  usage >&2
  exit 2
fi
if [[ $# -eq 1 ]]; then
  case "$1" in
    --dry-run)
      DRY_RUN=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage >&2
      exit 2
      ;;
  esac
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_HOST="${DEPLOY_HOST:-home-java}"
DEPLOY_USER="${DEPLOY_USER:-ilya}"
DEPLOY_PATH="${DEPLOY_PATH:-/home/ilya/langschool}"
DEPLOY_HEALTHCHECK_URL="${DEPLOY_HEALTHCHECK_URL:-http://127.0.0.1:8082/healthz}"
DEPLOY_HEALTHCHECK_ATTEMPTS="${DEPLOY_HEALTHCHECK_ATTEMPTS:-30}"
DEPLOY_HEALTHCHECK_DELAY="${DEPLOY_HEALTHCHECK_DELAY:-2}"
SSH_TARGET="${DEPLOY_USER}@${DEPLOY_HOST}"
SSH_DIR="${HOME}/.ssh"
KNOWN_HOSTS="${SSH_DIR}/known_hosts"
SSH_RESOLVED_HOST="$(ssh -G "$DEPLOY_HOST" | awk '/^hostname / { print $2; exit }')"
SSH_RESOLVED_PORT="$(ssh -G "$DEPLOY_HOST" | awk '/^port / { print $2; exit }')"
SSH_OPTIONS=(
  -o BatchMode=yes
  -o StrictHostKeyChecking=yes
  -o UserKnownHostsFile="$KNOWN_HOSTS"
)

mkdir -p "$SSH_DIR"
chmod 700 "$SSH_DIR"
touch "$KNOWN_HOSTS"
chmod 600 "$KNOWN_HOSTS"

# Trust-on-first-use is acceptable here because the connection already travels
# inside the tailnet and the first deploy needs to bootstrap known_hosts.
if [[ -n "$SSH_RESOLVED_HOST" ]] && ! ssh-keygen -F "$SSH_RESOLVED_HOST" -f "$KNOWN_HOSTS" >/dev/null; then
  ssh-keyscan -H -p "${SSH_RESOLVED_PORT:-22}" "$SSH_RESOLVED_HOST" >>"$KNOWN_HOSTS" 2>/dev/null
fi

ssh "${SSH_OPTIONS[@]}" "$SSH_TARGET" "mkdir -p '$DEPLOY_PATH'"

RSYNC_ARGS=(
  -az
  --delete
  --exclude=.env
  --exclude=.env.bak-*
  --exclude=.git/
  --exclude=.github/
  --exclude=frontend/node_modules/
  --exclude=frontend/dist/
  --exclude=build/
  --exclude=coverage.out
  --exclude=coverage_total.out
  --exclude=.DS_Store
)

if [[ "$DRY_RUN" -eq 1 ]]; then
  RSYNC_ARGS+=(--dry-run --itemize-changes)
  echo "Running rsync dry-run to ${SSH_TARGET}:${DEPLOY_PATH}/"
else
  echo "Syncing project to ${SSH_TARGET}:${DEPLOY_PATH}/"
fi

rsync -e "ssh -o BatchMode=yes -o StrictHostKeyChecking=yes -o UserKnownHostsFile=$KNOWN_HOSTS" \
  "${RSYNC_ARGS[@]}" \
  "${ROOT_DIR}/" \
  "${SSH_TARGET}:${DEPLOY_PATH}/"

if [[ "$DRY_RUN" -eq 1 ]]; then
  echo "Dry-run completed; remote deploy steps were skipped."
  exit 0
fi

echo "Starting remote deploy on ${SSH_TARGET}"
ssh "${SSH_OPTIONS[@]}" "$SSH_TARGET" \
  DEPLOY_PATH="$DEPLOY_PATH" \
  DEPLOY_HEALTHCHECK_URL="$DEPLOY_HEALTHCHECK_URL" \
  DEPLOY_HEALTHCHECK_ATTEMPTS="$DEPLOY_HEALTHCHECK_ATTEMPTS" \
  DEPLOY_HEALTHCHECK_DELAY="$DEPLOY_HEALTHCHECK_DELAY" \
  'bash -s' <<'EOF'
set -euo pipefail

cd "$DEPLOY_PATH"
docker compose up -d --build

for ((attempt = 1; attempt <= DEPLOY_HEALTHCHECK_ATTEMPTS; attempt++)); do
  if curl --fail --silent --show-error "$DEPLOY_HEALTHCHECK_URL" >/dev/null; then
    echo "Deploy health check passed: $DEPLOY_HEALTHCHECK_URL"
    tailscale funnel status || true
    exit 0
  fi
  sleep "$DEPLOY_HEALTHCHECK_DELAY"
done

echo "Deploy health check did not pass in time: $DEPLOY_HEALTHCHECK_URL" >&2
exit 1
EOF
