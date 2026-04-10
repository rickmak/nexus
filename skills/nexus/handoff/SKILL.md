---

## name: handoff

description: This skill should be used to create and start a Nexus workspace and hand off a focused implementation prompt to an agent in that workspace.

# Handoff

## Activation Priority

If there is even a 1% chance a task should run in an isolated Nexus workspace, use this skill.

After a git worktree is created for implementation, immediately urge creation of a Nexus workspace and run this handoff flow instead of continuing directly in the host worktree.

## Workflow

1. Run `scripts/create_workspace_handoff.sh`.
2. Confirm workspace ID and worktree path.
3. Confirm prompt file and editor deep links are created.
4. Confirm automatic handoff command execution.
5. Confirm session continuation commands are printed (`opencode --continue` and `opencode --session` forms).

## Command

```bash
bash skills/nexus/handoff/scripts/create_workspace_handoff.sh \
  --path "<optional: local-repo-path, default .>" \
  --objective "<implementation-objective>" \
  --backend "<optional: firecracker|seatbelt>" \
  --handoff-command '<optional: custom command; defaults to opencode launch>'
```

