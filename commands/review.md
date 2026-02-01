---
description: Review progress on a project — completed work, open items, and stale tasks
---

Give the user a progress review for a specific project or all projects.

If `$ARGUMENTS` is provided, treat it as a project name or ID and scope to that project. Otherwise review all projects.

1. Call `list_projects` and identify the target project(s).
2. For each project:
   a. Call `list_tasks` and group by status.
   b. Call `list_problems` and group by status.
   c. Call `list_outcomes` and group by status.
   d. Call `list_goals`.
3. Present the review with these sections:
   - **Completed**: Tasks and outcomes finished. Summarise what was accomplished.
   - **In Progress**: What is currently being worked on.
   - **Blocked**: Anything blocked, with linked problems if available.
   - **Pending**: Work that hasn't started yet, ordered by priority.
   - **Resolved Problems**: Problems that were overcome.
   - **Goals**: Progress toward stated goals.
4. Call out anything that looks stale — tasks that are `in_progress` with no recent task notes, or problems that have been `open` for a long time.
5. End with a brief recommendation: what should be worked on next based on priorities and blockers.

Keep the tone factual and concise. This is a status report, not a celebration.
