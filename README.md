# Loom

Loom is a web-based project and task management application that efficiently helps you manage your projects, tasks, problems, goals, and outcomes. It provides a modern web dashboard with real-time updates and a REST API for programmatic access, all backed by a local SQLite database.

## Features

- **Project Management**: Create, list, get, update, and delete projects via web UI and REST API
- **Task Management**: Create, list, get, update, and delete tasks with status, priority, type, and notes
- **Problem Tracking**: Capture problems linked to projects and optionally to specific tasks
- **Goal Tracking**: Capture goals with optional project/task links and goal types
- **Outcome Tracking**: Track outcomes linked to projects and optionally to tasks for progress over time
- **Web Dashboard**: Modern, responsive web interface with real-time updates via Server-Sent Events (SSE)
- **REST API**: Full REST API for programmatic access to all features
- **Local Storage**: All data stored in a local SQLite database (default: `~/.loom/loom.db`)

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

## Usage

### Starting the Web Server

```bash
# Using the binary directly
./loom

# Specify a custom port
./loom -addr :9090

# Using make
make web
```

Then open your browser to http://localhost:8080 (or your custom port) to view the dashboard.

### Command-Line Options

- `-addr`: Server address and port (default: `:8080`)

You can also set the `LOOM_DB_PATH` environment variable to use a custom database location.

## Web Dashboard

Loom provides a reactive web dashboard for visualizing and managing your projects and tasks. The dashboard provides real-time updates via Server-Sent Events (SSE) and a modern, desktop-optimized interface.

### Dashboard Features

- **Overview**: Shows recent activity across all data types
- **Projects**: View and filter all projects by status (active, planning, on_hold, completed, archived) with task status summaries
- **Tasks**: Filter by status, priority, type, and project
- **Problems**: Track issues linked to projects and tasks with status filtering
- **Outcomes**: Monitor progress tracking for projects with status filtering
- **Goals**: View short-term, career, values, and requirement goals
- **Real-time Updates**: Dashboard automatically refreshes when data changes
- **Search**: Global search across all items
- **Dark Theme**: Modern, eye-friendly dark interface optimized for desktop use

## REST API

Loom provides a REST API for programmatic access to all features. All endpoints return JSON responses.

### API Endpoints

- `GET /api/projects` - List all projects
- `GET /api/tasks?project_id=1&status=pending&task_type=feature` - List tasks with optional filters
- `GET /api/problems?project_id=1&task_id=2&status=open` - List problems with optional filters
- `GET /api/outcomes?project_id=1&task_id=2&status=completed` - List outcomes with optional filters
- `GET /api/goals?project_id=1&task_id=2&goal_type=short_term` - List goals with optional filters
- `GET /events` - Server-Sent Events (SSE) endpoint for real-time updates

All API endpoints include CORS headers for cross-origin access.

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
