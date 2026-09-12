#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

node scripts/verify-contracts.mjs

if ! command -v buf >/dev/null 2>&1; then
  echo "contracts: buf is required (https://buf.build/docs/cli/installation/)" >&2
  exit 1
fi

buf lint backend/proto
