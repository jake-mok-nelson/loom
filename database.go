package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
	ExternalLink string    `json:"external_link"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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

	database := &Database{db: db}
	if err := database.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return database, nil
}

// initSchema initializes the database schema
func (d *Database) initSchema() error {
	schema := `
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
		external_link TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	`

	_, err := d.db.Exec(schema)
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
	_, err := d.db.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

// Task operations

func (d *Database) CreateTask(projectID int64, title, description, status, priority, externalLink string) (*Task, error) {
	result, err := d.db.Exec(
		"INSERT INTO tasks (project_id, title, description, status, priority, external_link) VALUES (?, ?, ?, ?, ?, ?)",
		projectID, title, description, status, priority, externalLink,
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
		"SELECT id, project_id, title, description, status, priority, external_link, created_at, updated_at FROM tasks WHERE id = ?",
		id,
	).Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.ExternalLink, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (d *Database) ListTasks(projectID *int64, status *string) ([]*Task, error) {
	query := "SELECT id, project_id, title, description, status, priority, external_link, created_at, updated_at FROM tasks WHERE 1=1"
	args := []interface{}{}

	if projectID != nil {
		query += " AND project_id = ?"
		args = append(args, *projectID)
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

	var tasks []*Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.ExternalLink, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, &t)
	}
	return tasks, rows.Err()
}

func (d *Database) UpdateTask(id int64, title, description, status, priority, externalLink *string) (*Task, error) {
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

func (d *Database) DeleteTask(id int64) error {
	_, err := d.db.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}
