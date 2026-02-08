---
description: Mark tasks as completed, problems as resolved, or outcomes as completed
---

Help the user quickly resolve or complete items.

Parse `$ARGUMENTS` to determine what to resolve. Examples:
- `task 5` → mark task 5 as completed
- `problem 3` → mark problem 3 as resolved
- `outcome 2` → mark outcome 2 as completed
- `5` → ask the user which type of item to resolve

1. Determine the item type and ID from the arguments.
2. Call the appropriate `get_*` tool to verify the item exists and show its current state.
3. For tasks:
   a. Call `update_task` with `status=completed`.
   b. Ask if the user wants to add a final task note summarising what was done.
4. For problems:
   a. Call `update_problem` with `status=resolved`.
   b. Check if the problem is linked to a blocked task. If so, ask if the task should be unblocked (set to `in_progress`).
5. For outcomes:
   a. Call `update_outcome` with `status=completed`.
6. Show the updated item to the user.

If `$ARGUMENTS` is empty, call `get_active_work_summary` to show open items and ask what the user wants to resolve.
