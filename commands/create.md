---
description: Quickly create a Loom item — project, task, problem, goal, outcome, or task note
---

Help the user quickly create one or more Loom items.

Parse `$ARGUMENTS` to determine what to create. Examples:
- `task "Fix login bug" on project 1` → create a task
- `problem "API rate limiting" for project 2` → create a problem
- `goal "Ship v2 by March"` → create a goal
- `note "Completed auth refactor" on task 5` → create a task note
- `project "Backend Rewrite"` → create a project
- `outcome "Released MVP" on project 1` → create an outcome

1. Determine the item type from the arguments. If ambiguous, ask the user.
2. For tasks: infer `task_type` (general, feature, bugfix, chore, investigation) and `priority` (low, medium, high, urgent) from context, defaulting to `general` and `medium`. Ask the user to confirm before creating.
3. For problems: set status to `open` by default. If a project or task is mentioned, link it.
4. For goals: infer `goal_type` (short_term, career, values, requirement) from context, defaulting to `short_term`.
5. For outcomes: set status to `open` by default.
6. Call the appropriate `create_*` tool.
7. Show the created item to the user with its ID and key fields.

If `$ARGUMENTS` is empty, ask the user what they want to create.
