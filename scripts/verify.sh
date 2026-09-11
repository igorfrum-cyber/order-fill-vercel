#!/usr/bin/env bash
# Deterministic local pre-commit gate. CI calls the same component scripts and
# adds race detection, shuffled Go tests, and online vulnerability scanning.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

echo "==> toolchain"
"$root/scripts/verify-toolchain.sh"

echo "==> documentation"
node "$root/scripts/verify-docs.mjs"

echo "==> contracts"
"$root/scripts/verify-contracts.sh"

echo "==> frontend"
npm run test:load
npm run verify --prefix frontend

echo "==> backend v2"
while IFS= read -r module; do
  echo "==> ${module}"
  "$root/scripts/verify-go.sh" "$module"
done < <(find backend -name go.mod -exec dirname {} \; | sort)

echo "==> backend compose"
docker compose -f backend/deploy/docker-compose.yml config >/dev/null
docker compose -f deploy/docker-compose.yml config >/dev/null

echo "verify ok"
