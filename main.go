package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var db *Database

func main() {
	// Determine database path
	dbPath := os.Getenv("LOOM_DB_PATH")
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to get home directory:", err)
		}
		dbPath = filepath.Join(homeDir, ".loom", "loom.db")
	}

	// Initialize database
	var err error
	db, err = NewDatabase(dbPath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	log.Printf("Loom MCP server starting with database at: %s", dbPath)

	// Create MCP server
	s := server.NewMCPServer(
		"Loom",
		"1.0.0",
	)

	// Register project management tools
	s.AddTool(createProjectTool(), createProjectHandler)
	s.AddTool(listProjectsTool(), listProjectsHandler)
	s.AddTool(getProjectTool(), getProjectHandler)
	s.AddTool(updateProjectTool(), updateProjectHandler)
	s.AddTool(deleteProjectTool(), deleteProjectHandler)

	// Register task management tools
	s.AddTool(createTaskTool(), createTaskHandler)
	s.AddTool(listTasksTool(), listTasksHandler)
	s.AddTool(getTaskTool(), getTaskHandler)
	s.AddTool(updateTaskTool(), updateTaskHandler)
	s.AddTool(deleteTaskTool(), deleteTaskHandler)

	// Start server
	if err := server.ServeStdio(s); err != nil {
		log.Fatal(err)
	}
}

// Project management tools

func createProjectTool() mcp.Tool {
	return mcp.Tool{
		Name:        "create_project",
		Description: "Create a new project in Loom",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Project name",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Project description",
				},
			},
			Required: []string{"name"},
		},
	}
}

func createProjectHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	name, ok := arguments["name"].(string)
	if !ok {
		return mcp.NewToolResultError("name is required and must be a string"), nil
	}

	description := ""
	if desc, ok := arguments["description"].(string); ok {
		description = desc
	}

	project, err := db.CreateProject(name, description)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create project: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Project created successfully: ID=%d, Name=%s", project.ID, project.Name)), nil
}

func listProjectsTool() mcp.Tool {
	return mcp.Tool{
		Name:        "list_projects",
		Description: "List all projects in Loom",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}
}

func listProjectsHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	projects, err := db.ListProjects()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list projects: %v", err)), nil
	}

	if len(projects) == 0 {
		return mcp.NewToolResultText("No projects found"), nil
	}

	result := "Projects:\n"
	for _, p := range projects {
		result += fmt.Sprintf("- ID: %d, Name: %s, Description: %s\n", p.ID, p.Name, p.Description)
	}

	return mcp.NewToolResultText(result), nil
}

func getProjectTool() mcp.Tool {
	return mcp.Tool{
		Name:        "get_project",
		Description: "Get details of a specific project",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Project ID",
				},
			},
			Required: []string{"id"},
		},
	}
}

func getProjectHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	idFloat, ok := arguments["id"].(float64)
	if !ok {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	project, err := db.GetProject(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get project: %v", err)), nil
	}

	result := fmt.Sprintf("Project Details:\nID: %d\nName: %s\nDescription: %s\nCreated: %s\nUpdated: %s",
		project.ID, project.Name, project.Description, project.CreatedAt, project.UpdatedAt)

	return mcp.NewToolResultText(result), nil
}

func updateProjectTool() mcp.Tool {
	return mcp.Tool{
		Name:        "update_project",
		Description: "Update an existing project",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Project ID",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Project name (optional)",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Project description (optional)",
				},
			},
			Required: []string{"id"},
		},
	}
}

func updateProjectHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	idFloat, ok := arguments["id"].(float64)
	if !ok {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	var name *string
	if n, ok := arguments["name"].(string); ok {
		name = &n
	}

	var description *string
	if desc, ok := arguments["description"].(string); ok {
		description = &desc
	}

	if name == nil && description == nil {
		return mcp.NewToolResultError("at least one field (name or description) must be provided for update"), nil
	}

	project, err := db.UpdateProject(id, name, description)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update project: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Project updated successfully: ID=%d, Name=%s", project.ID, project.Name)), nil
}

func deleteProjectTool() mcp.Tool {
	return mcp.Tool{
		Name:        "delete_project",
		Description: "Delete a project and all its tasks",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Project ID",
				},
			},
			Required: []string{"id"},
		},
	}
}

func deleteProjectHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	idFloat, ok := arguments["id"].(float64)
	if !ok {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	if err := db.DeleteProject(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete project: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Project %d deleted successfully", id)), nil
}

// Task management tools

func createTaskTool() mcp.Tool {
	return mcp.Tool{
		Name:        "create_task",
		Description: "Create a new task in a project",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "number",
					"description": "Project ID",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Task title",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Task description",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Task status (pending, in_progress, completed, blocked)",
					"default":     "pending",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"description": "Task priority (low, medium, high, urgent)",
					"default":     "medium",
				},
			},
			Required: []string{"project_id", "title"},
		},
	}
}

func createTaskHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	projectIDFloat, ok := arguments["project_id"].(float64)
	if !ok {
		return mcp.NewToolResultError("project_id is required and must be a number"), nil
	}
	projectID := int64(projectIDFloat)

	// Check if project exists
	_, err := db.GetProject(projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Project with ID %d does not exist", projectID)), nil
	}

	title, ok := arguments["title"].(string)
	if !ok {
		return mcp.NewToolResultError("title is required and must be a string"), nil
	}

	description := ""
	if desc, ok := arguments["description"].(string); ok {
		description = desc
	}

	status := "pending"
	if s, ok := arguments["status"].(string); ok {
		if !isValidStatus(s) {
			return mcp.NewToolResultError("status must be one of: pending, in_progress, completed, blocked"), nil
		}
		status = s
	}

	priority := "medium"
	if p, ok := arguments["priority"].(string); ok {
		if !isValidPriority(p) {
			return mcp.NewToolResultError("priority must be one of: low, medium, high, urgent"), nil
		}
		priority = p
	}

	task, err := db.CreateTask(projectID, title, description, status, priority)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create task: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Task created successfully: ID=%d, Title=%s, Status=%s", task.ID, task.Title, task.Status)), nil
}

func isValidStatus(status string) bool {
	validStatuses := map[string]bool{
		"pending":     true,
		"in_progress": true,
		"completed":   true,
		"blocked":     true,
	}
	return validStatuses[status]
}

func isValidPriority(priority string) bool {
	validPriorities := map[string]bool{
		"low":    true,
		"medium": true,
		"high":   true,
		"urgent": true,
	}
	return validPriorities[priority]
}

func listTasksTool() mcp.Tool {
	return mcp.Tool{
		Name:        "list_tasks",
		Description: "List tasks, optionally filtered by project and/or status",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "number",
					"description": "Filter by project ID",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter by status (pending, in_progress, completed, blocked)",
				},
			},
		},
	}
}

func listTasksHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	var projectID *int64
	if idFloat, ok := arguments["project_id"].(float64); ok {
		id := int64(idFloat)
		projectID = &id
	}

	var status *string
	if s, ok := arguments["status"].(string); ok {
		status = &s
	}

	tasks, err := db.ListTasks(projectID, status)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list tasks: %v", err)), nil
	}

	if len(tasks) == 0 {
		return mcp.NewToolResultText("No tasks found"), nil
	}

	result := "Tasks:\n"
	for _, t := range tasks {
		result += fmt.Sprintf("- ID: %d, ProjectID: %d, Title: %s, Status: %s, Priority: %s\n",
			t.ID, t.ProjectID, t.Title, t.Status, t.Priority)
	}

	return mcp.NewToolResultText(result), nil
}

func getTaskTool() mcp.Tool {
	return mcp.Tool{
		Name:        "get_task",
		Description: "Get details of a specific task",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Task ID",
				},
			},
			Required: []string{"id"},
		},
	}
}

func getTaskHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	idFloat, ok := arguments["id"].(float64)
	if !ok {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	task, err := db.GetTask(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get task: %v", err)), nil
	}

	result := fmt.Sprintf("Task Details:\nID: %d\nProject ID: %d\nTitle: %s\nDescription: %s\nStatus: %s\nPriority: %s\nCreated: %s\nUpdated: %s",
		task.ID, task.ProjectID, task.Title, task.Description, task.Status, task.Priority, task.CreatedAt, task.UpdatedAt)

	return mcp.NewToolResultText(result), nil
}

func updateTaskTool() mcp.Tool {
	return mcp.Tool{
		Name:        "update_task",
		Description: "Update an existing task",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Task ID",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Task title (optional)",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Task description (optional)",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Task status: pending, in_progress, completed, blocked (optional)",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"description": "Task priority: low, medium, high, urgent (optional)",
				},
			},
			Required: []string{"id"},
		},
	}
}

func updateTaskHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	idFloat, ok := arguments["id"].(float64)
	if !ok {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	var title *string
	if t, ok := arguments["title"].(string); ok {
		title = &t
	}

	var description *string
	if desc, ok := arguments["description"].(string); ok {
		description = &desc
	}

	var status *string
	if s, ok := arguments["status"].(string); ok {
		if !isValidStatus(s) {
			return mcp.NewToolResultError("status must be one of: pending, in_progress, completed, blocked"), nil
		}
		status = &s
	}

	var priority *string
	if p, ok := arguments["priority"].(string); ok {
		if !isValidPriority(p) {
			return mcp.NewToolResultError("priority must be one of: low, medium, high, urgent"), nil
		}
		priority = &p
	}

	if title == nil && description == nil && status == nil && priority == nil {
		return mcp.NewToolResultError("at least one field (title, description, status, or priority) must be provided for update"), nil
	}

	task, err := db.UpdateTask(id, title, description, status, priority)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update task: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Task updated successfully: ID=%d, Title=%s, Status=%s", task.ID, task.Title, task.Status)), nil
}

func deleteTaskTool() mcp.Tool {
	return mcp.Tool{
		Name:        "delete_task",
		Description: "Delete a task",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Task ID",
				},
			},
			Required: []string{"id"},
		},
	}
}

func deleteTaskHandler(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	idFloat, ok := arguments["id"].(float64)
	if !ok {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	if err := db.DeleteTask(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete task: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Task %d deleted successfully", id)), nil
}
