---
name: git_ops
description: Execute safe git commands to check status, logs, or diffs.
parameters:
  type: object
  properties:
    command:
      type: string
      description: "The git command to execute (e.g., 'status', 'log -n 5'). Do NOT use push or reset."
  required:
    - command
---

# Git Operations Guide
- Use `git --no-pager` to avoid interactive prompts.
- Only use read-only commands like status, log, diff, branch.
- If the user asks for history, use `git log`.

