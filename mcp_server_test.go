package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
)

// setupTestMCPServer creates a test database and MCP server using mcptest
func setupTestMCPServer(t *testing.T) (*mcptest.Server, *Database, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "loom-mcp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	testDB, err := NewDatabase(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create database: %v", err)
	}

	srv := mcptest.NewUnstartedServer(t)

	srv.AddTools(projectTools(testDB, func(string) {})...)
	srv.AddTools(taskTools(testDB, func(string) {})...)
	srv.AddTools(problemTools(testDB, func(string) {})...)
	srv.AddTools(outcomeTools(testDB, func(string) {})...)
	srv.AddTools(goalTools(testDB, func(string) {})...)
	srv.AddTools(taskNoteTools(testDB, func(string) {})...)

	if err := srv.Start(context.Background()); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to start MCP server: %v", err)
	}

	cleanup := func() {
		srv.Close()
		testDB.Close()
		os.RemoveAll(tempDir)
	}

	return srv, testDB, cleanup
}

// callMCPTool calls a tool via the mcptest client and returns the result.
func callMCPTool(t *testing.T, srv *mcptest.Server, toolName string, args map[string]interface{}) *mcp.CallToolResult {
	t.Helper()

	var req mcp.CallToolRequest
	req.Params.Name = toolName
	req.Params.Arguments = args

	result, err := srv.Client().CallTool(context.Background(), req)
	if err != nil {
		t.Fatalf("CallTool %s failed: %v", toolName, err)
	}

	return result
}

