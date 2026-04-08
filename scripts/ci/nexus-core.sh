#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../.."
pnpm install

task build:workspace-daemon
task build:workspace-sdk
if [[ -d packages/nexus-ui ]]; then
  task build:nexus-ui
fi

task lint:workspace-daemon
task lint:workspace-sdk

task test:workspace-daemon
task test:workspace-sdk
if [[ -d packages/nexus-ui ]]; then
  task test:nexus-ui
fi
