package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Database struct {
	db *sql.DB
}

type Project struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	ExternalLink string    `json:"external_link"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Task struct {
	ID           int64     `json:"id"`
	ProjectID    int64     `json:"project_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	Priority     string    `json:"priority"`
	TaskType     string    `json:"task_type"`
	ExternalLink string    `json:"external_link"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Problem struct {
	ID          int64     `json:"id"`
	ProjectID   *int64    `json:"project_id"`
	TaskID      *int64    `json:"task_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Outcome struct {
	ID          int64     `json:"id"`
	ProjectID   int64     `json:"project_id"`
	TaskID      *int64    `json:"task_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Goal struct {
	ID          int64     `json:"id"`
	ProjectID   *int64    `json:"project_id"`
	TaskID      *int64    `json:"task_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	GoalType    string    `json:"goal_type"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TaskNote struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewDatabase creates a new database connection
func NewDatabase(dbPath string) (*Database, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign key enforcement (required for ON DELETE CASCADE)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	database := &Database{db: db}
	if err := database.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return database, nil
}

// initSchema initializes the database schema
func (d *Database) initSchema() error {
	tables := `
	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		external_link TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT DEFAULT 'pending',
		priority TEXT DEFAULT 'medium',
		task_type TEXT DEFAULT 'general',
		external_link TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS problems (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER,
		task_id INTEGER,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT DEFAULT 'open',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL,
		FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS outcomes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		task_id INTEGER,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT DEFAULT 'open',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
		FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS goals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER,
		task_id INTEGER,
		title TEXT NOT NULL,
		description TEXT,
		goal_type TEXT DEFAULT 'short_term',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL,
		FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS task_notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id INTEGER NOT NULL,
		note TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
	);
	`

	if _, err := d.db.Exec(tables); err != nil {
		return err
	}

	if _, err := d.db.Exec("ALTER TABLE tasks ADD COLUMN task_type TEXT DEFAULT 'general'"); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	if _, err := d.db.Exec("UPDATE tasks SET task_type = 'general' WHERE task_type IS NULL OR task_type = ''"); err != nil {
		return err
	}

	if err := d.ensureProblemProjectOptional(); err != nil {
		return err
	}

	indexes := `
	CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_task_type ON tasks(task_type);
	CREATE INDEX IF NOT EXISTS idx_problems_project_id ON problems(project_id);
	CREATE INDEX IF NOT EXISTS idx_problems_task_id ON problems(task_id);
	CREATE INDEX IF NOT EXISTS idx_problems_status ON problems(status);
	CREATE INDEX IF NOT EXISTS idx_outcomes_project_id ON outcomes(project_id);
	CREATE INDEX IF NOT EXISTS idx_outcomes_task_id ON outcomes(task_id);
	CREATE INDEX IF NOT EXISTS idx_outcomes_status ON outcomes(status);
	CREATE INDEX IF NOT EXISTS idx_goals_project_id ON goals(project_id);
	CREATE INDEX IF NOT EXISTS idx_goals_task_id ON goals(task_id);
	CREATE INDEX IF NOT EXISTS idx_goals_goal_type ON goals(goal_type);
	CREATE INDEX IF NOT EXISTS idx_task_notes_task_id ON task_notes(task_id);
	`

	_, err := d.db.Exec(indexes)
	return err
}

