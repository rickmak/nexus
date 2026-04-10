#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
NEXUS_MOD="$ROOT/packages/nexus"

write_env_sh() {
  local f="$1"
  {
    printf 'export NEXUS_CLI_PATH=%q\n' "$NEXUS_CLI_PATH"
    printf 'export PATH=%q\n' "$PATH"
  } >"$f"
  echo "sdk-runtime e2e: wrote $f"
}

build_nexus_cli() {
  local out="${1:?}/nexus"
  echo "sdk-runtime e2e: building nexus CLI -> $out"
  (cd "$NEXUS_MOD" && go build -o "$out" ./cmd/nexus)
  export NEXUS_CLI_PATH="$out"
  if [ -n "${GITHUB_ENV:-}" ]; then
    echo "NEXUS_CLI_PATH=$out" >>"$GITHUB_ENV"
  fi
}

run_seed_nexus_init() {
  local seed
  seed="$(mktemp -d "${TMPDIR:-/tmp}/nexus-e2e-seed.XXXXXX")"
  mkdir -p "$seed/repo"
  (
    cd "$seed/repo"
    git init
    git config user.email "e2e@nexus.test"
    git config user.name "Nexus E2E"
    echo test >README.md
    git add .
    git commit -m init
  )
  local abs
  abs="$(cd "$seed/repo" && pwd)"
  echo "sdk-runtime e2e: nexus init --project-root $abs (runtime tools via preflight autoinstall when needed)"
  "$NEXUS_CLI_PATH" init --project-root "$abs" --force
  rm -rf "$seed"
}

main() {
  local e2e_root
  e2e_root="$(mktemp -d "${TMPDIR:-/tmp}/nexus-e2e-runtime.XXXXXX")"
  build_nexus_cli "$e2e_root"
  run_seed_nexus_init
  write_env_sh "${GITHUB_WORKSPACE:-$ROOT}/.nexus-e2e-env.sh"
  echo "sdk-runtime e2e: prereqs ready (NEXUS_CLI_PATH=$NEXUS_CLI_PATH)"
}

main "$@"
