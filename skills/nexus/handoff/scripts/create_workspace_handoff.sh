#!/usr/bin/env bash
set -euo pipefail

path="."
objective=""
context=""
backend=""
handoff_command=""
nexus_bin=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --path) path="$2"; shift 2 ;;
    --objective) objective="$2"; shift 2 ;;
    --context) context="$2"; shift 2 ;;
    --backend) backend="$2"; shift 2 ;;
    --handoff-command) handoff_command="$2"; shift 2 ;;
    --nexus-bin) nexus_bin="$2"; shift 2 ;;
    *) echo "Unknown argument: $1" >&2; exit 2 ;;
  esac
done

if [[ -z "$objective" ]]; then
  echo "Required: --objective" >&2
  exit 2
fi

resolve_nexus_bin() {
  if [[ -n "$nexus_bin" && -x "$nexus_bin" ]]; then
    printf "%s" "$nexus_bin"
    return 0
  fi
  if [[ -n "${NEXUS_BIN:-}" && -x "${NEXUS_BIN}" ]]; then
    printf "%s" "${NEXUS_BIN}"
    return 0
  fi
  if command -v nexus >/dev/null 2>&1; then
    command -v nexus
    return 0
  fi
  if [[ -x "packages/nexus/nexus" ]]; then
    printf "%s" "packages/nexus/nexus"
    return 0
  fi
  if [[ -x "./nexus" ]]; then
    printf "%s" "./nexus"
    return 0
  fi
  echo "Unable to locate nexus binary. Set NEXUS_BIN or pass --nexus-bin." >&2
  return 1
}

nexus_cmd="$(resolve_nexus_bin)"
create_args=(workspace create --path "$path")
if [[ -n "$backend" ]]; then
  create_args+=(--backend "$backend")
fi
set +e
create_output="$("$nexus_cmd" "${create_args[@]}" 2>&1)"
create_status=$?
set -e
printf "%s\n" "$create_output"
if [[ $create_status -ne 0 ]]; then
  exit $create_status
fi

workspace_id="$(printf "%s\n" "$create_output" | sed -nE 's/.*\(id: ([^)]+)\).*/\1/p' | head -n1)"
worktree_path="$(printf "%s\n" "$create_output" | sed -nE 's/^local worktree:[[:space:]]+(.+)$/\1/p' | head -n1)"

if [[ -z "$workspace_id" ]]; then
  echo "Failed to parse workspace id from create output" >&2
  exit 1
fi

set +e
start_output="$("$nexus_cmd" workspace start "$workspace_id" 2>&1)"
start_status=$?
set -e
printf "%s\n" "$start_output"
if [[ $start_status -ne 0 ]]; then
  exit $start_status
fi
tmp_prompt_base="$(mktemp -t "nexus-handoff-${workspace_id}")"
prompt_file="${tmp_prompt_base}.md"
mv "$tmp_prompt_base" "$prompt_file"

cat > "$prompt_file" <<EOF
Continue implementation in workspace \`$workspace_id\` at \`$worktree_path\`.

## Objective

$objective

## Context

$context
EOF

echo "workspace_id=$workspace_id"
echo "worktree_path=$worktree_path"
echo "workspace_path=/workspace"
echo "prompt_file=$prompt_file"
if [[ -n "$worktree_path" ]]; then
  encoded_path="$(python3 - <<'PY' "$worktree_path"
import sys
from urllib.parse import quote
print(quote(sys.argv[1], safe="/"))
PY
)"
  echo "cursor_deeplink=cursor://file${encoded_path}"
  echo "vscode_deeplink=vscode://file${encoded_path}"
  echo "cursor_open_command=cursor \"$worktree_path\""
  echo "vscode_open_command=code \"$worktree_path\""
fi
echo "continue_last_session_command=cd \"$worktree_path\" && opencode --continue"
echo "continue_with_session_id_command=cd \"$worktree_path\" && opencode --session \"<session-id>\""
if [[ -z "$handoff_command" ]]; then
  handoff_command='cd "__WORKTREE__" && opencode "$(cat "__PROMPT_FILE__")"'
fi

resolved_command="${handoff_command//__WORKTREE__/$worktree_path}"
resolved_command="${resolved_command//__PROMPT_FILE__/$prompt_file}"
resolved_command="${resolved_command//__WORKSPACE_ID__/$workspace_id}"
echo "executing_handoff_command=$resolved_command"
if ! eval "$resolved_command"; then
  echo "handoff_command_failed=1"
  echo "manual_handoff_command=cd \"$worktree_path\" && opencode \"\$(cat \"$prompt_file\")\""
  exit 1
fi