func (d *Database) ensureProblemProjectOptional() error {
	var columnNotNull sql.NullString
	err := d.db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='problems'").Scan(&columnNotNull)
	if err != nil {
		return err
	}
	if !strings.Contains(columnNotNull.String, "project_id INTEGER NOT NULL") {
		return nil
	}

	migration := `
	ALTER TABLE problems RENAME TO problems_old;
	CREATE TABLE problems (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER,
		task_id INTEGER,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT DEFAULT 'open',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL,
		FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE SET NULL
	);
	INSERT INTO problems (id, project_id, task_id, title, description, status, created_at, updated_at)
	SELECT id, project_id, task_id, title, description, status, created_at, updated_at
	FROM problems_old;
	DROP TABLE problems_old;
	`
	_, err = d.db.Exec(migration)
	return err
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// Project operations

func (d *Database) CreateProject(name, description, externalLink string) (*Project, error) {
	result, err := d.db.Exec(
		"INSERT INTO projects (name, description, external_link) VALUES (?, ?, ?)",
		name, description, externalLink,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return d.GetProject(id)
}

func (d *Database) GetProject(id int64) (*Project, error) {
	var p Project
	err := d.db.QueryRow(
		"SELECT id, name, description, external_link, created_at, updated_at FROM projects WHERE id = ?",
		id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.ExternalLink, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (d *Database) ListProjects() ([]*Project, error) {
	rows, err := d.db.Query(
		"SELECT id, name, description, external_link, created_at, updated_at FROM projects ORDER BY updated_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.ExternalLink, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, &p)
	}
	return projects, rows.Err()
}

func (d *Database) UpdateProject(id int64, name, description, externalLink *string) (*Project, error) {
	updates := []string{}
	args := []interface{}{}

	if name != nil {
		updates = append(updates, "name = ?")
		args = append(args, *name)
	}
	if description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *description)
	}
	if externalLink != nil {
		updates = append(updates, "external_link = ?")
		args = append(args, *externalLink)
	}

	if len(updates) == 0 {
		return d.GetProject(id)
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := "UPDATE projects SET " + updates[0]
	for i := 1; i < len(updates); i++ {
		query += ", " + updates[i]
	}
	query += " WHERE id = ?"

	_, err := d.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	return d.GetProject(id)
}

func (d *Database) DeleteProject(id int64) error {
	result, err := d.db.Exec("DELETE FROM projects WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("project with ID %d not found", id)
	}
	return nil
}

// Task operations

func (d *Database) CreateTask(projectID int64, title, description, status, priority, taskType, externalLink string) (*Task, error) {
	if taskType == "" {
		taskType = "general"
	}
	result, err := d.db.Exec(
		"INSERT INTO tasks (project_id, title, description, status, priority, task_type, external_link) VALUES (?, ?, ?, ?, ?, ?, ?)",
		projectID, title, description, status, priority, taskType, externalLink,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return d.GetTask(id)
}

func (d *Database) GetTask(id int64) (*Task, error) {
	var t Task
	err := d.db.QueryRow(
		"SELECT id, project_id, title, description, status, priority, task_type, external_link, created_at, updated_at FROM tasks WHERE id = ?",
		id,
	).Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.TaskType, &t.ExternalLink, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (d *Database) ListTasks(projectID *int64, status *string, taskType *string) ([]*Task, error) {
	query := "SELECT id, project_id, title, description, status, priority, task_type, external_link, created_at, updated_at FROM tasks WHERE 1=1"
	args := []interface{}{}

	if projectID != nil {
		query += " AND project_id = ?"
		args = append(args, *projectID)
	}

	if status != nil {
		query += " AND status = ?"
		args = append(args, *status)
	}

	if taskType != nil {
		query += " AND task_type = ?"
		args = append(args, *taskType)
	}

	query += " ORDER BY updated_at DESC"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.TaskType, &t.ExternalLink, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, &t)
	}
	return tasks, rows.Err()
}

func (d *Database) UpdateTask(id int64, title, description, status, priority, taskType, externalLink *string) (*Task, error) {
	updates := []string{}
	args := []interface{}{}

	if title != nil {
		updates = append(updates, "title = ?")
		args = append(args, *title)
	}
	if description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *description)
	}
	if status != nil {
		updates = append(updates, "status = ?")
		args = append(args, *status)
	}
	if priority != nil {
		updates = append(updates, "priority = ?")
		args = append(args, *priority)
	}
	if taskType != nil {
		updates = append(updates, "task_type = ?")
		args = append(args, *taskType)
	}
	if externalLink != nil {
		updates = append(updates, "external_link = ?")
		args = append(args, *externalLink)
	}

	if len(updates) == 0 {
		return d.GetTask(id)
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := "UPDATE tasks SET " + updates[0]
	for i := 1; i < len(updates); i++ {
		query += ", " + updates[i]
	}
	query += " WHERE id = ?"

	_, err := d.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	return d.GetTask(id)
}

// Problem operations

func (d *Database) CreateProblem(projectID *int64, taskID *int64, title, description, status string) (*Problem, error) {
	result, err := d.db.Exec(
		"INSERT INTO problems (project_id, task_id, title, description, status) VALUES (?, ?, ?, ?, ?)",
		projectID, taskID, title, description, status,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return d.GetProblem(id)
}

func (d *Database) GetProblem(id int64) (*Problem, error) {
	var p Problem
	var projectID sql.NullInt64
	var taskID sql.NullInt64
	err := d.db.QueryRow(
		"SELECT id, project_id, task_id, title, description, status, created_at, updated_at FROM problems WHERE id = ?",
		id,
	).Scan(&p.ID, &projectID, &taskID, &p.Title, &p.Description, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if projectID.Valid {
		p.ProjectID = &projectID.Int64
	}
	if taskID.Valid {
		p.TaskID = &taskID.Int64
	}
	return &p, nil
}

func (d *Database) ListProblems(projectID *int64, taskID *int64, status *string) ([]*Problem, error) {
	query := "SELECT id, project_id, task_id, title, description, status, created_at, updated_at FROM problems WHERE 1=1"
	args := []interface{}{}

	if projectID != nil {
		query += " AND project_id = ?"
		args = append(args, *projectID)
	}

	if taskID != nil {
		query += " AND task_id = ?"
		args = append(args, *taskID)
	}

	if status != nil {
		query += " AND status = ?"
		args = append(args, *status)
	}

	query += " ORDER BY updated_at DESC"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var problems []*Problem
	for rows.Next() {
		var p Problem
		var projectID sql.NullInt64
		var taskID sql.NullInt64
		if err := rows.Scan(&p.ID, &projectID, &taskID, &p.Title, &p.Description, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if projectID.Valid {
			p.ProjectID = &projectID.Int64
		}
		if taskID.Valid {
			p.TaskID = &taskID.Int64
		}
		problems = append(problems, &p)
	}
	return problems, rows.Err()
}

func (d *Database) UpdateProblem(id int64, title, description, status *string) (*Problem, error) {
	updates := []string{}
	args := []interface{}{}

	if title != nil {
		updates = append(updates, "title = ?")
		args = append(args, *title)
	}
	if description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *description)
	}
	if status != nil {
		updates = append(updates, "status = ?")
		args = append(args, *status)
	}

	if len(updates) == 0 {
		return d.GetProblem(id)
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := "UPDATE problems SET " + updates[0]
	for i := 1; i < len(updates); i++ {
		query += ", " + updates[i]
	}
	query += " WHERE id = ?"

	_, err := d.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	return d.GetProblem(id)
}

func (d *Database) DeleteProblem(id int64) error {
	result, err := d.db.Exec("DELETE FROM problems WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("problem with ID %d not found", id)
	}
	return nil
}

// Outcome operations

func (d *Database) CreateOutcome(projectID int64, taskID *int64, title, description, status string) (*Outcome, error) {
	result, err := d.db.Exec(
		"INSERT INTO outcomes (project_id, task_id, title, description, status) VALUES (?, ?, ?, ?, ?)",
		projectID, taskID, title, description, status,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return d.GetOutcome(id)
}

func (d *Database) GetOutcome(id int64) (*Outcome, error) {
	var outcome Outcome
	var taskID sql.NullInt64
	err := d.db.QueryRow(
		"SELECT id, project_id, task_id, title, description, status, created_at, updated_at FROM outcomes WHERE id = ?",
		id,
	).Scan(&outcome.ID, &outcome.ProjectID, &taskID, &outcome.Title, &outcome.Description, &outcome.Status, &outcome.CreatedAt, &outcome.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if taskID.Valid {
		outcome.TaskID = &taskID.Int64
	}
	return &outcome, nil
}

func (d *Database) ListOutcomes(projectID *int64, taskID *int64, status *string) ([]*Outcome, error) {
	query := "SELECT id, project_id, task_id, title, description, status, created_at, updated_at FROM outcomes WHERE 1=1"
	args := []interface{}{}

	if projectID != nil {
		query += " AND project_id = ?"
		args = append(args, *projectID)
	}

	if taskID != nil {
		query += " AND task_id = ?"
		args = append(args, *taskID)
	}

	if status != nil {
		query += " AND status = ?"
		args = append(args, *status)
	}

	query += " ORDER BY updated_at DESC"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outcomes []*Outcome
	for rows.Next() {
		var outcome Outcome
		var taskID sql.NullInt64
		if err := rows.Scan(&outcome.ID, &outcome.ProjectID, &taskID, &outcome.Title, &outcome.Description, &outcome.Status, &outcome.CreatedAt, &outcome.UpdatedAt); err != nil {
			return nil, err
		}
		if taskID.Valid {
			outcome.TaskID = &taskID.Int64
		}
		outcomes = append(outcomes, &outcome)
	}
	return outcomes, rows.Err()
}

func (d *Database) UpdateOutcome(id int64, title, description, status *string) (*Outcome, error) {
	updates := []string{}
	args := []interface{}{}

	if title != nil {
		updates = append(updates, "title = ?")
		args = append(args, *title)
	}
	if description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *description)
	}
	if status != nil {
		updates = append(updates, "status = ?")
		args = append(args, *status)
	}

	if len(updates) == 0 {
		return d.GetOutcome(id)
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := "UPDATE outcomes SET " + updates[0]
	for i := 1; i < len(updates); i++ {
		query += ", " + updates[i]
	}
	query += " WHERE id = ?"

	_, err := d.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	return d.GetOutcome(id)
}

func (d *Database) DeleteOutcome(id int64) error {
	result, err := d.db.Exec("DELETE FROM outcomes WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("outcome with ID %d not found", id)
	}
	return nil
}

// Goal operations

func (d *Database) CreateGoal(projectID *int64, taskID *int64, title, description, goalType string) (*Goal, error) {
	if goalType == "" {
		goalType = "short_term"
	}
	result, err := d.db.Exec(
		"INSERT INTO goals (project_id, task_id, title, description, goal_type) VALUES (?, ?, ?, ?, ?)",
		projectID, taskID, title, description, goalType,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return d.GetGoal(id)
}

func (d *Database) GetGoal(id int64) (*Goal, error) {
	var g Goal
	var projectID sql.NullInt64
	var taskID sql.NullInt64
	err := d.db.QueryRow(
		"SELECT id, project_id, task_id, title, description, goal_type, created_at, updated_at FROM goals WHERE id = ?",
		id,
	).Scan(&g.ID, &projectID, &taskID, &g.Title, &g.Description, &g.GoalType, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if projectID.Valid {
		g.ProjectID = &projectID.Int64
	}
	if taskID.Valid {
		g.TaskID = &taskID.Int64
	}
	return &g, nil
}

func (d *Database) ListGoals(projectID *int64, taskID *int64, goalType *string) ([]*Goal, error) {
	query := "SELECT id, project_id, task_id, title, description, goal_type, created_at, updated_at FROM goals WHERE 1=1"
	args := []interface{}{}

	if projectID != nil {
		query += " AND project_id = ?"
		args = append(args, *projectID)
	}

	if taskID != nil {
		query += " AND task_id = ?"
		args = append(args, *taskID)
	}

	if goalType != nil {
		query += " AND goal_type = ?"
		args = append(args, *goalType)
	}

	query += " ORDER BY updated_at DESC"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []*Goal
	for rows.Next() {
		var g Goal
		var projectID sql.NullInt64
		var taskID sql.NullInt64
		if err := rows.Scan(&g.ID, &projectID, &taskID, &g.Title, &g.Description, &g.GoalType, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		if projectID.Valid {
			g.ProjectID = &projectID.Int64
		}
		if taskID.Valid {
			g.TaskID = &taskID.Int64
		}
		goals = append(goals, &g)
	}
	return goals, rows.Err()
}

func (d *Database) UpdateGoal(id int64, title, description, goalType *string) (*Goal, error) {
	updates := []string{}
	args := []interface{}{}

	if title != nil {
		updates = append(updates, "title = ?")
		args = append(args, *title)
	}
	if description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *description)
	}
	if goalType != nil {
		updates = append(updates, "goal_type = ?")
		args = append(args, *goalType)
	}

	if len(updates) == 0 {
		return d.GetGoal(id)
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := "UPDATE goals SET " + updates[0]
	for i := 1; i < len(updates); i++ {
		query += ", " + updates[i]
	}
	query += " WHERE id = ?"

	_, err := d.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	return d.GetGoal(id)
}

func (d *Database) DeleteGoal(id int64) error {
	result, err := d.db.Exec("DELETE FROM goals WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("goal with ID %d not found", id)
	}
	return nil
}

// Task note operations

func (d *Database) CreateTaskNote(taskID int64, note string) (*TaskNote, error) {
	result, err := d.db.Exec(
		"INSERT INTO task_notes (task_id, note) VALUES (?, ?)",
		taskID, note,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return d.GetTaskNote(id)
}

func (d *Database) GetTaskNote(id int64) (*TaskNote, error) {
	var note TaskNote
	err := d.db.QueryRow(
		"SELECT id, task_id, note, created_at, updated_at FROM task_notes WHERE id = ?",
		id,
	).Scan(&note.ID, &note.TaskID, &note.Note, &note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &note, nil
}

func (d *Database) ListTaskNotes(taskID int64) ([]*TaskNote, error) {
	rows, err := d.db.Query(
		"SELECT id, task_id, note, created_at, updated_at FROM task_notes WHERE task_id = ? ORDER BY updated_at DESC",
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*TaskNote
	for rows.Next() {
		var note TaskNote
		if err := rows.Scan(&note.ID, &note.TaskID, &note.Note, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, &note)
	}
	return notes, rows.Err()
}

func (d *Database) UpdateTaskNote(id int64, note string) (*TaskNote, error) {
	_, err := d.db.Exec("UPDATE task_notes SET note = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", note, id)
	if err != nil {
		return nil, err
	}
	return d.GetTaskNote(id)
}

func (d *Database) DeleteTaskNote(id int64) error {
	result, err := d.db.Exec("DELETE FROM task_notes WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("task note with ID %d not found", id)
	}
	return nil
}

func (d *Database) DeleteTask(id int64) error {
	result, err := d.db.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("task with ID %d not found", id)
	}
	return nil
}
