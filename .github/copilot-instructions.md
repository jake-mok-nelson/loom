# Copilot Instructions for Loom

## Project Overview

Loom is a web-based project and task management application written in Go that serves as both a REST API and an MCP (Model Context Protocol) server. It enables LLM applications to manage projects, tasks, problems, goals, and outcomes through the standardized MCP protocol, while providing a real-time web dashboard for visualization.

### Architecture

- **Backend**: Go 1.23+ with SQLite database (`modernc.org/sqlite`)
- **MCP Server**: Implements MCP Streamable HTTP transport (spec 2025-03-26) using `github.com/mark3labs/mcp-go`
- **Web Server**: Dual-server architecture
  - API server (default `:8080`): REST API + MCP endpoints + SSE events
  - Dashboard server (default `:3000`): Web UI
- **Real-time Updates**: Server-Sent Events (SSE) for dashboard auto-refresh
- **Voice Notifications**: Text-to-speech using echogarden/Kokoro TTS

### Core Components

1. **database.go**: SQLite database layer with CRUD operations for all entities
2. **mcp_server.go**: MCP server implementation with tool registration
3. **webserver.go**: HTTP handlers for REST API, SSE, and dashboard
4. **main.go**: Application entry point with server initialization

## Coding Conventions

### Go Style

- Follow standard Go conventions and formatting (`gofmt`)
- Use descriptive variable names
- Keep functions focused and single-purpose
- Use meaningful error messages with context
- Leverage table-driven tests

### Database Patterns

**Entities**: Projects, Tasks, Problems, Outcomes, Goals, TaskNotes

**Schema Conventions**:
- Use snake_case for column names
- Every table has: `id` (INTEGER PRIMARY KEY), `created_at`, `updated_at` (TIMESTAMP)
- Status fields use lowercase strings: "active", "pending", "completed", "archived"
- Foreign keys nullable when optional: `project_id`, `task_id`

**CRUD Operations**:
```go
// Create: Return (*Entity, error)
func (db *Database) CreateProject(name, description, status, externalLink string) (*Project, error)

// Get: Return (*Entity, error)
func (db *Database) GetProject(id int64) (*Project, error)

// List: Return ([]*Entity, error), never nil slice
func (db *Database) ListProjects() ([]*Project, error)

// Update: Return error
func (db *Database) UpdateProject(id int64, name, description, status, externalLink string) error

// Delete: Return error
func (db *Database) DeleteProject(id int64) error
```

**Testing Pattern**:
```go
func newTestDatabase(t *testing.T) *Database {
    t.Helper()
    dbPath := filepath.Join(t.TempDir(), "loom.db")
    database, err := NewDatabase(dbPath)
    if err != nil {
        t.Fatalf("failed to create database: %v", err)
    }
    t.Cleanup(func() {
        database.Close()
    })
    return database
}
```

### MCP Server Patterns

**Tool Registration**:
- Group tools by entity type (projectTools, taskTools, etc.)
- Each tool returns `[]server.ServerTool`
- Use descriptive tool names: `create_project`, `list_tasks`, `update_goal`

**Tool Definition Pattern**:
```go
{
    Tool: mcp.NewTool("tool_name",
        mcp.WithDescription("Clear description"),
        mcp.WithString("param", mcp.Required(), mcp.Description("Parameter description")),
        mcp.WithNumber("id", mcp.Description("Optional numeric parameter")),
    ),
    Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Extract required parameters with RequireString/RequireFloat
        param, err := req.RequireString("param")
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }
        
        // Extract optional parameters with GetString/GetFloat
        optionalParam := req.GetString("optional", "default")
        
        // Perform operation
        result, err := db.Operation(param, optionalParam)
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("failed: %v", err)), nil
        }
        
        // Announce creation events (for create operations only)
        announceFunc(fmt.Sprintf("Entity %s created", param))
        
        // Return JSON result
        return jsonToolResult(result)
    },
}
```

**Helper Function**:
```go
func jsonToolResult(data interface{}) (*mcp.CallToolResult, error) {
    jsonData, err := json.Marshal(data)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
    }
    return mcp.NewToolResultText(string(jsonData)), nil
}
```

### Web Server Patterns

**REST API Endpoints**:
- Read-only GET endpoints at `/api/{entity}`
- Support query parameters for filtering
- Always set CORS headers for cross-origin access
- Return JSON with proper error handling

**API Handler Pattern**:
```go
func (ws *WebServer) handleEntity(w http.ResponseWriter, r *http.Request) {
    // CORS headers
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
    
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // Parse query parameters
    param := r.URL.Query().Get("param")
    
    // Fetch data
    data, err := ws.db.ListEntity(param)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Return JSON
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(data)
}
```

**SSE Broadcasting**:
- Use `ws.broadcast(eventType, data)` to push updates to dashboard
- Event types: "refresh", "voice"
- Voice announcements only for create operations via MCP
- All database modifications through MCP trigger dashboard refresh

