---
description: Update a Loom item — change status, priority, description, or other fields
---

Help the user update an existing Loom item.

Parse `$ARGUMENTS` to determine what to update. Examples:
- `task 5 status in_progress` → update task status
- `task 3 priority high` → update task priority
- `problem 2 status resolved` → resolve a problem
- `project 1 description "New description"` → update project description
- `goal 4 assignee "jane@example.com"` → assign a goal

1. Determine the item type and ID from the arguments. If ambiguous, ask the user.
2. If the user provides just an ID without specifying the type, call `get_task`, `get_problem`, `get_project`, etc. to find the item.
3. Call the appropriate `update_*` tool with the specified changes.
4. Show the updated item to the user.
5. If the user is changing a task to `blocked`, ask if they want to create a problem to track the blocker.
6. If the user is resolving a problem linked to a blocked task, ask if they want to unblock the task too.

If `$ARGUMENTS` is empty, ask the user what they want to update.
