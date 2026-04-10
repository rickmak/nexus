#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
NEXUS_MOD="$ROOT/packages/nexus"
FC_VER="${NEXUS_TEST_FIRECRACKER_VERSION:-1.15.1}"
LIMA_VER="${NEXUS_TEST_LIMA_VERSION:-2.1.1}"

append_github_path() {
  local dir="$1"
  if [ -n "${GITHUB_PATH:-}" ] && [ -d "$dir" ]; then
    echo "$dir" >>"$GITHUB_PATH"
  fi
}

write_nexus_cli_env() {
  local out="$1"
  export NEXUS_CLI_PATH="$out"
  if [ -n "${GITHUB_ENV:-}" ]; then
    echo "NEXUS_CLI_PATH=$out" >>"$GITHUB_ENV"
  fi
}

write_env_sh() {
  local f="$1"
  {
    printf 'export NEXUS_CLI_PATH=%q\n' "$NEXUS_CLI_PATH"
    printf 'export PATH=%q\n' "$PATH"
  } >"$f"
  echo "sdk-runtime e2e: wrote $f"
}

build_nexus_cli() {
  local e2e_root="$1"
  local out="$e2e_root/nexus"
  echo "sdk-runtime e2e: building nexus CLI -> $out"
  (cd "$NEXUS_MOD" && go build -o "$out" ./cmd/nexus)
  write_nexus_cli_env "$out"
}

ensure_firecracker_linux() {
  local e2e_root="$1"
  if command -v firecracker >/dev/null 2>&1; then
    echo "sdk-runtime e2e: firecracker already on PATH"
    return 0
  fi
  local arch
  case "$(uname -m)" in
  x86_64) arch=x86_64 ;;
  aarch64 | arm64) arch=aarch64 ;;
  *)
    echo "sdk-runtime e2e: unsupported machine for Firecracker download: $(uname -m)" >&2
    return 1
    ;;
  esac
  local url="https://github.com/firecracker-microvm/firecracker/releases/download/v${FC_VER}/firecracker-v${FC_VER}-${arch}.tgz"
  local tgz="$e2e_root/firecracker.tgz"
  echo "sdk-runtime e2e: downloading Firecracker v${FC_VER} (${arch})..."
  curl -fsSL "$url" -o "$tgz"
  mkdir -p "$e2e_root/fc-extract" "$e2e_root/bin"
  tar -xzf "$tgz" -C "$e2e_root/fc-extract"
  local bin
  bin="$(find "$e2e_root/fc-extract" -type f \( -name 'firecracker-v*' -o -name firecracker \) ! -name '*.debug' 2>/dev/null | head -1 || true)"
  if [ -z "$bin" ] || [ ! -f "$bin" ]; then
    echo "sdk-runtime e2e: could not locate firecracker binary in archive" >&2
    return 1
  fi
  cp "$bin" "$e2e_root/bin/firecracker"
  chmod 755 "$e2e_root/bin/firecracker"
  export PATH="$e2e_root/bin:$PATH"
  append_github_path "$e2e_root/bin"
}

ensure_lima_darwin() {
  local e2e_root="$1"
  if command -v limactl >/dev/null 2>&1; then
    echo "sdk-runtime e2e: limactl already on PATH"
    return 0
  fi
  local darch
  case "$(uname -m)" in
  x86_64) darch=x86_64 ;;
  arm64) darch=arm64 ;;
  *)
    echo "sdk-runtime e2e: unsupported machine for Lima download: $(uname -m)" >&2
    return 1
    ;;
  esac
  local url="https://github.com/lima-vm/lima/releases/download/v${LIMA_VER}/lima-${LIMA_VER}-Darwin-${darch}.tar.gz"
  local tgz="$e2e_root/lima.tar.gz"
  echo "sdk-runtime e2e: downloading Lima v${LIMA_VER} (Darwin-${darch})..."
  curl -fsSL "$url" -o "$tgz"
  mkdir -p "$e2e_root/lima"
  tar -xzf "$tgz" -C "$e2e_root/lima"
  export PATH="$e2e_root/lima/bin:$PATH"
  append_github_path "$e2e_root/lima/bin"
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
  echo "sdk-runtime e2e: nexus init --project-root $abs"
  "$NEXUS_CLI_PATH" init --project-root "$abs" --force
  rm -rf "$seed"
}

main() {
  local e2e_root
  if [ -n "${E2E_ROOT:-}" ]; then
    e2e_root="$E2E_ROOT"
    mkdir -p "$e2e_root"
  else
    e2e_root="$(mktemp -d "${TMPDIR:-/tmp}/nexus-e2e-runtime.XXXXXX")"
  fi

  build_nexus_cli "$e2e_root"

  case "$(uname -s)" in
  Linux) ensure_firecracker_linux "$e2e_root" ;;
  Darwin) ensure_lima_darwin "$e2e_root" ;;
  *) echo "sdk-runtime e2e: skipping runtime downloads on $(uname -s)" ;;
  esac

  run_seed_nexus_init

  local env_out
  env_out="${GITHUB_WORKSPACE:-$ROOT}/.nexus-e2e-env.sh"
  write_env_sh "$env_out"

  echo "sdk-runtime e2e: sandbox prereqs ready (NEXUS_CLI_PATH=$NEXUS_CLI_PATH)"
}

main "$@"
