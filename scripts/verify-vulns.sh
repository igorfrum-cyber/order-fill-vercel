#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

if ! command -v govulncheck >/dev/null 2>&1; then
  echo "security: govulncheck is required; run go install golang.org/x/vuln/cmd/govulncheck@v1.8.0" >&2
  exit 1
fi

while IFS= read -r module; do
  echo "==> govulncheck ${module}"
  (
    cd "$module"
    GOWORK=off govulncheck ./...
  )
done < <(find backend -name go.mod -exec dirname {} \; | sort)
