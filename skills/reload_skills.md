---
name: reload_skills
description: Reload all skill definitions from the ./skills directory. Use this when the user adds, modifies, or deletes .md files in the skills folder and wants the agent to recognize the changes immediately.
parameters:
  type: object
  properties:
    reason:
      type: string
      description: "Optional reason for reloading (e.g., 'User added a new git skill')."
  required: []
---

# Reload Skills Guide
- This tool does not execute shell commands.
- It triggers the internal mechanism to re-read all .md files.
- After calling this, the agent will have access to the latest skills.
- Confirm to the user that skills have been reloaded and list the available ones if possible.

