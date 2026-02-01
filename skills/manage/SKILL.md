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
Issues or blockers, optionally linked to a project and/or task. A problem linked to a task must belong to the same project.

Fields: `title` (required), `project_id`, `task_id`, `description`, `status` (open | in_progress | resolved | blocked).

### Goals
Objectives optionally linked to a project and/or task. A goal linked to a task must belong to the same project.

Fields: `title` (required), `project_id`, `task_id`, `description`, `goal_type` (short_term | career | values | requirement).

### Outcomes
Results or milestones linked to a project (required) and optionally a task. A task must belong to the same project.

Fields: `project_id` (required), `title` (required), `task_id`, `description`, `status` (open | in_progress | completed | blocked).

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
- When giving the user an overview, combine results from multiple list calls to build a complete picture.

## What NOT to do

- Don't create duplicate projects or tasks. Check what exists first with `list_projects` / `list_tasks`.
- Don't add empty or trivial task notes. Every note should carry information worth reading later.
- Don't change task status without doing the associated work (or confirming the user has done it).
- Don't create problems for minor inconveniences — problems represent real blockers or issues that need tracking.
