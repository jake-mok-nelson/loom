# Loom

Loom is a web-based project and task management application that efficiently helps you manage your projects, tasks, problems, goals, and outcomes. It provides a modern web dashboard with real-time updates and a REST API for programmatic access, all backed by a local SQLite database. Loom also serves as an MCP (Model Context Protocol) server using the Streamable HTTP transport, enabling LLM applications to manage projects and tasks through the standardized MCP protocol.

## Features

- **Project Management**: Create, list, get, update, and delete projects via web UI and REST API
- **Task Management**: Create, list, get, update, and delete tasks with status, priority, type, and notes
- **Problem Tracking**: Capture problems linked to projects and optionally to specific tasks
- **Goal Tracking**: Capture goals with optional project/task links and goal types
- **Outcome Tracking**: Track outcomes linked to projects and optionally to tasks for progress over time
- **Voice Notifications**: Text-to-speech capability for LLM tools to send voice messages to users
- **Web Dashboard**: Modern, responsive web interface with real-time updates via Server-Sent Events (SSE)
- **MCP Server**: Full MCP (Model Context Protocol) server with Streamable HTTP transport (2025-03-26 spec) for LLM tool integration
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

Loom runs the API server (REST API, SSE, and MCP) and the website (dashboard) on separate ports:

```bash
# Using the binary directly (API + MCP on :8080, Dashboard on :3000)
./loom

# Specify custom ports
./loom -addr :9090 -web-addr :4000

# Using make
make web
```

Then open your browser to http://localhost:3000 (or your custom web port) to view the dashboard.
The API, SSE, and MCP Streamable HTTP endpoints are all available at http://localhost:8080 (or your custom API port).

### Command-Line Options

- `-addr`: API and MCP server address and port (default: `:8080`)
- `-web-addr`: Website server address and port (default: `:3000`)

You can also set the `LOOM_DB_PATH` environment variable to use a custom database location.

## Architecture: REST API vs MCP

Loom exposes two complementary interfaces on the same port, each serving a different audience:

| | REST API (`/api/*`) | MCP (`/sse`) |
|---|---|---|
| **Consumer** | Web dashboard, scripts, HTTP clients | LLM applications (Claude Desktop, etc.) |
| **Protocol** | Standard REST (JSON over HTTP) | JSON-RPC 2.0 over Streamable HTTP |
| **Capabilities** | **Read-only** — query projects, tasks, problems, outcomes, goals | **Full CRUD** — create, read, update, delete, and link/unlink all entities |
| **Real-time** | `GET /events` pushes SSE updates to the dashboard | `GET /sse` streams MCP notifications to LLM clients |
| **Use case** | Display data in the browser dashboard | Let AI agents manage projects programmatically |

### Why both?

- **The REST API powers the dashboard.** The web UI needs simple, fast GET endpoints to fetch and display data, and an SSE stream (`/events`) to refresh automatically when data changes. It doesn't need write access because users interact through their LLM tool, not through the dashboard directly.

- **MCP powers LLM tool integration.** AI agents (like Claude) need a standardized protocol to discover available tools and execute them. MCP provides this via JSON-RPC with full create/read/update/delete capabilities across all entity types. When an MCP tool modifies data, the dashboard receives a real-time refresh event through the REST SSE stream.

Together, the two interfaces form a loop: **LLM agents write data through MCP → the dashboard reads and displays it through the REST API → changes appear in real time via SSE.**

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

## MCP Server (Streamable HTTP)

