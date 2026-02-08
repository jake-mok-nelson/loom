---
description: Show all active and in-progress work across projects using a single efficient summary call
---

Give the user a quick snapshot of everything currently active or in progress.

1. Call `get_active_work_summary` to retrieve all active projects, pending/in-progress tasks, open/in-progress problems, and open/in-progress outcomes in a single call.
2. If `$ARGUMENTS` is provided, treat it as a project name or ID and filter the results to only show items related to that project.
3. Present the results grouped by project:
   - **Active Projects**: List each active project with its description.
   - **Tasks**: Show pending and in-progress tasks with their priority and type.
   - **Problems**: Show open and in-progress problems.
   - **Outcomes**: Show open and in-progress outcomes.
4. Highlight high-priority and urgent tasks — these need attention first.
5. If there are no active items, say so clearly and suggest the user create a project or check the `/review` command for completed work.

Keep the output concise. Use tables or short bullet lists. This is a quick glance, not a deep review.