// getTextContent extracts text content from a CallToolResult.
func getTextContent(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func TestNewMCPServer(t *testing.T) {
	s, _, cleanup := setupTestMCPServer(t)
	defer cleanup()

	if s == nil {
		t.Fatal("Expected non-nil MCP server")
	}
}

func TestMCPCreateAndListProjects(t *testing.T) {
	s, _, cleanup := setupTestMCPServer(t)
	defer cleanup()

	result := callMCPTool(t, s, "create_project", map[string]interface{}{
		"name":        "Test Project",
		"description": "A test project",
		"status":      "active",
	})
	if result.IsError {
		t.Fatalf("create_project returned error: %s", getTextContent(result))
	}

	result = callMCPTool(t, s, "list_projects", map[string]interface{}{})
	if result.IsError {
		t.Fatalf("list_projects returned error: %s", getTextContent(result))
	}

	var projects []Project
	if err := json.Unmarshal([]byte(getTextContent(result)), &projects); err != nil {
		t.Fatalf("Failed to parse projects JSON: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}
	if projects[0].Name != "Test Project" {
		t.Errorf("Expected project name 'Test Project', got '%s'", projects[0].Name)
	}
}

func TestMCPGetAndUpdateProject(t *testing.T) {
	s, testDB, cleanup := setupTestMCPServer(t)
	defer cleanup()

	project, _ := testDB.CreateProject("Original", "desc", "active", "")

	result := callMCPTool(t, s, "get_project", map[string]interface{}{
		"id": float64(project.ID),
	})
	if result.IsError {
		t.Fatalf("get_project returned error: %s", getTextContent(result))
	}

	result = callMCPTool(t, s, "update_project", map[string]interface{}{
		"id":   float64(project.ID),
		"name": "Updated",
	})
	if result.IsError {
		t.Fatalf("update_project returned error: %s", getTextContent(result))
	}

	var updated Project
	if err := json.Unmarshal([]byte(getTextContent(result)), &updated); err != nil {
		t.Fatalf("Failed to parse updated project JSON: %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("Expected updated name 'Updated', got '%s'", updated.Name)
	}
}

func TestMCPDeleteProject(t *testing.T) {
	s, testDB, cleanup := setupTestMCPServer(t)
	defer cleanup()

	project, _ := testDB.CreateProject("ToDelete", "", "", "")

	result := callMCPTool(t, s, "delete_project", map[string]interface{}{
		"id": float64(project.ID),
	})
	if result.IsError {
		t.Fatalf("delete_project returned error: %s", getTextContent(result))
	}

	result = callMCPTool(t, s, "list_projects", map[string]interface{}{})
	var projects []Project
	if err := json.Unmarshal([]byte(getTextContent(result)), &projects); err != nil {
		t.Fatalf("Failed to parse projects JSON: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects after delete, got %d", len(projects))
	}
}

func TestMCPCreateAndListTasks(t *testing.T) {
	s, testDB, cleanup := setupTestMCPServer(t)
	defer cleanup()

	project, _ := testDB.CreateProject("Test Project", "", "", "")

	result := callMCPTool(t, s, "create_task", map[string]interface{}{
		"project_id": float64(project.ID),
		"title":      "Test Task",
		"status":     "pending",
		"priority":   "high",
		"task_type":  "feature",
	})
	if result.IsError {
		t.Fatalf("create_task returned error: %s", getTextContent(result))
	}

	result = callMCPTool(t, s, "list_tasks", map[string]interface{}{})
	var tasks []Task
	if err := json.Unmarshal([]byte(getTextContent(result)), &tasks); err != nil {
		t.Fatalf("Failed to parse tasks JSON: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Title != "Test Task" {
		t.Errorf("Expected task title 'Test Task', got '%s'", tasks[0].Title)
	}
}

func TestMCPCreateAndListProblems(t *testing.T) {
	s, _, cleanup := setupTestMCPServer(t)
	defer cleanup()

	result := callMCPTool(t, s, "create_problem", map[string]interface{}{
		"title":       "Test Problem",
		"description": "A test problem",
		"status":      "open",
	})
	if result.IsError {
		t.Fatalf("create_problem returned error: %s", getTextContent(result))
	}

	result = callMCPTool(t, s, "list_problems", map[string]interface{}{})
	var problems []Problem
	if err := json.Unmarshal([]byte(getTextContent(result)), &problems); err != nil {
		t.Fatalf("Failed to parse problems JSON: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("Expected 1 problem, got %d", len(problems))
	}
	if problems[0].Title != "Test Problem" {
		t.Errorf("Expected problem title 'Test Problem', got '%s'", problems[0].Title)
	}
}

func TestMCPCreateAndListGoals(t *testing.T) {
	s, _, cleanup := setupTestMCPServer(t)
	defer cleanup()

	result := callMCPTool(t, s, "create_goal", map[string]interface{}{
		"title":     "Test Goal",
		"goal_type": "short_term",
	})
	if result.IsError {
		t.Fatalf("create_goal returned error: %s", getTextContent(result))
	}

	result = callMCPTool(t, s, "list_goals", map[string]interface{}{})
	var goals []Goal
	if err := json.Unmarshal([]byte(getTextContent(result)), &goals); err != nil {
		t.Fatalf("Failed to parse goals JSON: %v", err)
	}
	if len(goals) != 1 {
		t.Fatalf("Expected 1 goal, got %d", len(goals))
	}
	if goals[0].Title != "Test Goal" {
		t.Errorf("Expected goal title 'Test Goal', got '%s'", goals[0].Title)
	}
}

func TestMCPCreateAndListOutcomes(t *testing.T) {
	s, testDB, cleanup := setupTestMCPServer(t)
	defer cleanup()

	project, _ := testDB.CreateProject("Test Project", "", "", "")

	result := callMCPTool(t, s, "create_outcome", map[string]interface{}{
		"project_id":  float64(project.ID),
		"title":       "Test Outcome",
		"description": "A test outcome",
		"status":      "open",
	})
	if result.IsError {
		t.Fatalf("create_outcome returned error: %s", getTextContent(result))
	}

	result = callMCPTool(t, s, "list_outcomes", map[string]interface{}{})
	var outcomes []Outcome
	if err := json.Unmarshal([]byte(getTextContent(result)), &outcomes); err != nil {
		t.Fatalf("Failed to parse outcomes JSON: %v", err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("Expected 1 outcome, got %d", len(outcomes))
	}
	if outcomes[0].Title != "Test Outcome" {
		t.Errorf("Expected outcome title 'Test Outcome', got '%s'", outcomes[0].Title)
	}
}

func TestMCPCreateAndListTaskNotes(t *testing.T) {
	s, testDB, cleanup := setupTestMCPServer(t)
	defer cleanup()

	project, _ := testDB.CreateProject("Test Project", "", "", "")
	task, _ := testDB.CreateTask(project.ID, "Test Task", "", "pending", "high", "feature", "")

	result := callMCPTool(t, s, "create_task_note", map[string]interface{}{
		"task_id": float64(task.ID),
		"note":    "Test note content",
	})
	if result.IsError {
		t.Fatalf("create_task_note returned error: %s", getTextContent(result))
	}

	result = callMCPTool(t, s, "list_task_notes", map[string]interface{}{
		"task_id": float64(task.ID),
	})
	var notes []TaskNote
	if err := json.Unmarshal([]byte(getTextContent(result)), &notes); err != nil {
		t.Fatalf("Failed to parse task notes JSON: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("Expected 1 task note, got %d", len(notes))
	}
	if notes[0].Note != "Test note content" {
		t.Errorf("Expected note content 'Test note content', got '%s'", notes[0].Note)
	}
}

func TestMCPOptionalHelpers(t *testing.T) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"name": "hello",
		"id":   float64(42),
	}

	s := optionalString(req, "name")
	if s == nil || *s != "hello" {
		t.Errorf("Expected 'hello', got %v", s)
	}

	s = optionalString(req, "missing")
	if s != nil {
		t.Errorf("Expected nil, got %v", *s)
	}

	i := optionalInt64(req, "id")
	if i == nil || *i != 42 {
		t.Errorf("Expected 42, got %v", i)
	}

	i = optionalInt64(req, "missing")
	if i != nil {
		t.Errorf("Expected nil, got %v", *i)
	}
}
