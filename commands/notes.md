---
description: View or add notes on a task to record progress, decisions, and context
---

Help the user view or add task notes.

Parse `$ARGUMENTS` to determine the action. Examples:
- `show 5` or `view task 5` → show notes for task 5
- `add 5 "Completed the API integration"` → add a note to task 5
- `5` → show notes for task 5

1. If the user wants to view notes:
   a. Call `list_task_notes` with the task ID.
   b. Also call `get_task` to show the task title and status for context.
   c. Present the notes in chronological order with timestamps.
   d. If there are no notes, say so and suggest adding one.

2. If the user wants to add a note:
   a. Call `create_task_note` with the task ID and note content.
   b. Show the created note with its timestamp.

3. If `$ARGUMENTS` is empty, call `get_active_work_summary` to show in-progress tasks and ask which task the user wants to add notes to.

Notes should carry meaningful information. Remind the user if the note seems too vague (e.g. just "working on it").
