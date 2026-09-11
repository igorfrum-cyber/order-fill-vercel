#!/usr/bin/env bash
# Deterministic local pre-commit gate. CI calls the same component scripts and
# adds race detection, shuffled Go tests, and online vulnerability scanning.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

echo "==> toolchain"
"$root/scripts/verify-toolchain.sh"

echo "==> documentation"
node "$root/scripts/sync-docs.mjs" --write
node "$root/scripts/verify-docs.mjs"
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  if ! git diff --quiet -- backend/services/*/README.md; then
    echo "docs-sync: README обновлены из кода — добавьте их в коммит" >&2
    git diff --stat -- backend/services/*/README.md >&2
    exit 1
  fi
fi

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
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.prod.yml config >/dev/null

echo "verify ok"
