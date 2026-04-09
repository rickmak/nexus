#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "Skipping pty-lxc-managed-e2e (requires macOS)"
  exit 0
fi

if ! command -v limactl >/dev/null 2>&1; then
  brew install lima
fi

cd "$(dirname "$0")/../../packages/nexus"

PROJECT_ROOT="$(mktemp -d)"
go run ./cmd/nexus init --project-root "${PROJECT_ROOT}" --force

PORT=8095
TOKEN=ci-token
WS_DIR="$(mktemp -d)"
DAEMON_LOG="/tmp/nexus-daemon-lxc-managed-local.log"
SMOKE_LOG="pty-remote-smoke-lxc-managed-local.log"

NEXUS_RUNTIME_LXC_BOOTSTRAP_DOCKER=0 \
go run ./cmd/daemon --port "${PORT}" --workspace-dir "${WS_DIR}" --token "${TOKEN}" >"${DAEMON_LOG}" 2>&1 &
DAEMON_PID=$!

cleanup() {
  if kill -0 "${DAEMON_PID}" >/dev/null 2>&1; then
    kill "${DAEMON_PID}" >/dev/null 2>&1 || true
    wait "${DAEMON_PID}" 2>/dev/null || true
  fi
  rm -rf "${WS_DIR}" "${PROJECT_ROOT}"
}
trap cleanup EXIT

for _ in $(seq 1 50); do
  if curl -fsS "http://127.0.0.1:${PORT}/healthz" >/dev/null; then
    break
  fi
  sleep 1
done
curl -fsS "http://127.0.0.1:${PORT}/healthz" >/dev/null

TEST_REPO="$(mktemp -d)"
mkdir -p "${TEST_REPO}/.nexus"
cat > "${TEST_REPO}/.nexus/workspace.json" <<'JSON'
{
  "version": 1,
  "runtime": {
    "required": ["lxc"],
    "selection": "prefer-first"
  }
}
JSON
git -C "${TEST_REPO}" init -b main
git -C "${TEST_REPO}" config user.email ci@nexus.local
git -C "${TEST_REPO}" config user.name "Nexus CI"
touch "${TEST_REPO}/README.md"
git -C "${TEST_REPO}" add .
git -C "${TEST_REPO}" commit -m "init" >/dev/null

CREATE_OUTPUT=$(NEXUS_DAEMON_PORT="${PORT}" \
  NEXUS_DAEMON_TOKEN="${TOKEN}" \
  go run ./cmd/nexus workspace create \
    --repo "${TEST_REPO}" \
    --name "ci-lxc-local-${RANDOM}" \
    --profile "codex" \
    --backend "lxc")
echo "${CREATE_OUTPUT}"
WORKSPACE_ID=$(printf '%s\n' "${CREATE_OUTPUT}" | sed -nE 's/.*\(id: ([^)]+)\).*/\1/p' | tail -n 1)
if [[ -z "${WORKSPACE_ID}" ]]; then
  echo "failed to parse workspace id from create output" >&2
  exit 1
fi

NEXUS_DAEMON_WS="ws://127.0.0.1:${PORT}" \
NEXUS_DAEMON_TOKEN="${TOKEN}" \
NEXUS_PTY_SMOKE_LOG="${SMOKE_LOG}" \
NEXUS_PTY_TIMEOUT_MS=120000 \
node --experimental-websocket scripts/pty-remote-smoke.js "${WORKSPACE_ID}"
