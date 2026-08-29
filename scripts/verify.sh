#!/usr/bin/env bash
# Local precommit / CI gate. Keep this file in lockstep with
# .github/workflows/verify.yml — if CI grows a step, add it here too.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

echo "==> toolchain"
"$root/scripts/verify-toolchain.sh"

echo "==> frontend"
npm run verify --prefix frontend

echo "==> api-service"
"$root/scripts/verify-go.sh" services/api-service

echo "==> document-service"
"$root/scripts/verify-go.sh" services/document-service

echo "verify ok"
