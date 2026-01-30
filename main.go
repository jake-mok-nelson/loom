package main

import (
	"context"
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

	// Register problem management tools
	s.AddTool(createProblemTool(), createProblemHandler)
	s.AddTool(listProblemsTool(), listProblemsHandler)
	s.AddTool(getProblemTool(), getProblemHandler)
	s.AddTool(updateProblemTool(), updateProblemHandler)
	s.AddTool(deleteProblemTool(), deleteProblemHandler)

	// Register outcome management tools
	s.AddTool(createOutcomeTool(), createOutcomeHandler)
	s.AddTool(listOutcomesTool(), listOutcomesHandler)
	s.AddTool(getOutcomeTool(), getOutcomeHandler)
	s.AddTool(updateOutcomeTool(), updateOutcomeHandler)
	s.AddTool(deleteOutcomeTool(), deleteOutcomeHandler)

	// Register task note management tools
	s.AddTool(createTaskNoteTool(), createTaskNoteHandler)
	s.AddTool(listTaskNotesTool(), listTaskNotesHandler)
	s.AddTool(getTaskNoteTool(), getTaskNoteHandler)
	s.AddTool(updateTaskNoteTool(), updateTaskNoteHandler)
	s.AddTool(deleteTaskNoteTool(), deleteTaskNoteHandler)

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
				"external_link": map[string]interface{}{
					"type":        "string",
					"description": "External link to ticket system or other tracking tool",
				},
			},
			Required: []string{"name"},
		},
	}
}

func createProjectHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name is required and must be a string"), nil
	}

	description := request.GetString("description", "")
	externalLink := request.GetString("external_link", "")

	project, err := db.CreateProject(name, description, externalLink)
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

func listProjectsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projects, err := db.ListProjects()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list projects: %v", err)), nil
	}

	if len(projects) == 0 {
		return mcp.NewToolResultText("No projects found"), nil
	}

	result := "Projects:\n"
	for _, p := range projects {
		externalLinkStr := ""
		if p.ExternalLink != "" {
			externalLinkStr = fmt.Sprintf(", External Link: %s", p.ExternalLink)
		}
		result += fmt.Sprintf("- ID: %d, Name: %s, Description: %s%s\n", p.ID, p.Name, p.Description, externalLinkStr)
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

func getProjectHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	project, err := db.GetProject(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get project: %v", err)), nil
	}

	result := fmt.Sprintf("Project Details:\nID: %d\nName: %s\nDescription: %s\nExternal Link: %s\nCreated: %s\nUpdated: %s",
		project.ID, project.Name, project.Description, project.ExternalLink, project.CreatedAt, project.UpdatedAt)

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
				"external_link": map[string]interface{}{
					"type":        "string",
					"description": "External link to ticket system or other tracking tool (optional)",
				},
			},
			Required: []string{"id"},
		},
	}
}

func updateProjectHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	arguments := request.GetArguments()

	var name *string
	if n, ok := arguments["name"].(string); ok {
		name = &n
	}

	var description *string
	if desc, ok := arguments["description"].(string); ok {
		description = &desc
	}

	var externalLink *string
	if link, ok := arguments["external_link"].(string); ok {
		externalLink = &link
	}

	if name == nil && description == nil && externalLink == nil {
		return mcp.NewToolResultError("at least one field (name, description, or external_link) must be provided for update"), nil
	}

	project, err := db.UpdateProject(id, name, description, externalLink)
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

func deleteProjectHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
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
				"task_type": map[string]interface{}{
					"type":        "string",
					"description": "Task type (general, chore, investigation, feature, bugfix)",
					"default":     "general",
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
				"external_link": map[string]interface{}{
					"type":        "string",
					"description": "External link to ticket system or other tracking tool",
				},
			},
			Required: []string{"project_id", "title"},
		},
	}
}

func createTaskHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectIDFloat, err := request.RequireFloat("project_id")
	if err != nil {
		return mcp.NewToolResultError("project_id is required and must be a number"), nil
	}
	projectID := int64(projectIDFloat)

	// Check if project exists
	_, err = db.GetProject(projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Project with ID %d does not exist", projectID)), nil
	}

	title, err := request.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("title is required and must be a string"), nil
	}

	description := request.GetString("description", "")
	taskType := request.GetString("task_type", "general")
	if !isValidTaskType(taskType) {
		return mcp.NewToolResultError("task_type must be one of: general, chore, investigation, feature, bugfix"), nil
	}

	status := request.GetString("status", "pending")
	if !isValidStatus(status) {
		return mcp.NewToolResultError("status must be one of: pending, in_progress, completed, blocked"), nil
	}

	priority := request.GetString("priority", "medium")
	if !isValidPriority(priority) {
		return mcp.NewToolResultError("priority must be one of: low, medium, high, urgent"), nil
	}

	externalLink := request.GetString("external_link", "")

	task, err := db.CreateTask(projectID, title, description, status, priority, taskType, externalLink)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create task: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Task created successfully: ID=%d, Title=%s, Status=%s, Type=%s", task.ID, task.Title, task.Status, task.TaskType)), nil
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

func isValidTaskType(taskType string) bool {
	validTypes := map[string]bool{
		"general":       true,
		"chore":         true,
		"investigation": true,
		"feature":       true,
		"bugfix":        true,
	}
	return validTypes[taskType]
}

func isValidProblemStatus(status string) bool {
	validStatuses := map[string]bool{
		"open":        true,
		"in_progress": true,
		"resolved":    true,
		"blocked":     true,
	}
	return validStatuses[status]
}

func isValidOutcomeStatus(status string) bool {
	validStatuses := map[string]bool{
		"open":        true,
		"in_progress": true,
		"completed":   true,
		"blocked":     true,
	}
	return validStatuses[status]
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
				"task_type": map[string]interface{}{
					"type":        "string",
					"description": "Filter by task type (general, chore, investigation, feature, bugfix)",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter by status (pending, in_progress, completed, blocked)",
				},
			},
		},
	}
}

func listTasksHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()

	var projectID *int64
	if idFloat, ok := arguments["project_id"].(float64); ok {
		id := int64(idFloat)
		projectID = &id
	}

	var status *string
	if s, ok := arguments["status"].(string); ok {
		if !isValidStatus(s) {
			return mcp.NewToolResultError("status must be one of: pending, in_progress, completed, blocked"), nil
		}
		status = &s
	}

	var taskType *string
	if t, ok := arguments["task_type"].(string); ok {
		if !isValidTaskType(t) {
			return mcp.NewToolResultError("task_type must be one of: general, chore, investigation, feature, bugfix"), nil
		}
		taskType = &t
	}

	tasks, err := db.ListTasks(projectID, status, taskType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list tasks: %v", err)), nil
	}

	if len(tasks) == 0 {
		return mcp.NewToolResultText("No tasks found"), nil
	}

	result := "Tasks:\n"
	for _, t := range tasks {
		externalLinkStr := ""
		if t.ExternalLink != "" {
			externalLinkStr = fmt.Sprintf(", External Link: %s", t.ExternalLink)
		}
		result += fmt.Sprintf("- ID: %d, ProjectID: %d, Title: %s, Type: %s, Status: %s, Priority: %s%s\n",
			t.ID, t.ProjectID, t.Title, t.TaskType, t.Status, t.Priority, externalLinkStr)
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

func getTaskHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	task, err := db.GetTask(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get task: %v", err)), nil
	}

	result := fmt.Sprintf("Task Details:\nID: %d\nProject ID: %d\nTitle: %s\nDescription: %s\nStatus: %s\nPriority: %s\nTask Type: %s\nExternal Link: %s\nCreated: %s\nUpdated: %s",
		task.ID, task.ProjectID, task.Title, task.Description, task.Status, task.Priority, task.TaskType, task.ExternalLink, task.CreatedAt, task.UpdatedAt)

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
				"task_type": map[string]interface{}{
					"type":        "string",
					"description": "Task type: general, chore, investigation, feature, bugfix (optional)",
				},
				"external_link": map[string]interface{}{
					"type":        "string",
					"description": "External link to ticket system or other tracking tool (optional)",
				},
			},
			Required: []string{"id"},
		},
	}
}

func updateTaskHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	arguments := request.GetArguments()

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

	var taskType *string
	if t, ok := arguments["task_type"].(string); ok {
		if !isValidTaskType(t) {
			return mcp.NewToolResultError("task_type must be one of: general, chore, investigation, feature, bugfix"), nil
		}
		taskType = &t
	}

	var externalLink *string
	if link, ok := arguments["external_link"].(string); ok {
		externalLink = &link
	}

	if title == nil && description == nil && status == nil && priority == nil && taskType == nil && externalLink == nil {
		return mcp.NewToolResultError("at least one field (title, description, status, priority, task_type, or external_link) must be provided for update"), nil
	}

	task, err := db.UpdateTask(id, title, description, status, priority, taskType, externalLink)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update task: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Task updated successfully: ID=%d, Title=%s, Status=%s, Type=%s", task.ID, task.Title, task.Status, task.TaskType)), nil
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

func deleteTaskHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	if err := db.DeleteTask(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete task: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Task %d deleted successfully", id)), nil
}

// Problem management tools

func createProblemTool() mcp.Tool {
	return mcp.Tool{
		Name:        "create_problem",
		Description: "Create a new problem linked to work in a project",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "number",
					"description": "Project ID",
				},
				"task_id": map[string]interface{}{
					"type":        "number",
					"description": "Optional task ID to link the problem to specific work",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Problem title",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Problem description",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Problem status (open, in_progress, resolved, blocked)",
					"default":     "open",
				},
			},
			Required: []string{"project_id", "title"},
		},
	}
}

func createProblemHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectIDFloat, err := request.RequireFloat("project_id")
	if err != nil {
		return mcp.NewToolResultError("project_id is required and must be a number"), nil
	}
	projectID := int64(projectIDFloat)

	_, err = db.GetProject(projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Project with ID %d does not exist", projectID)), nil
	}

	arguments := request.GetArguments()
	var taskID *int64
	if taskIDFloat, ok := arguments["task_id"].(float64); ok {
		taskIDValue := int64(taskIDFloat)
		task, err := db.GetTask(taskIDValue)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Task with ID %d does not exist", taskIDValue)), nil
		}
		if task.ProjectID != projectID {
			return mcp.NewToolResultError("task_id must belong to the same project"), nil
		}
		taskID = &taskIDValue
	}

	title, err := request.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("title is required and must be a string"), nil
	}

	description := request.GetString("description", "")
	status := request.GetString("status", "open")
	if !isValidProblemStatus(status) {
		return mcp.NewToolResultError("status must be one of: open, in_progress, resolved, blocked"), nil
	}

	problem, err := db.CreateProblem(projectID, taskID, title, description, status)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create problem: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Problem created successfully: ID=%d, Title=%s, Status=%s", problem.ID, problem.Title, problem.Status)), nil
}

func listProblemsTool() mcp.Tool {
	return mcp.Tool{
		Name:        "list_problems",
		Description: "List problems, optionally filtered by project, task, and status",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "number",
					"description": "Filter by project ID",
				},
				"task_id": map[string]interface{}{
					"type":        "number",
					"description": "Filter by task ID",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter by status (open, in_progress, resolved, blocked)",
				},
			},
		},
	}
}

func listProblemsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()

	var projectID *int64
	if idFloat, ok := arguments["project_id"].(float64); ok {
		id := int64(idFloat)
		projectID = &id
	}

	var taskID *int64
	if taskIDFloat, ok := arguments["task_id"].(float64); ok {
		id := int64(taskIDFloat)
		taskID = &id
	}

	var status *string
	if s, ok := arguments["status"].(string); ok {
		if !isValidProblemStatus(s) {
			return mcp.NewToolResultError("status must be one of: open, in_progress, resolved, blocked"), nil
		}
		status = &s
	}

	problems, err := db.ListProblems(projectID, taskID, status)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list problems: %v", err)), nil
	}

	if len(problems) == 0 {
		return mcp.NewToolResultText("No problems found"), nil
	}

	result := "Problems:\n"
	for _, p := range problems {
		taskInfo := "none"
		if p.TaskID != nil {
			taskInfo = fmt.Sprintf("%d", *p.TaskID)
		}
		result += fmt.Sprintf("- ID: %d, ProjectID: %d, TaskID: %s, Title: %s, Status: %s\n",
			p.ID, p.ProjectID, taskInfo, p.Title, p.Status)
	}

	return mcp.NewToolResultText(result), nil
}

