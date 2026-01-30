# Loom

Loom is an MCP (Model Context Protocol) server that efficiently and simply helps users of LLMs like Claude to manage their projects and tasks. It provides convenient MCP tools for project and task management in a local SQLite database.

## Features

- **Project Management**: Create, list, get, update, and delete projects
- **Task Management**: Create, list, get, update, and delete tasks with status, priority, type, and notes
- **Problem Tracking**: Capture problems linked to projects and optionally to specific tasks
- **Outcome Tracking**: Track outcomes linked to projects and optionally to tasks for progress over time
- **Local Storage**: All data stored in a local SQLite database (default: `~/.loom/loom.db`)
- **MCP Integration**: Works seamlessly with any MCP-compatible LLM client

## Installation

### Prerequisites

- Go 1.23 or later

### Build from Source

```bash
git clone https://github.com/jake-mok-nelson/loom.git
cd loom
go build -o loom
```

### Install

```bash
# Move the binary to your PATH
sudo mv loom /usr/local/bin/
```

## Configuration

By default, Loom stores its database at `~/.loom/loom.db`. You can customize this location by setting the `LOOM_DB_PATH` environment variable:

```bash
export LOOM_DB_PATH=/path/to/your/loom.db
```

## MCP Tools

Loom provides the following MCP tools:

### Project Management

#### create_project
Create a new project.

**Arguments:**
- `name` (string, required): Project name
- `description` (string, optional): Project description
- `external_link` (string, optional): External link to ticket system or other tracking tool

#### list_projects
List all projects.

#### get_project
Get details of a specific project.

**Arguments:**
- `id` (number, required): Project ID

#### update_project
Update an existing project. Only provided fields will be updated.

**Arguments:**
- `id` (number, required): Project ID
- `name` (string, optional): Project name
- `description` (string, optional): Project description
- `external_link` (string, optional): External link to ticket system or other tracking tool

#### delete_project
Delete a project and all its associated tasks.

**Arguments:**
- `id` (number, required): Project ID

### Task Management

#### create_task
Create a new task in a project.

**Arguments:**
- `project_id` (number, required): Project ID
- `title` (string, required): Task title
- `description` (string, optional): Task description
- `task_type` (string, optional): Task type (general, chore, investigation, feature, bugfix) - default: "general"
- `status` (string, optional): Task status (pending, in_progress, completed, blocked) - default: "pending"
- `priority` (string, optional): Task priority (low, medium, high, urgent) - default: "medium"
- `external_link` (string, optional): External link to ticket system or other tracking tool

#### list_tasks
List tasks, optionally filtered by project, type, and/or status.

**Arguments:**
- `project_id` (number, optional): Filter by project ID
- `task_type` (string, optional): Filter by task type
- `status` (string, optional): Filter by status

#### get_task
Get details of a specific task.

**Arguments:**
- `id` (number, required): Task ID

#### update_task
Update an existing task. Only provided fields will be updated.

**Arguments:**
- `id` (number, required): Task ID
- `title` (string, optional): Task title
- `description` (string, optional): Task description
- `status` (string, optional): Task status (pending, in_progress, completed, blocked)
- `priority` (string, optional): Task priority (low, medium, high, urgent)
- `task_type` (string, optional): Task type (general, chore, investigation, feature, bugfix)
- `external_link` (string, optional): External link to ticket system or other tracking tool

#### delete_task
Delete a task.

**Arguments:**
- `id` (number, required): Task ID

### Problem Management

#### create_problem
Create a new problem linked to a project and optionally to a task.

**Arguments:**
- `project_id` (number, required): Project ID
- `task_id` (number, optional): Task ID (must belong to the project)
- `title` (string, required): Problem title
- `description` (string, optional): Problem description
- `status` (string, optional): Problem status (open, in_progress, resolved, blocked) - default: "open"

#### list_problems
List problems, optionally filtered by project, task, and/or status.

**Arguments:**
- `project_id` (number, optional): Filter by project ID
- `task_id` (number, optional): Filter by task ID
- `status` (string, optional): Filter by status

#### get_problem
Get details of a specific problem.

**Arguments:**
- `id` (number, required): Problem ID

#### update_problem
Update an existing problem. Only provided fields will be updated.

**Arguments:**
- `id` (number, required): Problem ID
- `title` (string, optional): Problem title
- `description` (string, optional): Problem description
- `status` (string, optional): Problem status (open, in_progress, resolved, blocked)

#### delete_problem
Delete a problem.

**Arguments:**
- `id` (number, required): Problem ID

### Outcome Management

#### create_outcome
Create a new outcome linked to a project and optionally to a task.

**Arguments:**
- `project_id` (number, required): Project ID
- `task_id` (number, optional): Task ID (must belong to the project)
- `title` (string, required): Outcome title
- `description` (string, optional): Outcome description
- `status` (string, optional): Outcome status (open, in_progress, completed, blocked) - default: "open"

#### list_outcomes
List outcomes, optionally filtered by project, task, and/or status.

**Arguments:**
- `project_id` (number, optional): Filter by project ID
- `task_id` (number, optional): Filter by task ID
- `status` (string, optional): Filter by status

#### get_outcome
Get details of a specific outcome.

**Arguments:**
- `id` (number, required): Outcome ID

#### update_outcome
Update an existing outcome. Only provided fields will be updated.

**Arguments:**
- `id` (number, required): Outcome ID
- `title` (string, optional): Outcome title
- `description` (string, optional): Outcome description
- `status` (string, optional): Outcome status (open, in_progress, completed, blocked)

#### delete_outcome
Delete an outcome.

**Arguments:**
- `id` (number, required): Outcome ID

### Task Note Management

#### create_task_note
Create a note on a task.

**Arguments:**
- `task_id` (number, required): Task ID
- `note` (string, required): Note content

#### list_task_notes
List notes for a task.

**Arguments:**
- `task_id` (number, required): Task ID

#### get_task_note
Get details of a specific task note.

**Arguments:**
- `id` (number, required): Task note ID

#### update_task_note
Update an existing task note.

**Arguments:**
- `id` (number, required): Task note ID
- `note` (string, required): Note content

#### delete_task_note
Delete a task note.

**Arguments:**
- `id` (number, required): Task note ID

## Usage with MCP Clients

To use Loom with an MCP-compatible client (like Claude Desktop), add it to your client's MCP configuration:

```json
{
  "mcpServers": {
    "loom": {
      "command": "/usr/local/bin/loom"
    }
  }
}
```

Or if you want to use a custom database location:

```json
{
  "mcpServers": {
    "loom": {
      "command": "/usr/local/bin/loom",
      "env": {
        "LOOM_DB_PATH": "/custom/path/loom.db"
      }
    }
  }
}
```

## Example Workflow

1. Create a project: `create_project(name="My Web App", description="A new web application project")`
2. Create tasks: `create_task(project_id=1, title="Setup database", status="in_progress", priority="high")`
3. List tasks: `list_tasks(project_id=1, status="in_progress")`
4. Update task: `update_task(id=1, title="Setup database", status="completed")`
5. View project details: `get_project(id=1)`

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o loom
```

## License

MIT
