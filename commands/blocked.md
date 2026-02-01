---
description: Show all blocked tasks, problems, and outcomes, optionally scoped to a project
---

Show the user everything that is currently blocked or unresolved.

If `$ARGUMENTS` is provided, treat it as a project name or ID and scope the results to that project. Otherwise show blocked items across all projects.

1. Call `list_tasks` with `status=blocked`. If scoped to a project, include `project_id`.
2. Call `list_problems` with `status=open` and `status=blocked` (two calls). If scoped, include `project_id`.
3. Call `list_outcomes` with `status=blocked`. If scoped, include `project_id`.
4. For each blocked task, call `list_task_notes` to show recent context on why it is blocked.
5. Present the results grouped by project, then by type (tasks, problems, outcomes).
6. If nothing is blocked, say so clearly.

The goal is to give the user a quick view of what needs unblocking so they can take action.
