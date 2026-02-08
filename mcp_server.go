package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewMCPServer creates a new MCP server with all Loom tools registered.
func NewMCPServer(database *Database, announceFunc func(string)) *server.MCPServer {
	s := server.NewMCPServer(
		"Loom",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s.AddTools(projectTools(database, announceFunc)...)
	s.AddTools(taskTools(database, announceFunc)...)
	s.AddTools(problemTools(database, announceFunc)...)
	s.AddTools(outcomeTools(database, announceFunc)...)
	s.AddTools(goalTools(database, announceFunc)...)
	s.AddTools(taskNoteTools(database, announceFunc)...)
	s.AddTools(summaryTools(database)...)

	return s
}

// NewMCPHandler creates a new MCP Streamable HTTP handler that can be
// mounted on an existing HTTP server mux at the "/sse" path.
func NewMCPHandler(mcpServer *server.MCPServer) *server.StreamableHTTPServer {
	return server.NewStreamableHTTPServer(mcpServer)
}

// --- Project Tools ---

func projectTools(db *Database, announceFunc func(string)) []server.ServerTool {
	return []server.ServerTool{
		{
			Tool: mcp.NewTool("create_project",
				mcp.WithDescription("Create a new project in Loom"),
				mcp.WithString("name", mcp.Required(), mcp.Description("Project name")),
				mcp.WithString("description", mcp.Description("Project description")),
				mcp.WithString("status", mcp.Description("Project status (e.g. active, planning, on_hold, completed, archived)")),
				mcp.WithString("external_link", mcp.Description("External link URL")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				name, err := req.RequireString("name")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				description := req.GetString("description", "")
				status := req.GetString("status", "")
				externalLink := req.GetString("external_link", "")

				project, err := db.CreateProject(name, description, status, externalLink)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to create project: %v", err)), nil
				}
				announceFunc(fmt.Sprintf("Project %s created", name))
				return jsonToolResult(project)
			},
		},
		{
			Tool: mcp.NewTool("list_projects",
				mcp.WithDescription("List all projects in Loom, optionally filtered by status"),
				mcp.WithString("status", mcp.Description("Filter by status (e.g. active, planning, on_hold, completed, archived)")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				status := optionalString(req, "status")
				projects, err := db.ListProjects(status)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list projects: %v", err)), nil
				}
				if projects == nil {
					projects = []*Project{}
				}
				return jsonToolResult(projects)
			},
		},
		{
			Tool: mcp.NewTool("get_project",
				mcp.WithDescription("Get details of a specific project"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Project ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				project, err := db.GetProject(int64(id))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to get project: %v", err)), nil
				}
				return jsonToolResult(project)
			},
		},
		{
			Tool: mcp.NewTool("update_project",
				mcp.WithDescription("Update an existing project"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Project ID")),
				mcp.WithString("name", mcp.Description("New project name")),
				mcp.WithString("description", mcp.Description("New project description")),
				mcp.WithString("status", mcp.Description("New project status")),
				mcp.WithString("external_link", mcp.Description("New external link URL")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				name := optionalString(req, "name")
				description := optionalString(req, "description")
				status := optionalString(req, "status")
				externalLink := optionalString(req, "external_link")

				project, err := db.UpdateProject(int64(id), name, description, status, externalLink)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to update project: %v", err)), nil
				}
				return jsonToolResult(project)
			},
		},
		{
			Tool: mcp.NewTool("delete_project",
				mcp.WithDescription("Delete a project and all its tasks"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Project ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := db.DeleteProject(int64(id)); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to delete project: %v", err)), nil
				}
				return mcp.NewToolResultText("project deleted successfully"), nil
			},
		},
	}
}

// --- Task Tools ---

func taskTools(db *Database, announceFunc func(string)) []server.ServerTool {
	return []server.ServerTool{
		{
			Tool: mcp.NewTool("create_task",
				mcp.WithDescription("Create a new task in a project"),
				mcp.WithNumber("project_id", mcp.Required(), mcp.Description("Project ID")),
				mcp.WithString("title", mcp.Required(), mcp.Description("Task title")),
				mcp.WithString("description", mcp.Description("Task description")),
				mcp.WithString("status", mcp.Description("Task status (e.g. pending, in_progress, completed)")),
				mcp.WithString("priority", mcp.Description("Task priority (e.g. high, medium, low)")),
				mcp.WithString("task_type", mcp.Description("Task type (e.g. feature, bugfix, chore)")),
				mcp.WithString("external_link", mcp.Description("External link URL")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectID, err := req.RequireFloat("project_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				title, err := req.RequireString("title")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				description := req.GetString("description", "")
				status := req.GetString("status", "")
				priority := req.GetString("priority", "")
				taskType := req.GetString("task_type", "")
				externalLink := req.GetString("external_link", "")

				task, err := db.CreateTask(int64(projectID), title, description, status, priority, taskType, externalLink)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to create task: %v", err)), nil
				}
				announceFunc(fmt.Sprintf("Task %s created", title))
				return jsonToolResult(task)
			},
		},
		{
			Tool: mcp.NewTool("list_tasks",
				mcp.WithDescription("List tasks, optionally filtered by project and/or status"),
				mcp.WithNumber("project_id", mcp.Description("Filter by project ID")),
				mcp.WithString("status", mcp.Description("Filter by status")),
				mcp.WithString("task_type", mcp.Description("Filter by task type")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectID := optionalInt64(req, "project_id")
				status := optionalString(req, "status")
				taskType := optionalString(req, "task_type")

				tasks, err := db.ListTasks(projectID, status, taskType)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list tasks: %v", err)), nil
				}
				if tasks == nil {
					tasks = []*Task{}
				}
				return jsonToolResult(tasks)
			},
		},
		{
			Tool: mcp.NewTool("get_task",
				mcp.WithDescription("Get details of a specific task"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Task ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				task, err := db.GetTask(int64(id))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to get task: %v", err)), nil
				}
				return jsonToolResult(task)
			},
		},
		{
			Tool: mcp.NewTool("update_task",
				mcp.WithDescription("Update an existing task"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Task ID")),
				mcp.WithString("title", mcp.Description("New task title")),
				mcp.WithString("description", mcp.Description("New task description")),
				mcp.WithString("status", mcp.Description("New task status")),
				mcp.WithString("priority", mcp.Description("New task priority")),
				mcp.WithString("task_type", mcp.Description("New task type")),
				mcp.WithString("external_link", mcp.Description("New external link URL")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				title := optionalString(req, "title")
				description := optionalString(req, "description")
				status := optionalString(req, "status")
				priority := optionalString(req, "priority")
				taskType := optionalString(req, "task_type")
				externalLink := optionalString(req, "external_link")

				task, err := db.UpdateTask(int64(id), title, description, status, priority, taskType, externalLink)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to update task: %v", err)), nil
				}
				return jsonToolResult(task)
			},
		},
		{
			Tool: mcp.NewTool("delete_task",
				mcp.WithDescription("Delete a task"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Task ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := db.DeleteTask(int64(id)); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to delete task: %v", err)), nil
				}
				return mcp.NewToolResultText("task deleted successfully"), nil
			},
		},
	}
}

// --- Problem Tools ---

func problemTools(db *Database, announceFunc func(string)) []server.ServerTool {
	return []server.ServerTool{
		{
			Tool: mcp.NewTool("create_problem",
				mcp.WithDescription("Create a new problem with optional project or task links and assignee"),
				mcp.WithString("title", mcp.Required(), mcp.Description("Problem title")),
				mcp.WithString("description", mcp.Description("Problem description")),
				mcp.WithString("status", mcp.Description("Problem status (e.g. open, resolved)")),
				mcp.WithString("assignee", mcp.Description("Assignee name")),
				mcp.WithNumber("project_id", mcp.Description("Linked project ID")),
				mcp.WithNumber("task_id", mcp.Description("Linked task ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				title, err := req.RequireString("title")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				description := req.GetString("description", "")
				status := req.GetString("status", "")
				assignee := req.GetString("assignee", "")
				projectID := optionalInt64(req, "project_id")
				taskID := optionalInt64(req, "task_id")

				problem, err := db.CreateProblem(projectID, taskID, title, description, status, assignee)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to create problem: %v", err)), nil
				}
				announceFunc(fmt.Sprintf("Problem %s created", title))
				return jsonToolResult(problem)
			},
		},
		{
			Tool: mcp.NewTool("list_problems",
				mcp.WithDescription("List problems, optionally filtered by project, task, status, and assignee"),
				mcp.WithNumber("project_id", mcp.Description("Filter by project ID")),
				mcp.WithNumber("task_id", mcp.Description("Filter by task ID")),
				mcp.WithString("status", mcp.Description("Filter by status")),
				mcp.WithString("assignee", mcp.Description("Filter by assignee")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectID := optionalInt64(req, "project_id")
				taskID := optionalInt64(req, "task_id")
				status := optionalString(req, "status")
				assignee := optionalString(req, "assignee")

				problems, err := db.ListProblems(projectID, taskID, status, assignee)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list problems: %v", err)), nil
				}
				if problems == nil {
					problems = []*Problem{}
				}
				return jsonToolResult(problems)
			},
		},
		{
			Tool: mcp.NewTool("get_problem",
				mcp.WithDescription("Get details of a specific problem"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Problem ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				problem, err := db.GetProblem(int64(id))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to get problem: %v", err)), nil
				}
				return jsonToolResult(problem)
			},
		},
		{
			Tool: mcp.NewTool("update_problem",
				mcp.WithDescription("Update an existing problem including assignee"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Problem ID")),
				mcp.WithString("title", mcp.Description("New problem title")),
				mcp.WithString("description", mcp.Description("New problem description")),
				mcp.WithString("status", mcp.Description("New problem status")),
				mcp.WithString("assignee", mcp.Description("New assignee")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				title := optionalString(req, "title")
				description := optionalString(req, "description")
				status := optionalString(req, "status")
				assignee := optionalString(req, "assignee")

				problem, err := db.UpdateProblem(int64(id), title, description, status, assignee)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to update problem: %v", err)), nil
				}
				return jsonToolResult(problem)
			},
		},
		{
			Tool: mcp.NewTool("delete_problem",
				mcp.WithDescription("Delete a problem"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Problem ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := db.DeleteProblem(int64(id)); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to delete problem: %v", err)), nil
				}
				return mcp.NewToolResultText("problem deleted successfully"), nil
			},
		},
		{
			Tool: mcp.NewTool("link_problem_to_project",
				mcp.WithDescription("Link a problem to an additional project (many-to-many relationship)"),
				mcp.WithNumber("problem_id", mcp.Required(), mcp.Description("Problem ID")),
				mcp.WithNumber("project_id", mcp.Required(), mcp.Description("Project ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				problemID, err := req.RequireFloat("problem_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				projectID, err := req.RequireFloat("project_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := db.LinkProblemToProject(int64(problemID), int64(projectID)); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to link problem to project: %v", err)), nil
				}
				return mcp.NewToolResultText("problem linked to project successfully"), nil
			},
		},
		{
			Tool: mcp.NewTool("unlink_problem_from_project",
				mcp.WithDescription("Remove a problem's link to a project"),
				mcp.WithNumber("problem_id", mcp.Required(), mcp.Description("Problem ID")),
				mcp.WithNumber("project_id", mcp.Required(), mcp.Description("Project ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				problemID, err := req.RequireFloat("problem_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				projectID, err := req.RequireFloat("project_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := db.UnlinkProblemFromProject(int64(problemID), int64(projectID)); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to unlink problem from project: %v", err)), nil
				}
				return mcp.NewToolResultText("problem unlinked from project successfully"), nil
			},
		},
		{
			Tool: mcp.NewTool("get_problem_projects",
				mcp.WithDescription("Get all projects linked to a problem"),
				mcp.WithNumber("problem_id", mcp.Required(), mcp.Description("Problem ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				problemID, err := req.RequireFloat("problem_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				projects, err := db.GetProblemProjects(int64(problemID))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to get problem projects: %v", err)), nil
				}
				if projects == nil {
					projects = []*Project{}
				}
				return jsonToolResult(projects)
			},
		},
		{
			Tool: mcp.NewTool("get_project_problems",
				mcp.WithDescription("Get all problems linked to a project (via junction table)"),
				mcp.WithNumber("project_id", mcp.Required(), mcp.Description("Project ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectID, err := req.RequireFloat("project_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				problems, err := db.GetProjectProblems(int64(projectID))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to get project problems: %v", err)), nil
				}
				if problems == nil {
					problems = []*Problem{}
				}
				return jsonToolResult(problems)
			},
		},
	}
}

// --- Outcome Tools ---

func outcomeTools(db *Database, announceFunc func(string)) []server.ServerTool {
	return []server.ServerTool{
		{
			Tool: mcp.NewTool("create_outcome",
				mcp.WithDescription("Create a new outcome connected to a project and optionally a task"),
				mcp.WithNumber("project_id", mcp.Required(), mcp.Description("Project ID")),
				mcp.WithString("title", mcp.Required(), mcp.Description("Outcome title")),
				mcp.WithString("description", mcp.Description("Outcome description")),
				mcp.WithString("status", mcp.Description("Outcome status (e.g. open, completed)")),
				mcp.WithNumber("task_id", mcp.Description("Linked task ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectID, err := req.RequireFloat("project_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				title, err := req.RequireString("title")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				description := req.GetString("description", "")
				status := req.GetString("status", "")
				taskID := optionalInt64(req, "task_id")

				outcome, err := db.CreateOutcome(int64(projectID), taskID, title, description, status)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to create outcome: %v", err)), nil
				}
				announceFunc(fmt.Sprintf("Outcome %s created", title))
				return jsonToolResult(outcome)
			},
		},
		{
			Tool: mcp.NewTool("list_outcomes",
				mcp.WithDescription("List outcomes, optionally filtered by project, task, and status"),
				mcp.WithNumber("project_id", mcp.Description("Filter by project ID")),
				mcp.WithNumber("task_id", mcp.Description("Filter by task ID")),
				mcp.WithString("status", mcp.Description("Filter by status")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectID := optionalInt64(req, "project_id")
				taskID := optionalInt64(req, "task_id")
				status := optionalString(req, "status")

				outcomes, err := db.ListOutcomes(projectID, taskID, status)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list outcomes: %v", err)), nil
				}
				if outcomes == nil {
					outcomes = []*Outcome{}
				}
				return jsonToolResult(outcomes)
			},
		},
		{
			Tool: mcp.NewTool("get_outcome",
				mcp.WithDescription("Get details of a specific outcome"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Outcome ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				outcome, err := db.GetOutcome(int64(id))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to get outcome: %v", err)), nil
				}
				return jsonToolResult(outcome)
			},
		},
		{
			Tool: mcp.NewTool("update_outcome",
				mcp.WithDescription("Update an existing outcome"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Outcome ID")),
				mcp.WithString("title", mcp.Description("New outcome title")),
				mcp.WithString("description", mcp.Description("New outcome description")),
				mcp.WithString("status", mcp.Description("New outcome status")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				title := optionalString(req, "title")
				description := optionalString(req, "description")
				status := optionalString(req, "status")

				outcome, err := db.UpdateOutcome(int64(id), title, description, status)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to update outcome: %v", err)), nil
				}
				return jsonToolResult(outcome)
			},
		},
		{
			Tool: mcp.NewTool("delete_outcome",
				mcp.WithDescription("Delete an outcome"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Outcome ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := db.DeleteOutcome(int64(id)); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to delete outcome: %v", err)), nil
				}
				return mcp.NewToolResultText("outcome deleted successfully"), nil
			},
		},
	}
}

// --- Goal Tools ---

func goalTools(db *Database, announceFunc func(string)) []server.ServerTool {
	return []server.ServerTool{
		{
			Tool: mcp.NewTool("create_goal",
				mcp.WithDescription("Create a goal with optional project or task links and assignee"),
				mcp.WithString("title", mcp.Required(), mcp.Description("Goal title")),
				mcp.WithString("description", mcp.Description("Goal description")),
				mcp.WithString("goal_type", mcp.Description("Goal type (e.g. short_term, career, values, requirement)")),
				mcp.WithString("assignee", mcp.Description("Assignee name")),
				mcp.WithNumber("project_id", mcp.Description("Linked project ID")),
				mcp.WithNumber("task_id", mcp.Description("Linked task ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				title, err := req.RequireString("title")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				description := req.GetString("description", "")
				goalType := req.GetString("goal_type", "")
				assignee := req.GetString("assignee", "")
				projectID := optionalInt64(req, "project_id")
				taskID := optionalInt64(req, "task_id")

				goal, err := db.CreateGoal(projectID, taskID, title, description, goalType, assignee)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to create goal: %v", err)), nil
				}
				announceFunc(fmt.Sprintf("Goal %s created", title))
				return jsonToolResult(goal)
			},
		},
		{
			Tool: mcp.NewTool("list_goals",
				mcp.WithDescription("List goals, optionally filtered by project, task, goal type, and assignee"),
				mcp.WithNumber("project_id", mcp.Description("Filter by project ID")),
				mcp.WithNumber("task_id", mcp.Description("Filter by task ID")),
				mcp.WithString("goal_type", mcp.Description("Filter by goal type")),
				mcp.WithString("assignee", mcp.Description("Filter by assignee")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectID := optionalInt64(req, "project_id")
				taskID := optionalInt64(req, "task_id")
				goalType := optionalString(req, "goal_type")
				assignee := optionalString(req, "assignee")

				goals, err := db.ListGoals(projectID, taskID, goalType, assignee)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list goals: %v", err)), nil
				}
				if goals == nil {
					goals = []*Goal{}
				}
				return jsonToolResult(goals)
			},
		},
		{
			Tool: mcp.NewTool("get_goal",
				mcp.WithDescription("Get details of a specific goal"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Goal ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				goal, err := db.GetGoal(int64(id))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to get goal: %v", err)), nil
				}
				return jsonToolResult(goal)
			},
		},
		{
			Tool: mcp.NewTool("update_goal",
				mcp.WithDescription("Update an existing goal including assignee"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Goal ID")),
				mcp.WithString("title", mcp.Description("New goal title")),
				mcp.WithString("description", mcp.Description("New goal description")),
				mcp.WithString("goal_type", mcp.Description("New goal type")),
				mcp.WithString("assignee", mcp.Description("New assignee")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				title := optionalString(req, "title")
				description := optionalString(req, "description")
				goalType := optionalString(req, "goal_type")
				assignee := optionalString(req, "assignee")

				goal, err := db.UpdateGoal(int64(id), title, description, goalType, assignee)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to update goal: %v", err)), nil
				}
				return jsonToolResult(goal)
			},
		},
		{
			Tool: mcp.NewTool("delete_goal",
				mcp.WithDescription("Delete a goal"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Goal ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := db.DeleteGoal(int64(id)); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to delete goal: %v", err)), nil
				}
				return mcp.NewToolResultText("goal deleted successfully"), nil
			},
		},
		{
			Tool: mcp.NewTool("link_goal_to_project",
				mcp.WithDescription("Link a goal to an additional project (many-to-many relationship)"),
				mcp.WithNumber("goal_id", mcp.Required(), mcp.Description("Goal ID")),
				mcp.WithNumber("project_id", mcp.Required(), mcp.Description("Project ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				goalID, err := req.RequireFloat("goal_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				projectID, err := req.RequireFloat("project_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := db.LinkGoalToProject(int64(goalID), int64(projectID)); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to link goal to project: %v", err)), nil
				}
				return mcp.NewToolResultText("goal linked to project successfully"), nil
			},
		},
		{
			Tool: mcp.NewTool("unlink_goal_from_project",
				mcp.WithDescription("Remove a goal's link to a project"),
				mcp.WithNumber("goal_id", mcp.Required(), mcp.Description("Goal ID")),
				mcp.WithNumber("project_id", mcp.Required(), mcp.Description("Project ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				goalID, err := req.RequireFloat("goal_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				projectID, err := req.RequireFloat("project_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := db.UnlinkGoalFromProject(int64(goalID), int64(projectID)); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to unlink goal from project: %v", err)), nil
				}
				return mcp.NewToolResultText("goal unlinked from project successfully"), nil
			},
		},
		{
			Tool: mcp.NewTool("get_goal_projects",
				mcp.WithDescription("Get all projects linked to a goal"),
				mcp.WithNumber("goal_id", mcp.Required(), mcp.Description("Goal ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				goalID, err := req.RequireFloat("goal_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				projects, err := db.GetGoalProjects(int64(goalID))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to get goal projects: %v", err)), nil
				}
				if projects == nil {
					projects = []*Project{}
				}
				return jsonToolResult(projects)
			},
		},
		{
			Tool: mcp.NewTool("get_project_goals",
				mcp.WithDescription("Get all goals linked to a project (via junction table)"),
				mcp.WithNumber("project_id", mcp.Required(), mcp.Description("Project ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectID, err := req.RequireFloat("project_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				goals, err := db.GetProjectGoals(int64(projectID))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to get project goals: %v", err)), nil
				}
				if goals == nil {
					goals = []*Goal{}
				}
				return jsonToolResult(goals)
			},
		},
	}
}

// --- Task Note Tools ---

func taskNoteTools(db *Database, announceFunc func(string)) []server.ServerTool {
	return []server.ServerTool{
		{
			Tool: mcp.NewTool("create_task_note",
				mcp.WithDescription("Create a note on a task"),
				mcp.WithNumber("task_id", mcp.Required(), mcp.Description("Task ID")),
				mcp.WithString("note", mcp.Required(), mcp.Description("Note content")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				taskID, err := req.RequireFloat("task_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				note, err := req.RequireString("note")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				taskNote, err := db.CreateTaskNote(int64(taskID), note)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to create task note: %v", err)), nil
				}
				announceFunc("Task note created")
				return jsonToolResult(taskNote)
			},
		},
		{
			Tool: mcp.NewTool("list_task_notes",
				mcp.WithDescription("List notes for a task"),
				mcp.WithNumber("task_id", mcp.Required(), mcp.Description("Task ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				taskID, err := req.RequireFloat("task_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				notes, err := db.ListTaskNotes(int64(taskID))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list task notes: %v", err)), nil
				}
				if notes == nil {
					notes = []*TaskNote{}
				}
				return jsonToolResult(notes)
			},
		},
		{
			Tool: mcp.NewTool("get_task_note",
				mcp.WithDescription("Get details of a specific task note"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Task note ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				note, err := db.GetTaskNote(int64(id))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to get task note: %v", err)), nil
				}
				return jsonToolResult(note)
			},
		},
		{
			Tool: mcp.NewTool("update_task_note",
				mcp.WithDescription("Update an existing task note"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Task note ID")),
				mcp.WithString("note", mcp.Required(), mcp.Description("Updated note content")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				note, err := req.RequireString("note")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				taskNote, err := db.UpdateTaskNote(int64(id), note)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to update task note: %v", err)), nil
				}
				return jsonToolResult(taskNote)
			},
		},
		{
			Tool: mcp.NewTool("delete_task_note",
				mcp.WithDescription("Delete a task note"),
				mcp.WithNumber("id", mcp.Required(), mcp.Description("Task note ID")),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				id, err := req.RequireFloat("id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := db.DeleteTaskNote(int64(id)); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to delete task note: %v", err)), nil
				}
				return mcp.NewToolResultText("task note deleted successfully"), nil
			},
		},
	}
}

// --- Summary Tools ---

// ActiveWorkSummary represents a consolidated view of all active/in-progress work items.
type ActiveWorkSummary struct {
	Projects []*Project `json:"projects"`
	Tasks    []*Task    `json:"tasks"`
	Problems []*Problem `json:"problems"`
	Outcomes []*Outcome `json:"outcomes"`
}

func summaryTools(db *Database) []server.ServerTool {
	return []server.ServerTool{
		{
			Tool: mcp.NewTool("get_active_work_summary",
				mcp.WithDescription("Get a consolidated summary of all active work: active projects, pending/in-progress tasks, open/in-progress problems, and open/in-progress outcomes. This is more token-efficient than calling multiple list tools separately."),
			),
			Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				activeStatus := "active"
				projects, err := db.ListProjects(&activeStatus)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list active projects: %v", err)), nil
				}
				if projects == nil {
					projects = []*Project{}
				}

				// Get pending and in_progress tasks
				pendingStatus := "pending"
				pendingTasks, err := db.ListTasks(nil, &pendingStatus, nil)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list pending tasks: %v", err)), nil
				}
				inProgressStatus := "in_progress"
				inProgressTasks, err := db.ListTasks(nil, &inProgressStatus, nil)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list in-progress tasks: %v", err)), nil
				}
				tasks := append(pendingTasks, inProgressTasks...)
				if len(tasks) == 0 {
					tasks = []*Task{}
				}

				// Get open and in_progress problems
				openStatus := "open"
				openProblems, err := db.ListProblems(nil, nil, &openStatus, nil)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list open problems: %v", err)), nil
				}
				inProgressProblems, err := db.ListProblems(nil, nil, &inProgressStatus, nil)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list in-progress problems: %v", err)), nil
				}
				problems := append(openProblems, inProgressProblems...)
				if len(problems) == 0 {
					problems = []*Problem{}
				}

				// Get open and in_progress outcomes
				openOutcomes, err := db.ListOutcomes(nil, nil, &openStatus)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list open outcomes: %v", err)), nil
				}
				inProgressOutcomes, err := db.ListOutcomes(nil, nil, &inProgressStatus)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to list in-progress outcomes: %v", err)), nil
				}
				outcomes := append(openOutcomes, inProgressOutcomes...)
				if len(outcomes) == 0 {
					outcomes = []*Outcome{}
				}

				summary := ActiveWorkSummary{
					Projects: projects,
					Tasks:    tasks,
					Problems: problems,
					Outcomes: outcomes,
				}
				return jsonToolResult(summary)
			},
		},
	}
}

// --- Helpers ---

// jsonToolResult marshals data to JSON and returns it as a tool result.
func jsonToolResult(data interface{}) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// optionalString returns a pointer to the string value of the given argument,
// or nil if the argument is not present.
func optionalString(req mcp.CallToolRequest, key string) *string {
	args := req.GetArguments()
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return &s
		}
	}
	return nil
}

// optionalInt64 returns a pointer to the int64 value of the given numeric argument,
// or nil if the argument is not present.
func optionalInt64(req mcp.CallToolRequest, key string) *int64 {
	args := req.GetArguments()
	if v, ok := args[key]; ok {
		if f, ok := v.(float64); ok {
			i := int64(f)
			return &i
		}
	}
	return nil
}
