#!/usr/bin/env bash
# Fail if Node/Go pins drift across package.json, go.mod, Dockerfiles, and CI.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

fail() {
  printf 'toolchain: %s\n' "$*" >&2
  exit 1
}

read_json_engines_major() {
  node -e '
    const fs = require("fs");
    const file = process.argv[1];
    const engines = JSON.parse(fs.readFileSync(file, "utf8")).engines || {};
    const spec = String(engines.node || "");
    const match = spec.match(/(\d+)/);
    if (!match) {
      console.error("missing engines.node in " + file);
      process.exit(1);
    }
    process.stdout.write(match[1]);
  ' "$1"
}

go_mod_version() {
  awk '/^go / { print $2; exit }' "$1"
}

go_major_minor() {
  awk -F. '{ print $1 "." $2 }' <<<"$1"
}

version_ge() {
  [[ $1 == "$2" || $(printf '%s\n' "$1" "$2" | sort -V | tail -n1) == "$1" ]]
}

api_go=$(go_mod_version services/api-service/go.mod)
doc_go=$(go_mod_version services/document-service/go.mod)
[[ -n $api_go && -n $doc_go ]] || fail "could not read go version from go.mod"
[[ $api_go == "$doc_go" ]] || fail "go.mod versions differ: api-service=$api_go document-service=$doc_go"

go_mm=$(go_major_minor "$api_go")
for dockerfile in services/api-service/Dockerfile services/document-service/Dockerfile; do
  grep -Eq "^FROM golang:${go_mm}([.-]|$)" "$dockerfile" ||
    fail "$dockerfile must use golang:${go_mm} to match go.mod ${api_go}"
done

local_go=$(go env GOVERSION | sed 's/^go//')
local_go=${local_go%%-*}
version_ge "$local_go" "$api_go" ||
  fail "local Go ${local_go} is older than go.mod ${api_go}"

root_node=$(read_json_engines_major package.json)
frontend_node=$(read_json_engines_major frontend/package.json)
[[ $root_node == "$frontend_node" ]] ||
  fail "engines.node major differs: root=$root_node frontend=$frontend_node"

grep -Eq "^FROM node:${frontend_node}([.-]|$)" frontend/Dockerfile ||
  fail "frontend/Dockerfile must use node:${frontend_node}"

grep -Eq "node-version:[[:space:]]*\"${frontend_node}\"" .github/workflows/verify.yml ||
  fail "CI node-version must be \"${frontend_node}\""

grep -Fq "go-version-file: services/api-service/go.mod" .github/workflows/verify.yml ||
  fail "CI must pin Go from services/api-service/go.mod"

local_node_major=$(node -p "process.versions.node.split('.')[0]")
((local_node_major >= frontend_node)) ||
  fail "local Node ${local_node_major} is older than engines.node ${frontend_node}"

echo "toolchain ok: node>=${frontend_node} (local $(node -v)), go ${api_go} (local go${local_go})"
