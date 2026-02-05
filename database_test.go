package main

import (
	"path/filepath"
	"testing"
)

func newTestDatabase(t *testing.T) *Database {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "loom.db")
	database, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	})

	return database
}

// --- Project CRUD ---

func TestCreateProject(t *testing.T) {
	db := newTestDatabase(t)

	project, err := db.CreateProject("Test Project", "A description", "", "https://example.com")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	if project.ID == 0 {
		t.Fatal("expected non-zero project ID")
	}
	if project.Name != "Test Project" {
		t.Fatalf("expected name %q, got %q", "Test Project", project.Name)
	}
	if project.Description != "A description" {
		t.Fatalf("expected description %q, got %q", "A description", project.Description)
	}
	if project.ExternalLink != "https://example.com" {
		t.Fatalf("expected external link %q, got %q", "https://example.com", project.ExternalLink)
	}
}

func TestGetProject(t *testing.T) {
	db := newTestDatabase(t)

	project, err := db.CreateProject("My Project", "desc", "", "")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	loaded, err := db.GetProject(project.ID)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}
	if loaded.Name != "My Project" {
		t.Fatalf("expected name %q, got %q", "My Project", loaded.Name)
	}
}

func TestGetProjectNotFound(t *testing.T) {
	db := newTestDatabase(t)

	_, err := db.GetProject(9999)
	if err == nil {
		t.Fatal("expected error for non-existent project")
	}
}

func TestListProjects(t *testing.T) {
	db := newTestDatabase(t)

	projects, err := db.ListProjects()
	if err != nil {
		t.Fatalf("failed to list projects: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected 0 projects, got %d", len(projects))
	}

	db.CreateProject("P1", "", "", "")
	db.CreateProject("P2", "", "", "")

	projects, err = db.ListProjects()
	if err != nil {
		t.Fatalf("failed to list projects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
}

func TestUpdateProject(t *testing.T) {
	db := newTestDatabase(t)

	project, err := db.CreateProject("Original", "original desc", "", "")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	newName := "Updated"
	newDesc := "updated desc"
	updated, err := db.UpdateProject(project.ID, &newName, &newDesc, nil, nil)
	if err != nil {
		t.Fatalf("failed to update project: %v", err)
	}
	if updated.Name != "Updated" {
		t.Fatalf("expected name %q, got %q", "Updated", updated.Name)
	}
	if updated.Description != "updated desc" {
		t.Fatalf("expected description %q, got %q", "updated desc", updated.Description)
	}
}

func TestUpdateProjectNoFields(t *testing.T) {
	db := newTestDatabase(t)

	project, err := db.CreateProject("NoChange", "", "", "")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	result, err := db.UpdateProject(project.ID, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to update project with no fields: %v", err)
	}
	if result.Name != "NoChange" {
		t.Fatalf("expected unchanged name %q, got %q", "NoChange", result.Name)
	}
}

func TestDeleteProject(t *testing.T) {
	db := newTestDatabase(t)

	project, err := db.CreateProject("ToDelete", "", "", "")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	if err := db.DeleteProject(project.ID); err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	_, err = db.GetProject(project.ID)
	if err == nil {
		t.Fatal("expected error after deleting project")
	}
}

func TestDeleteProjectNotFound(t *testing.T) {
	db := newTestDatabase(t)

	err := db.DeleteProject(9999)
	if err == nil {
		t.Fatal("expected error deleting non-existent project")
	}
}

// --- Task CRUD ---

func TestCreateTask(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")

	task, err := db.CreateTask(project.ID, "Task 1", "desc", "pending", "medium", "general", "https://jira.example.com/1")
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}
	if task.Title != "Task 1" {
		t.Fatalf("expected title %q, got %q", "Task 1", task.Title)
	}
	if task.Status != "pending" {
		t.Fatalf("expected status %q, got %q", "pending", task.Status)
	}
	if task.Priority != "medium" {
		t.Fatalf("expected priority %q, got %q", "medium", task.Priority)
	}
	if task.TaskType != "general" {
		t.Fatalf("expected task type %q, got %q", "general", task.TaskType)
	}
	if task.ExternalLink != "https://jira.example.com/1" {
		t.Fatalf("expected external link %q, got %q", "https://jira.example.com/1", task.ExternalLink)
	}
}

func TestCreateTaskDefaultType(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")

	task, err := db.CreateTask(project.ID, "Task", "", "pending", "low", "", "")
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}
	if task.TaskType != "general" {
		t.Fatalf("expected default task type %q, got %q", "general", task.TaskType)
	}
}

