#!/usr/bin/env bash
# One-time self-hosted runner on the laptop. Do not expose this on a public
# fork-PR workflow: the deploy job only runs on push to dev.
#
# 1. GitHub → repo Settings → Actions → Runners → New self-hosted runner
# 2. Copy the registration token (not a PAT)
# 3. RUNNER_TOKEN=... bash scripts/install-github-runner.sh
set -euo pipefail

token=${RUNNER_TOKEN:-${1:-}}
if [[ -z "$token" ]]; then
  echo "usage: RUNNER_TOKEN=<from GitHub> bash scripts/install-github-runner.sh" >&2
  exit 1
fi

dir=${RUNNER_DIR:-"$HOME/actions-runner"}
url=${GITHUB_REPO_URL:-https://github.com/igorfrum-cyber/order-fill-vercel}
mkdir -p "$dir"
cd "$dir"

if [[ ! -x ./config.sh ]]; then
  version=${RUNNER_VERSION:-2.337.0}
  tarball="actions-runner-linux-x64-${version}.tar.gz"
  curl -fsSL -o "$tarball" \
    "https://github.com/actions/runner/releases/download/v${version}/${tarball}"
  tar xzf "$tarball"
  rm -f "$tarball"
fi

./config.sh --unattended \
  --url "$url" \
  --token "$token" \
  --name "${RUNNER_NAME:-order-fill-laptop}" \
  --labels order-fill \
  --work _work \
  --replace

if [[ "$(id -u)" -eq 0 ]]; then
  ./svc.sh install
  ./svc.sh start
else
  echo "Registering systemd service needs sudo:"
  echo "  cd $dir && sudo ./svc.sh install && sudo ./svc.sh start"
fi