func getProblemTool() mcp.Tool {
	return mcp.Tool{
		Name:        "get_problem",
		Description: "Get details of a specific problem",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Problem ID",
				},
			},
			Required: []string{"id"},
		},
	}
}

func getProblemHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	problem, err := db.GetProblem(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get problem: %v", err)), nil
	}

	taskInfo := "none"
	if problem.TaskID != nil {
		taskInfo = fmt.Sprintf("%d", *problem.TaskID)
	}
	result := fmt.Sprintf("Problem Details:\nID: %d\nProject ID: %d\nTask ID: %s\nTitle: %s\nDescription: %s\nStatus: %s\nCreated: %s\nUpdated: %s",
		problem.ID, problem.ProjectID, taskInfo, problem.Title, problem.Description, problem.Status, problem.CreatedAt, problem.UpdatedAt)

	return mcp.NewToolResultText(result), nil
}

func updateProblemTool() mcp.Tool {
	return mcp.Tool{
		Name:        "update_problem",
		Description: "Update an existing problem",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Problem ID",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Problem title (optional)",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Problem description (optional)",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Problem status: open, in_progress, resolved, blocked (optional)",
				},
			},
			Required: []string{"id"},
		},
	}
}

func updateProblemHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	arguments := request.GetArguments()

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
		if !isValidProblemStatus(s) {
			return mcp.NewToolResultError("status must be one of: open, in_progress, resolved, blocked"), nil
		}
		status = &s
	}

	if title == nil && description == nil && status == nil {
		return mcp.NewToolResultError("at least one field (title, description, or status) must be provided for update"), nil
	}

	problem, err := db.UpdateProblem(id, title, description, status)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update problem: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Problem updated successfully: ID=%d, Title=%s, Status=%s", problem.ID, problem.Title, problem.Status)), nil
}

func deleteProblemTool() mcp.Tool {
	return mcp.Tool{
		Name:        "delete_problem",
		Description: "Delete a problem",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Problem ID",
				},
			},
			Required: []string{"id"},
		},
	}
}

func deleteProblemHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	if err := db.DeleteProblem(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete problem: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Problem %d deleted successfully", id)), nil
}

// Outcome management tools

func createOutcomeTool() mcp.Tool {
	return mcp.Tool{
		Name:        "create_outcome",
		Description: "Create a new outcome connected to a project and optionally a task",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "number",
					"description": "Project ID",
				},
				"task_id": map[string]interface{}{
					"type":        "number",
					"description": "Optional task ID to link the outcome to work",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Outcome title",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Outcome description",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Outcome status (open, in_progress, completed, blocked)",
					"default":     "open",
				},
			},
			Required: []string{"project_id", "title"},
		},
	}
}

func createOutcomeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectIDFloat, err := request.RequireFloat("project_id")
	if err != nil {
		return mcp.NewToolResultError("project_id is required and must be a number"), nil
	}
	projectID := int64(projectIDFloat)

	_, err = db.GetProject(projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Project with ID %d does not exist", projectID)), nil
	}

	arguments := request.GetArguments()
	var taskID *int64
	if taskIDFloat, ok := arguments["task_id"].(float64); ok {
		taskIDValue := int64(taskIDFloat)
		task, err := db.GetTask(taskIDValue)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Task with ID %d does not exist", taskIDValue)), nil
		}
		if task.ProjectID != projectID {
			return mcp.NewToolResultError("task_id must belong to the same project"), nil
		}
		taskID = &taskIDValue
	}

	title, err := request.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("title is required and must be a string"), nil
	}

	description := request.GetString("description", "")
	status := request.GetString("status", "open")
	if !isValidOutcomeStatus(status) {
		return mcp.NewToolResultError("status must be one of: open, in_progress, completed, blocked"), nil
	}

	outcome, err := db.CreateOutcome(projectID, taskID, title, description, status)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create outcome: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Outcome created successfully: ID=%d, Title=%s, Status=%s", outcome.ID, outcome.Title, outcome.Status)), nil
}

