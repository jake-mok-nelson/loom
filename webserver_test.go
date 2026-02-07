package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// setupTestWebServer creates a test database and web server
func setupTestWebServer(t *testing.T) (*WebServer, *Database, func()) {
	// Create temp directory for test database
	tempDir, err := os.MkdirTemp("", "loom-webserver-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	testDB, err := NewDatabase(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create database: %v", err)
	}

	ws := NewWebServer(testDB, ":0", ":0", nil)

	cleanup := func() {
		testDB.Close()
		os.RemoveAll(tempDir)
	}

	return ws, testDB, cleanup
}

func TestNewWebServer(t *testing.T) {
	ws, _, cleanup := setupTestWebServer(t)
	defer cleanup()

	if ws == nil {
		t.Fatal("Expected non-nil WebServer")
	}

	if ws.addr != ":0" {
		t.Errorf("Expected addr :0, got %s", ws.addr)
	}

	if ws.webAddr != ":0" {
		t.Errorf("Expected webAddr :0, got %s", ws.webAddr)
	}

	if ws.clients == nil {
		t.Error("Expected clients map to be initialized")
	}
}

func TestHandleDashboard(t *testing.T) {
	ws, _, cleanup := setupTestWebServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "localhost:3000"
	rr := httptest.NewRecorder()

	ws.handleDashboard(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Handler returned wrong content type: got %v want text/html; charset=utf-8", contentType)
	}

	body := rr.Body.String()
	if len(body) == 0 {
		t.Error("Expected non-empty body")
	}

	// Check for key HTML elements
	if !contains(body, "<!DOCTYPE html>") {
		t.Error("Expected HTML doctype")
	}
	if !contains(body, "Loom Dashboard") {
		t.Error("Expected page title")
	}
	if !contains(body, "Projects") {
		t.Error("Expected Projects navigation")
	}
	if !contains(body, "API_BASE_URL") {
		t.Error("Expected API_BASE_URL to be injected into the HTML")
	}
}

func TestHandleDashboard404(t *testing.T) {
	ws, _, cleanup := setupTestWebServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rr := httptest.NewRecorder()

	ws.handleDashboard(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code for non-root path: got %v want %v", status, http.StatusNotFound)
	}
}

func TestHandleProjectsEmpty(t *testing.T) {
	ws, _, cleanup := setupTestWebServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/projects", nil)
	rr := httptest.NewRecorder()

	ws.handleProjects(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Handler returned wrong content type: got %v want application/json", contentType)
	}

	var projects []Project
	if err := json.Unmarshal(rr.Body.Bytes(), &projects); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("Expected empty projects array, got %d items", len(projects))
	}
}

func TestHandleProjectsWithData(t *testing.T) {
	ws, testDB, cleanup := setupTestWebServer(t)
	defer cleanup()

	// Create test project
	_, err := testDB.CreateProject("Test Project", "A test description", "", "https://example.com")
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/projects", nil)
	rr := httptest.NewRecorder()

	ws.handleProjects(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var projects []Project
	if err := json.Unmarshal(rr.Body.Bytes(), &projects); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}

	if projects[0].Name != "Test Project" {
		t.Errorf("Expected project name 'Test Project', got '%s'", projects[0].Name)
	}
}

func TestHandleTasksEmpty(t *testing.T) {
	ws, _, cleanup := setupTestWebServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/tasks", nil)
	rr := httptest.NewRecorder()

	ws.handleTasks(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var tasks []Task
	if err := json.Unmarshal(rr.Body.Bytes(), &tasks); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("Expected empty tasks array, got %d items", len(tasks))
	}
}

