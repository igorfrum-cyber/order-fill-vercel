#!/usr/bin/env bash
# Fast-forward origin/dev and rebuild the running Compose stack.
# Keeps .env and named volumes. Does not wipe data.
set -euo pipefail

root=${DEPLOY_DIR:-"$HOME/srv/order-fill-vercel"}
cd "$root"

if [[ ! -d .git ]]; then
  echo "deploy: $root is not a git checkout" >&2
  exit 1
fi
if [[ ! -f .env ]]; then
  echo "deploy: missing $root/.env" >&2
  exit 1
fi

# shellcheck source=prod-env.sh
source "$root/scripts/prod-env.sh"

git fetch origin
git checkout dev
git pull --ff-only origin dev

compose=(docker compose --env-file .env -f deploy/docker-compose.yml)
if [[ "$(read_dotenv .env APP_ENV)" == "production" ]]; then
  check_production_env .env
  bash "$root/scripts/gen-internal-tls.sh"
  compose+=(-f deploy/docker-compose.prod.yml)
fi
if [[ -f deploy/docker-compose.https.yml ]]; then
  compose+=(-f deploy/docker-compose.https.yml)
fi

"${compose[@]}" up -d --build
"${compose[@]}" ps
echo "deploy ok"