func listOutcomesTool() mcp.Tool {
	return mcp.Tool{
		Name:        "list_outcomes",
		Description: "List outcomes, optionally filtered by project, task, and status",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "number",
					"description": "Filter by project ID",
				},
				"task_id": map[string]interface{}{
					"type":        "number",
					"description": "Filter by task ID",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter by status (open, in_progress, completed, blocked)",
				},
			},
		},
	}
}

func listOutcomesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()

	var projectID *int64
	if idFloat, ok := arguments["project_id"].(float64); ok {
		id := int64(idFloat)
		projectID = &id
	}

	var taskID *int64
	if taskIDFloat, ok := arguments["task_id"].(float64); ok {
		id := int64(taskIDFloat)
		taskID = &id
	}

	var status *string
	if s, ok := arguments["status"].(string); ok {
		if !isValidOutcomeStatus(s) {
			return mcp.NewToolResultError("status must be one of: open, in_progress, completed, blocked"), nil
		}
		status = &s
	}

	outcomes, err := db.ListOutcomes(projectID, taskID, status)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list outcomes: %v", err)), nil
	}

	if len(outcomes) == 0 {
		return mcp.NewToolResultText("No outcomes found"), nil
	}

	result := "Outcomes:\n"
	for _, o := range outcomes {
		taskInfo := "none"
		if o.TaskID != nil {
			taskInfo = fmt.Sprintf("%d", *o.TaskID)
		}
		result += fmt.Sprintf("- ID: %d, ProjectID: %d, TaskID: %s, Title: %s, Status: %s\n",
			o.ID, o.ProjectID, taskInfo, o.Title, o.Status)
	}

	return mcp.NewToolResultText(result), nil
}

func getOutcomeTool() mcp.Tool {
	return mcp.Tool{
		Name:        "get_outcome",
		Description: "Get details of a specific outcome",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Outcome ID",
				},
			},
			Required: []string{"id"},
		},
	}
}

func getOutcomeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	outcome, err := db.GetOutcome(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get outcome: %v", err)), nil
	}

	taskInfo := "none"
	if outcome.TaskID != nil {
		taskInfo = fmt.Sprintf("%d", *outcome.TaskID)
	}
	result := fmt.Sprintf("Outcome Details:\nID: %d\nProject ID: %d\nTask ID: %s\nTitle: %s\nDescription: %s\nStatus: %s\nCreated: %s\nUpdated: %s",
		outcome.ID, outcome.ProjectID, taskInfo, outcome.Title, outcome.Description, outcome.Status, outcome.CreatedAt, outcome.UpdatedAt)

	return mcp.NewToolResultText(result), nil
}

func updateOutcomeTool() mcp.Tool {
	return mcp.Tool{
		Name:        "update_outcome",
		Description: "Update an existing outcome",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Outcome ID",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Outcome title (optional)",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Outcome description (optional)",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Outcome status: open, in_progress, completed, blocked (optional)",
				},
			},
			Required: []string{"id"},
		},
	}
}

func updateOutcomeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	arguments := request.GetArguments()

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
		if !isValidOutcomeStatus(s) {
			return mcp.NewToolResultError("status must be one of: open, in_progress, completed, blocked"), nil
		}
		status = &s
	}

	if title == nil && description == nil && status == nil {
		return mcp.NewToolResultError("at least one field (title, description, or status) must be provided for update"), nil
	}

	outcome, err := db.UpdateOutcome(id, title, description, status)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update outcome: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Outcome updated successfully: ID=%d, Title=%s, Status=%s", outcome.ID, outcome.Title, outcome.Status)), nil
}

func deleteOutcomeTool() mcp.Tool {
	return mcp.Tool{
		Name:        "delete_outcome",
		Description: "Delete an outcome",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Outcome ID",
				},
			},
			Required: []string{"id"},
		},
	}
}

func deleteOutcomeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	if err := db.DeleteOutcome(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete outcome: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Outcome %d deleted successfully", id)), nil
}

// Task note management tools

func createTaskNoteTool() mcp.Tool {
	return mcp.Tool{
		Name:        "create_task_note",
		Description: "Create a note on a task",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"task_id": map[string]interface{}{
					"type":        "number",
					"description": "Task ID",
				},
				"note": map[string]interface{}{
					"type":        "string",
					"description": "Note text",
				},
			},
			Required: []string{"task_id", "note"},
		},
	}
}

func createTaskNoteHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskIDFloat, err := request.RequireFloat("task_id")
	if err != nil {
		return mcp.NewToolResultError("task_id is required and must be a number"), nil
	}
	taskID := int64(taskIDFloat)

	_, err = db.GetTask(taskID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Task with ID %d does not exist", taskID)), nil
	}

	note, err := request.RequireString("note")
	if err != nil {
		return mcp.NewToolResultError("note is required and must be a string"), nil
	}

	taskNote, err := db.CreateTaskNote(taskID, note)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create task note: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Task note created successfully: ID=%d, TaskID=%d", taskNote.ID, taskNote.TaskID)), nil
}

func listTaskNotesTool() mcp.Tool {
	return mcp.Tool{
		Name:        "list_task_notes",
		Description: "List notes for a task",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"task_id": map[string]interface{}{
					"type":        "number",
					"description": "Task ID",
				},
			},
			Required: []string{"task_id"},
		},
	}
}

func listTaskNotesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskIDFloat, err := request.RequireFloat("task_id")
	if err != nil {
		return mcp.NewToolResultError("task_id is required and must be a number"), nil
	}
	taskID := int64(taskIDFloat)

	notes, err := db.ListTaskNotes(taskID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list task notes: %v", err)), nil
	}

	if len(notes) == 0 {
		return mcp.NewToolResultText("No task notes found"), nil
	}

	result := "Task Notes:\n"
	for _, note := range notes {
		result += fmt.Sprintf("- ID: %d, TaskID: %d, Note: %s\n", note.ID, note.TaskID, note.Note)
	}

	return mcp.NewToolResultText(result), nil
}

func getTaskNoteTool() mcp.Tool {
	return mcp.Tool{
		Name:        "get_task_note",
		Description: "Get details of a specific task note",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Task note ID",
				},
			},
			Required: []string{"id"},
		},
	}
}

func getTaskNoteHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	note, err := db.GetTaskNote(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get task note: %v", err)), nil
	}

	result := fmt.Sprintf("Task Note Details:\nID: %d\nTask ID: %d\nNote: %s\nCreated: %s\nUpdated: %s",
		note.ID, note.TaskID, note.Note, note.CreatedAt, note.UpdatedAt)

	return mcp.NewToolResultText(result), nil
}

func updateTaskNoteTool() mcp.Tool {
	return mcp.Tool{
		Name:        "update_task_note",
		Description: "Update an existing task note",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Task note ID",
				},
				"note": map[string]interface{}{
					"type":        "string",
					"description": "Note text",
				},
			},
			Required: []string{"id", "note"},
		},
	}
}

func updateTaskNoteHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	note, err := request.RequireString("note")
	if err != nil {
		return mcp.NewToolResultError("note is required and must be a string"), nil
	}

	taskNote, err := db.UpdateTaskNote(id, note)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update task note: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Task note updated successfully: ID=%d, TaskID=%d", taskNote.ID, taskNote.TaskID)), nil
}

func deleteTaskNoteTool() mcp.Tool {
	return mcp.Tool{
		Name:        "delete_task_note",
		Description: "Delete a task note",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "number",
					"description": "Task note ID",
				},
			},
			Required: []string{"id"},
		},
	}
}

func deleteTaskNoteHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat, err := request.RequireFloat("id")
	if err != nil {
		return mcp.NewToolResultError("id is required and must be a number"), nil
	}
	id := int64(idFloat)

	if err := db.DeleteTaskNote(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete task note: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Task note %d deleted successfully", id)), nil
}