### Testing

**Running Tests**:
```bash
go test ./...          # Run all tests
go test -v ./...       # Verbose output
go test -run TestName  # Run specific test
```

**Test Structure**:
- Use table-driven tests when testing multiple scenarios
- Use `t.Helper()` for test helper functions
- Use `t.TempDir()` for temporary database files
- Use `t.Cleanup()` for resource cleanup
- Descriptive test names: `TestCreateProject`, `TestListTasksWithFilters`

**Test Pattern**:
```go
func TestOperation(t *testing.T) {
    db := newTestDatabase(t)
    
    // Setup
    entity, err := db.CreateEntity("test", "description")
    if err != nil {
        t.Fatalf("setup failed: %v", err)
    }
    
    // Operation under test
    result, err := db.OperationUnderTest(entity.ID)
    
    // Assertions
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }
    if result.Field != expected {
        t.Errorf("expected %v, got %v", expected, result.Field)
    }
}
```

### Building and Running

**Development**:
```bash
make build          # Build binary
make test           # Run tests
make run            # Build and run with default ports
make web            # Build and run with explicit ports
make clean          # Remove binary
```

**Running**:
```bash
./loom                              # Default: API on :8080, Dashboard on :3000
./loom -addr :9090 -web-addr :4000  # Custom ports
LOOM_DB_PATH=/custom/path.db ./loom # Custom database location
```

## Code Guidelines

### When Adding New Features

1. **Database Layer**: Add schema, CRUD operations, and tests in `database.go` / `database_test.go`
2. **MCP Tools**: Register tools in `mcp_server.go` following naming conventions
3. **REST API**: Add read-only endpoints in `webserver.go` if dashboard needs data
4. **Testing**: Write comprehensive tests for all database operations
5. **Voice Announcements**: Only announce entity creation, not updates/deletes

### Common Patterns

**Error Handling**:
- Database operations: Return errors with context
- HTTP handlers: Use `http.Error()` with appropriate status codes
- MCP tools: Return `mcp.NewToolResultError()` for errors

**Nullable Foreign Keys**:
```go
type Entity struct {
    ProjectID *int64 `json:"project_id"` // Nullable
    TaskID    *int64 `json:"task_id"`    // Nullable
}
```

**Status Enums** (use lowercase strings):
- Projects: "active", "planning", "on_hold", "completed", "archived"
- Tasks: "pending", "in_progress", "completed", "blocked"
- Problems: "open", "in_progress", "resolved", "closed"
- Outcomes: "not_started", "in_progress", "completed", "blocked"

**Priority Enums** (lowercase):
- "low", "medium", "high", "critical"

### Dependencies

- **Database**: `modernc.org/sqlite` (pure Go SQLite)
- **MCP**: `github.com/mark3labs/mcp-go` (MCP SDK)
- **No web framework**: Standard library `net/http`

**Adding Dependencies**:
```bash
go get <package>
go mod tidy
```

## File Organization

```
.
├── main.go              # Entry point, server initialization
├── database.go          # Database layer, all CRUD operations
├── database_test.go     # Database tests
├── mcp_server.go        # MCP server, tool definitions
├── mcp_server_test.go   # MCP integration tests
├── webserver.go         # HTTP handlers, SSE, voice API
├── webserver_test.go    # Web server tests
├── go.mod               # Go module definition
├── Makefile             # Build and run targets
├── README.md            # Documentation
└── .github/
    └── copilot-instructions.md  # This file
```

## Key Concepts

### Dual-Server Architecture
- **API Server** (`:8080`): Serves REST API, MCP endpoints, SSE events
- **Dashboard Server** (`:3000`): Serves static web UI
- Separation allows independent scaling and simpler CORS handling

### MCP vs REST API
- **MCP** (`/sse`): Full CRUD for LLM agents, JSON-RPC 2.0, Streamable HTTP
- **REST API** (`/api/*`): Read-only for dashboard, simple GET endpoints
- LLM writes via MCP → Dashboard reads via REST → Real-time updates via SSE

### Voice Notifications
- Powered by echogarden + Kokoro TTS (offline, no API key)
- Only announced on entity creation (not updates/deletes)
- Dashboard controls: speaker icon (🔊/🔇) with persistent state
- Endpoint: `POST /api/voice` (JSON with `text` field, returns WAV)

## Tips for Copilot

- Follow the established patterns in existing code
- Use table-driven tests for multiple scenarios
- Keep database operations in `database.go`
- Keep MCP tools in `mcp_server.go`
- Keep HTTP handlers in `webserver.go`
- Use meaningful error messages with context
- Never return nil slices from List operations
- Always use `t.Helper()` in test helper functions
- Use `announceFunc()` only for creation events in MCP tools
- Broadcast SSE "refresh" events after database modifications