func TestGetTask(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "Task 1", "desc", "pending", "medium", "feature", "")

	loaded, err := db.GetTask(task.ID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if loaded.Title != "Task 1" {
		t.Fatalf("expected title %q, got %q", "Task 1", loaded.Title)
	}
	if loaded.ProjectID != project.ID {
		t.Fatalf("expected project ID %d, got %d", project.ID, loaded.ProjectID)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	db := newTestDatabase(t)

	_, err := db.GetTask(9999)
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestListTasks(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	db.CreateTask(project.ID, "T1", "", "pending", "low", "general", "")
	db.CreateTask(project.ID, "T2", "", "completed", "high", "bugfix", "")

	// List all
	tasks, err := db.ListTasks(nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	// Filter by status
	status := "pending"
	tasks, err = db.ListTasks(nil, &status, nil)
	if err != nil {
		t.Fatalf("failed to list tasks by status: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 pending task, got %d", len(tasks))
	}

	// Filter by task type
	taskType := "bugfix"
	tasks, err = db.ListTasks(nil, nil, &taskType)
	if err != nil {
		t.Fatalf("failed to list tasks by type: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 bugfix task, got %d", len(tasks))
	}

	// Filter by project ID
	tasks, err = db.ListTasks(&project.ID, nil, nil)
	if err != nil {
		t.Fatalf("failed to list tasks by project: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks for project, got %d", len(tasks))
	}
}

func TestUpdateTask(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "Original", "", "pending", "low", "general", "")

	newTitle := "Updated"
	newStatus := "in_progress"
	newPriority := "high"
	newType := "feature"
	updated, err := db.UpdateTask(task.ID, &newTitle, nil, &newStatus, &newPriority, &newType, nil)
	if err != nil {
		t.Fatalf("failed to update task: %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("expected title %q, got %q", "Updated", updated.Title)
	}
	if updated.Status != "in_progress" {
		t.Fatalf("expected status %q, got %q", "in_progress", updated.Status)
	}
	if updated.Priority != "high" {
		t.Fatalf("expected priority %q, got %q", "high", updated.Priority)
	}
	if updated.TaskType != "feature" {
		t.Fatalf("expected task type %q, got %q", "feature", updated.TaskType)
	}
}

func TestDeleteTask(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")

	if err := db.DeleteTask(task.ID); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}
	_, err := db.GetTask(task.ID)
	if err == nil {
		t.Fatal("expected error after deleting task")
	}
}

func TestDeleteTaskNotFound(t *testing.T) {
	db := newTestDatabase(t)

	err := db.DeleteTask(9999)
	if err == nil {
		t.Fatal("expected error deleting non-existent task")
	}
}

// --- Problem CRUD ---

func TestCreateProblemWithoutProject(t *testing.T) {
	database := newTestDatabase(t)

	problem, err := database.CreateProblem(nil, nil, "Unlinked problem", "Needs attention", "open")
	if err != nil {
		t.Fatalf("failed to create problem: %v", err)
	}
	if problem.ProjectID != nil {
		t.Fatalf("expected nil project ID, got %d", *problem.ProjectID)
	}

	loaded, err := database.GetProblem(problem.ID)
	if err != nil {
		t.Fatalf("failed to load problem: %v", err)
	}
	if loaded.ProjectID != nil {
		t.Fatalf("expected nil project ID after load, got %d", *loaded.ProjectID)
	}

	problems, err := database.ListProblems(nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to list problems: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}
}

func TestCreateProblemWithProject(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	problem, err := db.CreateProblem(&project.ID, nil, "Linked problem", "desc", "open")
	if err != nil {
		t.Fatalf("failed to create problem: %v", err)
	}
	if problem.ProjectID == nil || *problem.ProjectID != project.ID {
		t.Fatalf("expected project ID %d, got %v", project.ID, problem.ProjectID)
	}
}

func TestCreateProblemWithTask(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")

	problem, err := db.CreateProblem(&project.ID, &task.ID, "Task problem", "", "open")
	if err != nil {
		t.Fatalf("failed to create problem with task: %v", err)
	}
	if problem.TaskID == nil || *problem.TaskID != task.ID {
		t.Fatalf("expected task ID %d, got %v", task.ID, problem.TaskID)
	}
}

func TestGetProblemNotFound(t *testing.T) {
	db := newTestDatabase(t)

	_, err := db.GetProblem(9999)
	if err == nil {
		t.Fatal("expected error for non-existent problem")
	}
}

func TestListProblemsFiltered(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")

	db.CreateProblem(&project.ID, &task.ID, "P1", "", "open")
	db.CreateProblem(&project.ID, nil, "P2", "", "in_progress")
	db.CreateProblem(nil, nil, "P3", "", "open")

	// Filter by project
	problems, _ := db.ListProblems(&project.ID, nil, nil)
	if len(problems) != 2 {
		t.Fatalf("expected 2 problems for project, got %d", len(problems))
	}

	// Filter by task
	problems, _ = db.ListProblems(nil, &task.ID, nil)
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem for task, got %d", len(problems))
	}

	// Filter by status
	status := "open"
	problems, _ = db.ListProblems(nil, nil, &status)
	if len(problems) != 2 {
		t.Fatalf("expected 2 open problems, got %d", len(problems))
	}
}

func TestUpdateProblem(t *testing.T) {
	db := newTestDatabase(t)

	problem, _ := db.CreateProblem(nil, nil, "Original", "desc", "open")

	newTitle := "Updated"
	newStatus := "resolved"
	updated, err := db.UpdateProblem(problem.ID, &newTitle, nil, &newStatus)
	if err != nil {
		t.Fatalf("failed to update problem: %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("expected title %q, got %q", "Updated", updated.Title)
	}
	if updated.Status != "resolved" {
		t.Fatalf("expected status %q, got %q", "resolved", updated.Status)
	}
}

func TestDeleteProblem(t *testing.T) {
	db := newTestDatabase(t)

	problem, _ := db.CreateProblem(nil, nil, "ToDelete", "", "open")

	if err := db.DeleteProblem(problem.ID); err != nil {
		t.Fatalf("failed to delete problem: %v", err)
	}
	_, err := db.GetProblem(problem.ID)
	if err == nil {
		t.Fatal("expected error after deleting problem")
	}
}

func TestDeleteProblemNotFound(t *testing.T) {
	db := newTestDatabase(t)

	err := db.DeleteProblem(9999)
	if err == nil {
		t.Fatal("expected error deleting non-existent problem")
	}
}

// --- Outcome CRUD ---

func TestCreateOutcome(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")

	outcome, err := db.CreateOutcome(project.ID, nil, "Outcome 1", "desc", "open")
	if err != nil {
		t.Fatalf("failed to create outcome: %v", err)
	}
	if outcome.Title != "Outcome 1" {
		t.Fatalf("expected title %q, got %q", "Outcome 1", outcome.Title)
	}
	if outcome.ProjectID != project.ID {
		t.Fatalf("expected project ID %d, got %d", project.ID, outcome.ProjectID)
	}
	if outcome.TaskID != nil {
		t.Fatalf("expected nil task ID, got %d", *outcome.TaskID)
	}
}

func TestCreateOutcomeWithTask(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")

	outcome, err := db.CreateOutcome(project.ID, &task.ID, "Outcome", "", "open")
	if err != nil {
		t.Fatalf("failed to create outcome with task: %v", err)
	}
	if outcome.TaskID == nil || *outcome.TaskID != task.ID {
		t.Fatalf("expected task ID %d, got %v", task.ID, outcome.TaskID)
	}
}

func TestGetOutcomeNotFound(t *testing.T) {
	db := newTestDatabase(t)

	_, err := db.GetOutcome(9999)
	if err == nil {
		t.Fatal("expected error for non-existent outcome")
	}
}

func TestListOutcomes(t *testing.T) {
	db := newTestDatabase(t)

	p1, _ := db.CreateProject("P1", "", "", "")
	p2, _ := db.CreateProject("P2", "", "", "")
	task, _ := db.CreateTask(p1.ID, "T", "", "pending", "low", "general", "")

	db.CreateOutcome(p1.ID, &task.ID, "O1", "", "open")
	db.CreateOutcome(p1.ID, nil, "O2", "", "completed")
	db.CreateOutcome(p2.ID, nil, "O3", "", "open")

	// All
	outcomes, _ := db.ListOutcomes(nil, nil, nil)
	if len(outcomes) != 3 {
		t.Fatalf("expected 3 outcomes, got %d", len(outcomes))
	}

	// By project
	outcomes, _ = db.ListOutcomes(&p1.ID, nil, nil)
	if len(outcomes) != 2 {
		t.Fatalf("expected 2 outcomes for p1, got %d", len(outcomes))
	}

	// By task
	outcomes, _ = db.ListOutcomes(nil, &task.ID, nil)
	if len(outcomes) != 1 {
		t.Fatalf("expected 1 outcome for task, got %d", len(outcomes))
	}

	// By status
	status := "open"
	outcomes, _ = db.ListOutcomes(nil, nil, &status)
	if len(outcomes) != 2 {
		t.Fatalf("expected 2 open outcomes, got %d", len(outcomes))
	}
}

func TestUpdateOutcome(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	outcome, _ := db.CreateOutcome(project.ID, nil, "Original", "", "open")

	newTitle := "Updated"
	newStatus := "completed"
	updated, err := db.UpdateOutcome(outcome.ID, &newTitle, nil, &newStatus)
	if err != nil {
		t.Fatalf("failed to update outcome: %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("expected title %q, got %q", "Updated", updated.Title)
	}
	if updated.Status != "completed" {
		t.Fatalf("expected status %q, got %q", "completed", updated.Status)
	}
}

func TestDeleteOutcome(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	outcome, _ := db.CreateOutcome(project.ID, nil, "ToDelete", "", "open")

	if err := db.DeleteOutcome(outcome.ID); err != nil {
		t.Fatalf("failed to delete outcome: %v", err)
	}
	_, err := db.GetOutcome(outcome.ID)
	if err == nil {
		t.Fatal("expected error after deleting outcome")
	}
}

func TestDeleteOutcomeNotFound(t *testing.T) {
	db := newTestDatabase(t)

	err := db.DeleteOutcome(9999)
	if err == nil {
		t.Fatal("expected error deleting non-existent outcome")
	}
}

// --- Goal CRUD ---

func TestCreateGoalWithoutProject(t *testing.T) {
	database := newTestDatabase(t)

	goal, err := database.CreateGoal(nil, nil, "Career goal", "Move into leadership", "career")
	if err != nil {
		t.Fatalf("failed to create goal: %v", err)
	}
	if goal.ProjectID != nil {
		t.Fatalf("expected nil project ID, got %d", *goal.ProjectID)
	}
	if goal.GoalType != "career" {
		t.Fatalf("expected goal type career, got %s", goal.GoalType)
	}

	goals, err := database.ListGoals(nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to list goals: %v", err)
	}
	if len(goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(goals))
	}

	updatedTitle := "Updated career goal"
	updatedType := "values"
	updated, err := database.UpdateGoal(goal.ID, &updatedTitle, nil, &updatedType)
	if err != nil {
		t.Fatalf("failed to update goal: %v", err)
	}
	if updated.Title != updatedTitle {
		t.Fatalf("expected updated title %q, got %q", updatedTitle, updated.Title)
	}
	if updated.GoalType != updatedType {
		t.Fatalf("expected updated goal type %q, got %q", updatedType, updated.GoalType)
	}

	if err := database.DeleteGoal(goal.ID); err != nil {
		t.Fatalf("failed to delete goal: %v", err)
	}
}

func TestCreateGoalWithProject(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	goal, err := db.CreateGoal(&project.ID, nil, "Project goal", "", "short_term")
	if err != nil {
		t.Fatalf("failed to create goal with project: %v", err)
	}
	if goal.ProjectID == nil || *goal.ProjectID != project.ID {
		t.Fatalf("expected project ID %d, got %v", project.ID, goal.ProjectID)
	}
}

func TestCreateGoalWithTask(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")

	goal, err := db.CreateGoal(&project.ID, &task.ID, "Task goal", "", "requirement")
	if err != nil {
		t.Fatalf("failed to create goal with task: %v", err)
	}
	if goal.TaskID == nil || *goal.TaskID != task.ID {
		t.Fatalf("expected task ID %d, got %v", task.ID, goal.TaskID)
	}
}

func TestCreateGoalDefaultType(t *testing.T) {
	db := newTestDatabase(t)

	goal, err := db.CreateGoal(nil, nil, "Default type goal", "", "")
	if err != nil {
		t.Fatalf("failed to create goal: %v", err)
	}
	if goal.GoalType != "short_term" {
		t.Fatalf("expected default goal type %q, got %q", "short_term", goal.GoalType)
	}
}

func TestGetGoalNotFound(t *testing.T) {
	db := newTestDatabase(t)

	_, err := db.GetGoal(9999)
	if err == nil {
		t.Fatal("expected error for non-existent goal")
	}
}

func TestListGoalsFiltered(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")

	db.CreateGoal(&project.ID, &task.ID, "G1", "", "short_term")
	db.CreateGoal(&project.ID, nil, "G2", "", "career")
	db.CreateGoal(nil, nil, "G3", "", "short_term")

	// By project
	goals, _ := db.ListGoals(&project.ID, nil, nil)
	if len(goals) != 2 {
		t.Fatalf("expected 2 goals for project, got %d", len(goals))
	}

	// By task
	goals, _ = db.ListGoals(nil, &task.ID, nil)
	if len(goals) != 1 {
		t.Fatalf("expected 1 goal for task, got %d", len(goals))
	}

	// By type
	goalType := "short_term"
	goals, _ = db.ListGoals(nil, nil, &goalType)
	if len(goals) != 2 {
		t.Fatalf("expected 2 short_term goals, got %d", len(goals))
	}
}

func TestDeleteGoalNotFound(t *testing.T) {
	db := newTestDatabase(t)

	err := db.DeleteGoal(9999)
	if err == nil {
		t.Fatal("expected error deleting non-existent goal")
	}
}

// --- TaskNote CRUD ---

func TestCreateTaskNote(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")

	note, err := db.CreateTaskNote(task.ID, "This is a note")
	if err != nil {
		t.Fatalf("failed to create task note: %v", err)
	}
	if note.TaskID != task.ID {
		t.Fatalf("expected task ID %d, got %d", task.ID, note.TaskID)
	}
	if note.Note != "This is a note" {
		t.Fatalf("expected note %q, got %q", "This is a note", note.Note)
	}
}

func TestGetTaskNote(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")
	note, _ := db.CreateTaskNote(task.ID, "A note")

	loaded, err := db.GetTaskNote(note.ID)
	if err != nil {
		t.Fatalf("failed to get task note: %v", err)
	}
	if loaded.Note != "A note" {
		t.Fatalf("expected note %q, got %q", "A note", loaded.Note)
	}
}

func TestGetTaskNoteNotFound(t *testing.T) {
	db := newTestDatabase(t)

	_, err := db.GetTaskNote(9999)
	if err == nil {
		t.Fatal("expected error for non-existent task note")
	}
}

func TestListTaskNotes(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")

	db.CreateTaskNote(task.ID, "Note 1")
	db.CreateTaskNote(task.ID, "Note 2")

	notes, err := db.ListTaskNotes(task.ID)
	if err != nil {
		t.Fatalf("failed to list task notes: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
}

func TestListTaskNotesEmpty(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")

	notes, err := db.ListTaskNotes(task.ID)
	if err != nil {
		t.Fatalf("failed to list task notes: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d", len(notes))
	}
}

func TestUpdateTaskNote(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")
	note, _ := db.CreateTaskNote(task.ID, "Original note")

	updated, err := db.UpdateTaskNote(note.ID, "Updated note")
	if err != nil {
		t.Fatalf("failed to update task note: %v", err)
	}
	if updated.Note != "Updated note" {
		t.Fatalf("expected note %q, got %q", "Updated note", updated.Note)
	}
}

func TestDeleteTaskNote(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")
	note, _ := db.CreateTaskNote(task.ID, "To delete")

	if err := db.DeleteTaskNote(note.ID); err != nil {
		t.Fatalf("failed to delete task note: %v", err)
	}
	_, err := db.GetTaskNote(note.ID)
	if err == nil {
		t.Fatal("expected error after deleting task note")
	}
}

func TestDeleteTaskNoteNotFound(t *testing.T) {
	db := newTestDatabase(t)

	err := db.DeleteTaskNote(9999)
	if err == nil {
		t.Fatal("expected error deleting non-existent task note")
	}
}

// --- Foreign Key Cascade Tests ---

func TestDeleteProjectCascadesToTasks(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task1, _ := db.CreateTask(project.ID, "T1", "", "pending", "low", "general", "")
	task2, _ := db.CreateTask(project.ID, "T2", "", "pending", "low", "general", "")

	if err := db.DeleteProject(project.ID); err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	// Tasks should be cascade-deleted
	_, err := db.GetTask(task1.ID)
	if err == nil {
		t.Fatal("expected task1 to be cascade-deleted with project")
	}
	_, err = db.GetTask(task2.ID)
	if err == nil {
		t.Fatal("expected task2 to be cascade-deleted with project")
	}
}

func TestDeleteProjectCascadesToOutcomes(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	outcome, _ := db.CreateOutcome(project.ID, nil, "O", "", "open")

	if err := db.DeleteProject(project.ID); err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	_, err := db.GetOutcome(outcome.ID)
	if err == nil {
		t.Fatal("expected outcome to be cascade-deleted with project")
	}
}

func TestDeleteProjectSetsNullOnProblems(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	problem, _ := db.CreateProblem(&project.ID, nil, "Problem", "", "open")

	if err := db.DeleteProject(project.ID); err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	// Problem should still exist but with null project_id
	loaded, err := db.GetProblem(problem.ID)
	if err != nil {
		t.Fatalf("problem should still exist after project deletion: %v", err)
	}
	if loaded.ProjectID != nil {
		t.Fatalf("expected nil project ID after cascade, got %d", *loaded.ProjectID)
	}
}

func TestDeleteProjectSetsNullOnGoals(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	goal, _ := db.CreateGoal(&project.ID, nil, "Goal", "", "short_term")

	if err := db.DeleteProject(project.ID); err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	loaded, err := db.GetGoal(goal.ID)
	if err != nil {
		t.Fatalf("goal should still exist after project deletion: %v", err)
	}
	if loaded.ProjectID != nil {
		t.Fatalf("expected nil project ID after cascade, got %d", *loaded.ProjectID)
	}
}

func TestDeleteTaskCascadesToNotes(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")
	note, _ := db.CreateTaskNote(task.ID, "A note")

	if err := db.DeleteTask(task.ID); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	_, err := db.GetTaskNote(note.ID)
	if err == nil {
		t.Fatal("expected task note to be cascade-deleted with task")
	}
}

func TestDeleteTaskSetsNullOnProblems(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")
	problem, _ := db.CreateProblem(&project.ID, &task.ID, "Problem", "", "open")

	if err := db.DeleteTask(task.ID); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	loaded, err := db.GetProblem(problem.ID)
	if err != nil {
		t.Fatalf("problem should still exist after task deletion: %v", err)
	}
	if loaded.TaskID != nil {
		t.Fatalf("expected nil task ID after cascade, got %d", *loaded.TaskID)
	}
	// project_id should remain
	if loaded.ProjectID == nil || *loaded.ProjectID != project.ID {
		t.Fatalf("expected project ID to remain %d, got %v", project.ID, loaded.ProjectID)
	}
}

func TestDeleteTaskSetsNullOnOutcomes(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")
	outcome, _ := db.CreateOutcome(project.ID, &task.ID, "Outcome", "", "open")

	if err := db.DeleteTask(task.ID); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	loaded, err := db.GetOutcome(outcome.ID)
	if err != nil {
		t.Fatalf("outcome should still exist after task deletion: %v", err)
	}
	if loaded.TaskID != nil {
		t.Fatalf("expected nil task ID after cascade, got %d", *loaded.TaskID)
	}
}

func TestDeleteTaskSetsNullOnGoals(t *testing.T) {
	db := newTestDatabase(t)

	project, _ := db.CreateProject("P", "", "", "")
	task, _ := db.CreateTask(project.ID, "T", "", "pending", "low", "general", "")
	goal, _ := db.CreateGoal(&project.ID, &task.ID, "Goal", "", "short_term")

	if err := db.DeleteTask(task.ID); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	loaded, err := db.GetGoal(goal.ID)
	if err != nil {
		t.Fatalf("goal should still exist after task deletion: %v", err)
	}
	if loaded.TaskID != nil {
		t.Fatalf("expected nil task ID after cascade, got %d", *loaded.TaskID)
	}
}

// --- Foreign Key Enforcement Tests ---

func TestForeignKeyEnforcement(t *testing.T) {
	db := newTestDatabase(t)

	// Creating a task with a non-existent project_id should fail
	_, err := db.CreateTask(9999, "Bad Task", "", "pending", "low", "general", "")
	if err == nil {
		t.Fatal("expected foreign key error when creating task with non-existent project_id")
	}
}

func TestForeignKeyEnforcementTaskNotes(t *testing.T) {
	db := newTestDatabase(t)

	// Creating a task note with a non-existent task_id should fail
	_, err := db.CreateTaskNote(9999, "Bad note")
	if err == nil {
		t.Fatal("expected foreign key error when creating task note with non-existent task_id")
	}
}

func TestForeignKeyEnforcementOutcomes(t *testing.T) {
	db := newTestDatabase(t)

	// Creating an outcome with a non-existent project_id should fail
	_, err := db.CreateOutcome(9999, nil, "Bad outcome", "", "open")
	if err == nil {
		t.Fatal("expected foreign key error when creating outcome with non-existent project_id")
	}
}

// --- Database Initialization ---

func TestNewDatabaseCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "subdir", "nested", "loom.db")

	database, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("failed to create database in nested directory: %v", err)
	}
	database.Close()
}

func TestNewDatabaseIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "loom.db")

	// First open creates schema
	db1, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("first open failed: %v", err)
	}

	// Create some data
	_, err = db1.CreateProject("Test", "", "", "")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	db1.Close()

	// Second open should work and preserve data
	db2, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("second open failed: %v", err)
	}
	defer db2.Close()

	projects, err := db2.ListProjects()
	if err != nil {
		t.Fatalf("failed to list projects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project after reopen, got %d", len(projects))
	}
}
