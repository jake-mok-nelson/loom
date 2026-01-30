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
