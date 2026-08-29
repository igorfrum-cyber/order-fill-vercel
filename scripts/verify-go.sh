#!/usr/bin/env bash
# Lint, security, modules, build, and tests for one Go module.
set -euo pipefail

GOLANGCI_LINT_VERSION=v2.12.2
GOSEC_VERSION=v2.22.10

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: $0 <go-module-dir> [all|lint]" >&2
  exit 2
fi

dir=$1
mode=${2:-all}
if [[ $mode != all && $mode != lint ]]; then
  echo "usage: $0 <go-module-dir> [all|lint]" >&2
  exit 2
fi
if [[ ! -d $dir ]]; then
  echo "missing Go module: $dir" >&2
  exit 1
fi

export PATH="$(go env GOPATH)/bin:${PATH}"

root=$(cd "$(dirname "$0")/.." && pwd)
export GOLANGCI_LINT_CACHE="${root}/.cache/golangci-lint"
mkdir -p "$GOLANGCI_LINT_CACHE"
cd "$root/$dir"

ensure_golangci_lint() {
  if command -v golangci-lint >/dev/null 2>&1; then
    return
  fi
  echo "installing golangci-lint ${GOLANGCI_LINT_VERSION}"
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh |
    sh -s -- -b "$(go env GOPATH)/bin" "$GOLANGCI_LINT_VERSION"
}

ensure_gosec() {
  if command -v gosec >/dev/null 2>&1; then
    return
  fi
  echo "installing gosec ${GOSEC_VERSION}"
  go install "github.com/securego/gosec/v2/cmd/gosec@${GOSEC_VERSION}"
}

unformatted=$(gofmt -l .)
if [[ -n $unformatted ]]; then
  echo "gofmt needed in $dir:" >&2
  echo "$unformatted" >&2
  exit 1
fi

go vet ./...

ensure_golangci_lint
golangci-lint run --timeout=5m

ensure_gosec
gosec -quiet -exclude-generated ./...

mod_before=$(mktemp)
sum_before=$(mktemp)
trap 'rm -f "$mod_before" "$sum_before"' EXIT
cp go.mod "$mod_before"
cp go.sum "$sum_before"
go mod tidy
if ! cmp -s go.mod "$mod_before" || ! cmp -s go.sum "$sum_before"; then
  echo "go mod tidy changed go.mod / go.sum in $dir — commit the result" >&2
  diff -u "$mod_before" go.mod || true
  diff -u "$sum_before" go.sum || true
  exit 1
fi

if [[ $mode == lint ]]; then
  exit 0
fi

go build ./...
go test ./...
