#!/bin/bash
set -euo pipefail
cd /Users/newman/magic/nexus
git status -sb > /Users/newman/magic/nexus/.agent-git-status.txt
git diff --stat HEAD >> /Users/newman/magic/nexus/.agent-git-status.txt
if git diff --quiet HEAD && git diff --cached --quiet; then
  echo "NO_CHANGES" >> /Users/newman/magic/nexus/.agent-git-status.txt
  exit 0
fi
git add -A
git commit -m "refactor: consolidate Lima guest into lima package

Backup: Swift runtime labels, git bind-mount perms on guest, limaguest→lima GuestDriver, daemon wiring, e2e assertion string."
echo "COMMITTED" >> /Users/newman/magic/nexus/.agent-git-status.txt
git log -1 --oneline >> /Users/newman/magic/nexus/.agent-git-status.txt
