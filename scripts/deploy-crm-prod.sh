#!/usr/bin/env bash
set -euo pipefail

cd /root/work/multica-crm
ENV_FILE=/root/.multica/server/.env

if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: missing $ENV_FILE" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

: "${MULTICA_WEB_IMAGE:=ghcr.io/flyinsz/multica-web}"
: "${MULTICA_BACKEND_IMAGE:=ghcr.io/flyinsz/multica-backend}"
: "${MULTICA_IMAGE_TAG:=crm-latest}"
: "${POSTGRES_PORT:=15432}"
: "${FRONTEND_PORT:=13010}"
: "${PORT:=8080}"

if [[ -z "${RESEND_API_KEY:-}" ]]; then
  echo "ERROR: RESEND_API_KEY missing in $ENV_FILE; refusing deploy because login email codes would not send." >&2
  exit 2
fi

if [[ ${#RESEND_API_KEY} -lt 20 ]]; then
  echo "ERROR: RESEND_API_KEY looks invalid (too short); refusing deploy." >&2
  exit 2
fi

if [[ -z "${FRONTEND_ORIGIN:-}" ]]; then
  export FRONTEND_ORIGIN="http://127.0.0.1:${FRONTEND_PORT}"
fi
if [[ -z "${MULTICA_APP_URL:-}" ]]; then
  export MULTICA_APP_URL="$FRONTEND_ORIGIN"
fi

export MULTICA_WEB_IMAGE MULTICA_BACKEND_IMAGE MULTICA_IMAGE_TAG POSTGRES_PORT FRONTEND_PORT PORT RESEND_API_KEY RESEND_FROM_EMAIL FRONTEND_ORIGIN MULTICA_APP_URL COOKIE_DOMAIN

docker compose -p multica -f docker-compose.selfhost.yml pull frontend backend
docker compose -p multica -f docker-compose.selfhost.yml up -d --no-deps --force-recreate backend frontend

echo "Deployed $MULTICA_BACKEND_IMAGE:$MULTICA_IMAGE_TAG and $MULTICA_WEB_IMAGE:$MULTICA_IMAGE_TAG"
