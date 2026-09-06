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

backend_go=$(go_mod_version backend/pkg/go.mod)
[[ -n $backend_go ]] || fail "could not read go version from backend/pkg/go.mod"
work_go=$(go_mod_version backend/go.work)
[[ $work_go == "$backend_go" ]] || fail "backend/go.work go=$work_go must match backend/pkg/go.mod go=$backend_go"
work_toolchain=$(awk '/^toolchain / { sub(/^go/, "", $2); print $2; exit }' backend/go.work)
[[ -z $work_toolchain || $work_toolchain == "$backend_go" ]] ||
  fail "backend/go.work toolchain go${work_toolchain} must match go.mod ${backend_go}"
while IFS= read -r modfile; do
  module_go=$(go_mod_version "$modfile")
  [[ $module_go == "$backend_go" ]] || fail "go.mod versions differ: backend/pkg=$backend_go $modfile=$module_go"
done < <(find backend -name go.mod | sort)

go_mm=$(go_major_minor "$backend_go")
for dockerfile in backend/services/*/Dockerfile; do
  grep -Eq "^FROM (mirror\.gcr\.io/library/)?golang:${go_mm}([.-]|$)" "$dockerfile" ||
    fail "$dockerfile must use golang:${go_mm} to match go.mod ${backend_go}"
done

local_go=$(cd "$root/backend/pkg" && go env GOVERSION | sed 's/^go//')
local_go=${local_go%%-*}
version_ge "$local_go" "$backend_go" ||
  fail "local Go ${local_go} is older than go.mod ${backend_go}"

root_node=$(read_json_engines_major package.json)
frontend_node=$(read_json_engines_major frontend/package.json)
[[ $root_node == "$frontend_node" ]] ||
  fail "engines.node major differs: root=$root_node frontend=$frontend_node"

grep -Eq "^FROM (mirror\.gcr\.io/library/)?node:${frontend_node}([.-]|$)" frontend/Dockerfile ||
  fail "frontend/Dockerfile must use node:${frontend_node}"

grep -Eq "node-version:[[:space:]]*\"${frontend_node}\"" .github/workflows/verify.yml ||
  fail "CI node-version must be \"${frontend_node}\""

grep -Fq "go-version-file: backend/pkg/go.mod" .github/workflows/verify.yml ||
  fail "CI must pin Go from backend/pkg/go.mod"

local_node_major=$(node -p "process.versions.node.split('.')[0]")
((local_node_major >= frontend_node)) ||
  fail "local Node ${local_node_major} is older than engines.node ${frontend_node}"

echo "toolchain ok: node>=${frontend_node} (local $(node -v)), go ${backend_go} (local go${local_go})"
