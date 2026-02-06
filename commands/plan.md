---
description: Interactively plan a project by creating a project, tasks, and goals in Loom
---

Help the user plan a project from scratch or extend an existing one.

1. Ask the user what they want to build or accomplish. If `$ARGUMENTS` is provided, use it as the starting description.
2. Check `list_projects` to see if a matching project already exists. If so, ask whether to extend it or create a new one.
3. Create the project with `create_project` (or use the existing one).
4. Break the work down into concrete tasks. Suggest task titles, types, and priorities. Ask the user to confirm or adjust before creating them.
5. For each confirmed task, call `create_task` with appropriate `task_type` and `priority`.
6. Ask if there are any goals for this project (short-term milestones, requirements, career goals). Create them with `create_goal`. If the user wants to link their goals to a superior's goals, set the `assignee` field to their manager's ID or email.
7. Ask if there are any known problems or risks. Create them with `create_problem`. If the problem affects multiple projects, use `link_problem_to_project` to create additional linkages.
8. If goals or problems apply to multiple projects, use `link_goal_to_project` or `link_problem_to_project` to add those relationships.
9. Finish by showing a summary of what was created: the project, all tasks with their priorities, and any goals or problems.

Be collaborative — suggest structure but let the user drive decisions. Don't create anything without confirmation.
