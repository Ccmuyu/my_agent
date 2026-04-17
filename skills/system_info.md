---
name: sys_info
description: Check system resources like disk usage or memory.
parameters:
  type: object
  properties:
    command:
      type: string
      description: "The shell command to run (e.g., 'df -h', 'free -m')."
  required:
    - command
---

# System Info Guide
- Use standard Linux commands.
- Summarize the output if it's too long.

