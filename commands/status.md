---
description: Show a dashboard of all Loom projects with task, problem, goal, and outcome summaries
---

Give the user a concise dashboard of everything in Loom.

1. Call `list_projects` to get all projects.
2. For each project, call `list_tasks`, `list_problems`, `list_outcomes`, and `list_goals` filtered by `project_id`.
3. Also call `get_project_problems` and `get_project_goals` to find items linked via the many-to-many junction tables.
4. Present a summary table per project showing:
   - Total tasks and breakdown by status (pending, in_progress, completed, blocked)
   - Open problems count (including those linked via junction table)
   - Open/in-progress goals count (including those linked via junction table)
   - Open/in-progress outcomes count
5. Highlight any blocked tasks or open problems — these need attention.
6. If there are goals or problems assigned to a specific person, group them by assignee.
7. If there are no projects yet, tell the user and suggest creating one.

Keep the output compact. Use tables or short bullet lists, not verbose paragraphs.
