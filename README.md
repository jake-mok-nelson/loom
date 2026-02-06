# Loom

Loom is a web-based project and task management application that efficiently helps you manage your projects, tasks, problems, goals, and outcomes. It provides a modern web dashboard with real-time updates and a REST API for programmatic access, all backed by a local SQLite database.

## Features

- **Project Management**: Create, list, get, update, and delete projects via web UI and REST API
- **Task Management**: Create, list, get, update, and delete tasks with status, priority, type, and notes
- **Problem Tracking**: Capture problems linked to projects and optionally to specific tasks
- **Goal Tracking**: Capture goals with optional project/task links and goal types
- **Outcome Tracking**: Track outcomes linked to projects and optionally to tasks for progress over time
- **Voice Notifications**: Text-to-speech capability for LLM tools to send voice messages to users
- **Web Dashboard**: Modern, responsive web interface with real-time updates via Server-Sent Events (SSE)
- **REST API**: Full REST API for programmatic access to all features
- **Local Storage**: All data stored in a local SQLite database (default: `~/.loom/loom.db`)

## Installation

### Prerequisites

- Go 1.23 or later
- Node.js and npm (optional, only needed to install echogarden for voice notifications)

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

# Install echogarden for voice notifications (optional)
npm install -g echogarden
```

## Configuration

By default, Loom stores its database at `~/.loom/loom.db`. You can customize this location by setting the `LOOM_DB_PATH` environment variable:

```bash
export LOOM_DB_PATH=/path/to/your/loom.db
```

## Usage

### Starting the Servers

Loom runs the API server (REST API + SSE) and the website (dashboard) on separate ports:

```bash
# Using the binary directly (API on :8080, Dashboard on :3000)
./loom

# Specify custom ports
./loom -addr :9090 -web-addr :4000

# Using make
make web
```

Then open your browser to http://localhost:3000 (or your custom web port) to view the dashboard.
The API and SSE endpoints are available at http://localhost:8080 (or your custom API port).

### Command-Line Options

- `-addr`: API server address and port (default: `:8080`)
- `-web-addr`: Website server address and port (default: `:3000`)

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
- `POST /api/voice` - Text-to-speech endpoint (accepts JSON with `text` field, returns WAV audio)
- `GET /events` - Server-Sent Events (SSE) endpoint for real-time updates

All API endpoints include CORS headers for cross-origin access.

### Voice Notifications

LLM tools can send voice messages to users through the `/api/voice` endpoint. The dashboard includes a speaker icon in the navbar that allows users to mute/unmute voice notifications. Voice state persists across sessions.

Example usage from JavaScript:
```javascript
// Use the built-in speakText function in the dashboard
speakText("Your task has been completed successfully");
```

Example usage via API:
```bash
curl -X POST http://localhost:8080/api/voice \
  -H "Content-Type: application/json" \
  -d '{"text":"Hello, your task is complete"}'
```

The voice feature uses echogarden with a British English voice. Users can toggle voice notifications using the speaker icon (🔊/🔇) in the dashboard navbar.

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