func TestHandleTasksWithFilters(t *testing.T) {
	ws, testDB, cleanup := setupTestWebServer(t)
	defer cleanup()

	// Create test project and tasks
	project, _ := testDB.CreateProject("Test Project", "", "", "")
	testDB.CreateTask(project.ID, "Task 1", "", "pending", "high", "feature", "")
	testDB.CreateTask(project.ID, "Task 2", "", "completed", "low", "bugfix", "")

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{"all tasks", "/api/tasks", 2},
		{"filter by status", "/api/tasks?status=pending", 1},
		{"filter by project", "/api/tasks?project_id=" + strconv.FormatInt(project.ID, 10), 2},
		{"filter by type", "/api/tasks?task_type=feature", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.query, nil)
			rr := httptest.NewRecorder()

			ws.handleTasks(rr, req)

			var tasks []Task
			json.Unmarshal(rr.Body.Bytes(), &tasks)

			if len(tasks) != tt.expectedCount {
				t.Errorf("Expected %d tasks, got %d", tt.expectedCount, len(tasks))
			}
		})
	}
}

func TestHandleProblemsEmpty(t *testing.T) {
	ws, _, cleanup := setupTestWebServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/problems", nil)
	rr := httptest.NewRecorder()

	ws.handleProblems(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var problems []Problem
	if err := json.Unmarshal(rr.Body.Bytes(), &problems); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if len(problems) != 0 {
		t.Errorf("Expected empty problems array, got %d items", len(problems))
	}
}

func TestHandleProblemsWithData(t *testing.T) {
	ws, testDB, cleanup := setupTestWebServer(t)
	defer cleanup()

	// Create test problem
	project, _ := testDB.CreateProject("Test Project", "", "", "")
	_, err := testDB.CreateProblem(&project.ID, nil, "Test Problem", "A problem description", "open", "")
	if err != nil {
		t.Fatalf("Failed to create test problem: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/problems", nil)
	rr := httptest.NewRecorder()

	ws.handleProblems(rr, req)

	var problems []Problem
	json.Unmarshal(rr.Body.Bytes(), &problems)

	if len(problems) != 1 {
		t.Fatalf("Expected 1 problem, got %d", len(problems))
	}

	if problems[0].Title != "Test Problem" {
		t.Errorf("Expected problem title 'Test Problem', got '%s'", problems[0].Title)
	}
}

func TestHandleOutcomesEmpty(t *testing.T) {
	ws, _, cleanup := setupTestWebServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/outcomes", nil)
	rr := httptest.NewRecorder()

	ws.handleOutcomes(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var outcomes []Outcome
	if err := json.Unmarshal(rr.Body.Bytes(), &outcomes); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if len(outcomes) != 0 {
		t.Errorf("Expected empty outcomes array, got %d items", len(outcomes))
	}
}

func TestHandleOutcomesWithData(t *testing.T) {
	ws, testDB, cleanup := setupTestWebServer(t)
	defer cleanup()

	// Create test outcome
	project, _ := testDB.CreateProject("Test Project", "", "", "")
	_, err := testDB.CreateOutcome(project.ID, nil, "Test Outcome", "An outcome description", "open")
	if err != nil {
		t.Fatalf("Failed to create test outcome: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/outcomes", nil)
	rr := httptest.NewRecorder()

	ws.handleOutcomes(rr, req)

	var outcomes []Outcome
	json.Unmarshal(rr.Body.Bytes(), &outcomes)

	if len(outcomes) != 1 {
		t.Fatalf("Expected 1 outcome, got %d", len(outcomes))
	}

	if outcomes[0].Title != "Test Outcome" {
		t.Errorf("Expected outcome title 'Test Outcome', got '%s'", outcomes[0].Title)
	}
}

func TestHandleGoalsEmpty(t *testing.T) {
	ws, _, cleanup := setupTestWebServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/goals", nil)
	rr := httptest.NewRecorder()

	ws.handleGoals(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var goals []Goal
	if err := json.Unmarshal(rr.Body.Bytes(), &goals); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if len(goals) != 0 {
		t.Errorf("Expected empty goals array, got %d items", len(goals))
	}
}

func TestHandleGoalsWithData(t *testing.T) {
	ws, testDB, cleanup := setupTestWebServer(t)
	defer cleanup()

	// Create test goal
	_, err := testDB.CreateGoal(nil, nil, "Test Goal", "A goal description", "short_term", "")
	if err != nil {
		t.Fatalf("Failed to create test goal: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/goals", nil)
	rr := httptest.NewRecorder()

	ws.handleGoals(rr, req)

	var goals []Goal
	json.Unmarshal(rr.Body.Bytes(), &goals)

	if len(goals) != 1 {
		t.Fatalf("Expected 1 goal, got %d", len(goals))
	}

	if goals[0].Title != "Test Goal" {
		t.Errorf("Expected goal title 'Test Goal', got '%s'", goals[0].Title)
	}
}

func TestHandleSSE(t *testing.T) {
	ws, _, cleanup := setupTestWebServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/events", nil)
	rr := httptest.NewRecorder()

	// Use a channel to signal when we've received the initial response
	done := make(chan bool)

	go func() {
		// Give the handler time to write headers and initial event
		time.Sleep(100 * time.Millisecond)
		done <- true
	}()

	go func() {
		ws.handleSSE(rr, req)
	}()

	<-done

	// Check headers
	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("Handler returned wrong content type: got %v want text/event-stream", contentType)
	}

	cacheControl := rr.Header().Get("Cache-Control")
	if cacheControl != "no-cache" {
		t.Errorf("Handler returned wrong cache-control: got %v want no-cache", cacheControl)
	}
}

func TestBroadcast(t *testing.T) {
	ws, _, cleanup := setupTestWebServer(t)
	defer cleanup()

	// Create a test client channel
	clientChan := make(chan string, 10)

	ws.clientsMux.Lock()
	ws.clients[clientChan] = true
	ws.clientsMux.Unlock()

	// Broadcast a message
	testData := map[string]string{"test": "data"}
	ws.broadcast("test_event", testData)

	// Check if message was received
	select {
	case msg := <-clientChan:
		if !contains(msg, "event: test_event") {
			t.Errorf("Expected event type in message, got: %s", msg)
		}
		if !contains(msg, `"test":"data"`) {
			t.Errorf("Expected data in message, got: %s", msg)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for broadcast message")
	}

	// Cleanup
	ws.clientsMux.Lock()
	delete(ws.clients, clientChan)
	close(clientChan)
	ws.clientsMux.Unlock()
}

func TestCORSHeaders(t *testing.T) {
	ws, _, cleanup := setupTestWebServer(t)
	defer cleanup()

	endpoints := []string{"/api/projects", "/api/tasks", "/api/problems", "/api/outcomes", "/api/goals"}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := httptest.NewRequest("GET", endpoint, nil)
			rr := httptest.NewRecorder()

			switch endpoint {
			case "/api/projects":
				ws.handleProjects(rr, req)
			case "/api/tasks":
				ws.handleTasks(rr, req)
			case "/api/problems":
				ws.handleProblems(rr, req)
			case "/api/outcomes":
				ws.handleOutcomes(rr, req)
			case "/api/goals":
				ws.handleGoals(rr, req)
			}

			cors := rr.Header().Get("Access-Control-Allow-Origin")
			if cors != "*" {
				t.Errorf("Expected CORS header *, got %s", cors)
			}
		})
	}
}

func TestAPIBaseURL(t *testing.T) {
	ws := NewWebServer(nil, ":8080", ":3000", nil)

	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{"localhost with port", "localhost:3000", "http://localhost:8080"},
		{"ip with port", "192.168.1.1:3000", "http://192.168.1.1:8080"},
		{"hostname with port", "example.com:3000", "http://example.com:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tt.host
			result := ws.apiBaseURL(req)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSeparateServers(t *testing.T) {
	// Create temp directory for test database
	tempDir, err := os.MkdirTemp("", "loom-separate-server-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	testDB, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer testDB.Close()

	ws := NewWebServer(testDB, ":0", ":0", nil)

	// Verify the webAddr and addr are different fields
	if ws.addr == "" {
		t.Error("Expected non-empty API addr")
	}
	if ws.webAddr == "" {
		t.Error("Expected non-empty web addr")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
