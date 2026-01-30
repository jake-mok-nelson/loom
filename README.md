# Loom

Loom is an MCP (Model Context Protocol) server that efficiently and simply helps users of LLMs like Claude to manage their projects and tasks. It provides convenient MCP tools for project and task management in a local SQLite database.

## Features

- **Project Management**: Create, list, get, update, and delete projects
- **Task Management**: Create, list, get, update, and delete tasks with status and priority tracking
- **Local Storage**: All data stored in a local SQLite database (default: `~/.loom/loom.db`)
- **MCP Integration**: Works seamlessly with any MCP-compatible LLM client

## Installation

### Prerequisites

- Go 1.21 or later

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

#### list_projects
List all projects.

#### get_project
Get details of a specific project.

**Arguments:**
- `id` (number, required): Project ID

#### update_project
Update an existing project.

**Arguments:**
- `id` (number, required): Project ID
- `name` (string, required): Project name
- `description` (string, optional): Project description

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
- `status` (string, optional): Task status (pending, in_progress, completed, blocked) - default: "pending"
- `priority` (string, optional): Task priority (low, medium, high, urgent) - default: "medium"

#### list_tasks
List tasks, optionally filtered by project and/or status.

**Arguments:**
- `project_id` (number, optional): Filter by project ID
- `status` (string, optional): Filter by status

#### get_task
Get details of a specific task.

**Arguments:**
- `id` (number, required): Task ID

#### update_task
Update an existing task.

**Arguments:**
- `id` (number, required): Task ID
- `title` (string, required): Task title
- `description` (string, optional): Task description
- `status` (string, optional): Task status
- `priority` (string, optional): Task priority

#### delete_task
Delete a task.

**Arguments:**
- `id` (number, required): Task ID

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
