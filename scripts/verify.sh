#!/usr/bin/env bash
# Local precommit / CI gate. Keep this file in lockstep with
# .github/workflows/verify.yml — if CI grows a step, add it here too.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

echo "==> toolchain"
"$root/scripts/verify-toolchain.sh"

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

echo "verify ok"