Loom implements the [Model Context Protocol (MCP)](https://modelcontextprotocol.io) using the **Streamable HTTP** transport (spec version `2025-03-26`). This replaces the legacy HTTP+SSE transport and provides a single endpoint at `/sse` that supports both JSON and streaming responses.

### MCP Endpoint

- `POST /sse` - Send JSON-RPC 2.0 requests (initialize, tools/list, tools/call, etc.)
- `GET /sse` - Open a streaming connection for server-to-client notifications
- `DELETE /sse` - Terminate a session

### Available MCP Tools

All project management operations are available as MCP tools:

| Tool | Description |
|------|-------------|
| `create_project` | Create a new project |
| `list_projects` | List all projects |
| `get_project` | Get project details |
| `update_project` | Update a project |
| `delete_project` | Delete a project |
| `create_task` | Create a task in a project |
| `list_tasks` | List tasks with filters |
| `get_task` | Get task details |
| `update_task` | Update a task |
| `delete_task` | Delete a task |
| `create_problem` | Create a problem |
| `list_problems` | List problems with filters |
| `get_problem` | Get problem details |
| `update_problem` | Update a problem |
| `delete_problem` | Delete a problem |
| `link_problem_to_project` | Link problem to project |
| `unlink_problem_from_project` | Unlink problem from project |
| `get_problem_projects` | Get projects for a problem |
| `get_project_problems` | Get problems for a project |
| `create_outcome` | Create an outcome |
| `list_outcomes` | List outcomes with filters |
| `get_outcome` | Get outcome details |
| `update_outcome` | Update an outcome |
| `delete_outcome` | Delete an outcome |
| `create_goal` | Create a goal |
| `list_goals` | List goals with filters |
| `get_goal` | Get goal details |
| `update_goal` | Update a goal |
| `delete_goal` | Delete a goal |
| `link_goal_to_project` | Link goal to project |
| `unlink_goal_from_project` | Unlink goal from project |
| `get_goal_projects` | Get projects for a goal |
| `get_project_goals` | Get goals for a project |
| `create_task_note` | Create a note on a task |
| `list_task_notes` | List notes for a task |
| `get_task_note` | Get a task note |
| `update_task_note` | Update a task note |
| `delete_task_note` | Delete a task note |

### MCP Client Configuration

To connect an MCP client (e.g., Claude Desktop) to Loom, use the Streamable HTTP transport configuration:

```json
{
  "mcpServers": {
    "loom": {
      "type": "streamable-http",
      "url": "http://localhost:8080/sse"
    }
  }
}
```

### Voice Notifications

Loom provides text-to-speech capabilities for voice announcements when tasks, problems, goals, and outcomes are created via MCP. The dashboard includes a speaker icon in the navbar (🔊/🔇) that allows users to mute/unmute voice notifications. Voice state persists across sessions.

#### Setting up Kokoro TTS

Loom uses [echogarden](https://github.com/ampleforth/echogarden) with the **Kokoro** offline TTS engine for high-quality, natural-sounding voice synthesis.

##### Installation

1. **Install echogarden** (requires Node.js 18+):
```bash
npm install -g echogarden
```

2. **Download Kokoro models** (optional, models download automatically on first use):
The Kokoro models will be automatically downloaded on the first call to `/api/voice`. The download includes:
   - `kokoro-82m-v1.0-fp32` (301MB) - Main model
   - `kokoro-82m-v1.0-voices` (26MB) - Voice presets

Models are cached in `~/.cache/echogarden/` for subsequent uses.

##### Configuration

Loom is pre-configured to use Kokoro. No additional setup is needed. The voice endpoint uses:
- **Engine**: Kokoro (offline, no API key required)
- **Quality**: 24kHz sample rate, natural-sounding voices
- **Voice**: Automatically selects appropriate voice for detected language

##### Testing Voice Synthesis

Test the voice endpoint manually:
```bash
curl -X POST http://localhost:8080/api/voice \
  -H "Content-Type: application/json" \
  -d '{"text":"Hello from Kokoro"}' \
  -o output.wav && open output.wav
```

##### Usage

Voice announcements are automatically sent to the dashboard via SSE when items are created:
- When you create a task via MCP: "Task {title} created"
- When you create a problem via MCP: "Problem {title} created"
- When you create a goal via MCP: "Goal {title} created"
- When you create an outcome via MCP: "Outcome {title} created"

Users can toggle voice notifications using the speaker icon (🔊/🔇) in the dashboard navbar.

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
