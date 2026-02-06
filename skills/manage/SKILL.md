---
description: Project and task management with Loom. Use when the user is working on a project, planning work, tracking tasks, or when you need to record progress, decisions, and blockers.
---

# Loom Project & Task Management

You have access to a Loom MCP server that stores projects, tasks, problems, goals, outcomes, and task notes in a local SQLite database. Use it proactively to keep work organised as you go — don't wait for the user to ask you to update Loom.

## Data Model

### Projects
The top-level container. Every task and outcome belongs to a project. Problems and goals can optionally link to a project.

Fields: `name` (required), `description`, `external_link` (e.g. a Jira or GitHub URL).

### Tasks
Units of work inside a project.

Fields: `project_id` (required), `title` (required), `description`, `task_type` (general | chore | investigation | feature | bugfix), `status` (pending | in_progress | completed | blocked), `priority` (low | medium | high | urgent), `external_link`.

### Task Notes
Timestamped notes attached to a task. Use these to record progress, decisions, context, and findings as work happens.

Fields: `task_id` (required), `note` (required).

### Problems
Issues or blockers, optionally linked to a project and/or task. A problem linked to a task must belong to the same project. Problems support many-to-many relationships with projects via `link_problem_to_project`.

Fields: `title` (required), `project_id`, `task_id`, `description`, `status` (open | in_progress | resolved | blocked), `assignee` (optional - use to link problems to a person, e.g., a superior's ID or email).

### Goals
Objectives optionally linked to a project and/or task. A goal linked to a task must belong to the same project. Goals support many-to-many relationships with projects via `link_goal_to_project`, allowing you to share goals across projects or link your goals to those of your superiors.

Fields: `title` (required), `project_id`, `task_id`, `description`, `goal_type` (short_term | career | values | requirement), `assignee` (optional - use to link goals to a person, e.g., your manager or superior).

### Outcomes
Results or milestones linked to a project (required) and optionally a task. A task must belong to the same project.

Fields: `project_id` (required), `title` (required), `task_id`, `description`, `status` (open | in_progress | completed | blocked).

## Multiple Project Linkages

Goals and problems can be linked to multiple projects using junction tables:
- Use `link_goal_to_project` / `unlink_goal_from_project` to manage goal-project relationships
- Use `link_problem_to_project` / `unlink_problem_from_project` to manage problem-project relationships
- Use `get_goal_projects` / `get_project_goals` to query goal-project links
- Use `get_problem_projects` / `get_project_problems` to query problem-project links

This allows you to:
- Share problems across multiple projects
- Align your goals with those of your superiors by setting the `assignee` field
- Track cross-project dependencies and blockers

## Assignee Field

Both goals and problems now support an optional `assignee` field:
- Use to assign responsibility for a problem or goal
- Useful for linking your goals to a superior's goals (set assignee to the superior's ID/email)
- Filter by assignee using `list_goals` or `list_problems` with the `assignee` parameter

## Proactive Behaviour

Follow these conventions automatically throughout the conversation. Do not wait for the user to tell you to update Loom.

### Starting work
- When beginning work on a task, call `update_task` to set its status to `in_progress`.
- If no task exists for the work you are about to do, call `create_task` first, then mark it `in_progress`.
- If no project exists yet, ask the user for a project name or suggest one, then create it.

### During work
- **Add task notes** as you go to record meaningful progress, decisions made, approaches tried, and important context. A note like "Refactored auth middleware to use JWT validation" is useful; "Working on it" is not.
- When you discover new work that is out of scope for the current task, **create a new task** for it rather than expanding the current one.
- When you encounter a blocker or issue, **create a problem** linked to the current task and project and set the task status to `blocked`.
- When a problem is resolved, update the problem status to `resolved` and unblock the task (set it back to `in_progress`).

### Completing work
- When a task is finished, call `update_task` to set its status to `completed`.
- Add a final task note summarising what was done and any follow-up items.
- If the completed work achieves a goal or outcome, update those too.

### Task types
Choose the right `task_type` when creating tasks:
- `feature` — new functionality
- `bugfix` — fixing broken behaviour
- `chore` — maintenance, dependency updates, config changes
- `investigation` — research, spikes, understanding a problem
- `general` — anything that doesn't fit the above

### Priority
Set `priority` based on impact and urgency:
- `urgent` — blocking other work or users right now
- `high` — important and should be done soon
- `medium` — normal priority (default)
- `low` — nice to have, no time pressure

### External links
When the user mentions a GitHub issue, Jira ticket, or any external reference, store it in `external_link` on the relevant project or task.

## Querying

- Use `list_tasks` with filters (`project_id`, `status`, `task_type`) rather than `get_task` in a loop.
- Use `list_problems` and `list_outcomes` with filters similarly.
- Use `list_goals` with filters (`project_id`, `task_id`, `goal_type`, `assignee`) to find specific goals.
- When giving the user an overview, combine results from multiple list calls to build a complete picture.

## What NOT to do

- Don't create duplicate projects or tasks. Check what exists first with `list_projects` / `list_tasks`.
- Don't add empty or trivial task notes. Every note should carry information worth reading later.
- Don't change task status without doing the associated work (or confirming the user has done it).
- Don't create problems for minor inconveniences — problems represent real blockers or issues that need tracking.
