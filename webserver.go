package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// WebServer handles HTTP requests and SSE for the Loom dashboard
type WebServer struct {
	db         *Database
	addr       string
	clients    map[chan string]bool
	clientsMux sync.RWMutex
}

// NewWebServer creates a new web server instance
func NewWebServer(db *Database, addr string) *WebServer {
	return &WebServer{
		db:      db,
		addr:    addr,
		clients: make(map[chan string]bool),
	}
}

// Start begins the web server
func (ws *WebServer) Start() error {
	mux := http.NewServeMux()

	// Serve the main dashboard
	mux.HandleFunc("/", ws.handleDashboard)

	// API endpoints
	mux.HandleFunc("/api/projects", ws.handleProjects)
	mux.HandleFunc("/api/tasks", ws.handleTasks)
	mux.HandleFunc("/api/problems", ws.handleProblems)
	mux.HandleFunc("/api/outcomes", ws.handleOutcomes)
	mux.HandleFunc("/api/goals", ws.handleGoals)

	// SSE endpoint for real-time updates
	mux.HandleFunc("/events", ws.handleSSE)

	log.Printf("Starting Loom web server at http://%s", ws.addr)
	return http.ListenAndServe(ws.addr, mux)
}

// broadcast sends an event to all connected SSE clients
func (ws *WebServer) broadcast(eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling event data: %v", err)
		return
	}

	event := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, jsonData)

	ws.clientsMux.RLock()
	defer ws.clientsMux.RUnlock()

	for client := range ws.clients {
		select {
		case client <- event:
		default:
			// Client buffer full, skip
		}
	}
}

// handleSSE handles Server-Sent Events connections
func (ws *WebServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for this client
	clientChan := make(chan string, 10)

	ws.clientsMux.Lock()
	ws.clients[clientChan] = true
	ws.clientsMux.Unlock()

	defer func() {
		ws.clientsMux.Lock()
		delete(ws.clients, clientChan)
		close(clientChan)
		ws.clientsMux.Unlock()
	}()

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Keep connection alive with periodic heartbeats
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-clientChan:
			fmt.Fprint(w, event)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-ticker.C:
			fmt.Fprintf(w, "event: heartbeat\ndata: {\"time\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

// handleProjects handles the /api/projects endpoint
func (ws *WebServer) handleProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	projects, err := ws.db.ListProjects()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	if projects == nil {
		projects = []*Project{}
	}

	json.NewEncoder(w).Encode(projects)
}

// handleTasks handles the /api/tasks endpoint
func (ws *WebServer) handleTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse query parameters
	var projectID *int64
	var status *string
	var taskType *string

	if pidStr := r.URL.Query().Get("project_id"); pidStr != "" {
		if pid, err := strconv.ParseInt(pidStr, 10, 64); err == nil {
			projectID = &pid
		}
	}

	if s := r.URL.Query().Get("status"); s != "" {
		status = &s
	}

	if t := r.URL.Query().Get("task_type"); t != "" {
		taskType = &t
	}

	tasks, err := ws.db.ListTasks(projectID, status, taskType)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	if tasks == nil {
		tasks = []*Task{}
	}

	json.NewEncoder(w).Encode(tasks)
}

// handleProblems handles the /api/problems endpoint
func (ws *WebServer) handleProblems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var projectID *int64
	var taskID *int64
	var status *string

	if pidStr := r.URL.Query().Get("project_id"); pidStr != "" {
		if pid, err := strconv.ParseInt(pidStr, 10, 64); err == nil {
			projectID = &pid
		}
	}

	if tidStr := r.URL.Query().Get("task_id"); tidStr != "" {
		if tid, err := strconv.ParseInt(tidStr, 10, 64); err == nil {
			taskID = &tid
		}
	}

	if s := r.URL.Query().Get("status"); s != "" {
		status = &s
	}

	problems, err := ws.db.ListProblems(projectID, taskID, status)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	if problems == nil {
		problems = []*Problem{}
	}

	json.NewEncoder(w).Encode(problems)
}

// handleOutcomes handles the /api/outcomes endpoint
func (ws *WebServer) handleOutcomes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var projectID *int64
	var taskID *int64
	var status *string

	if pidStr := r.URL.Query().Get("project_id"); pidStr != "" {
		if pid, err := strconv.ParseInt(pidStr, 10, 64); err == nil {
			projectID = &pid
		}
	}

	if tidStr := r.URL.Query().Get("task_id"); tidStr != "" {
		if tid, err := strconv.ParseInt(tidStr, 10, 64); err == nil {
			taskID = &tid
		}
	}

	if s := r.URL.Query().Get("status"); s != "" {
		status = &s
	}

	outcomes, err := ws.db.ListOutcomes(projectID, taskID, status)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	if outcomes == nil {
		outcomes = []*Outcome{}
	}

	json.NewEncoder(w).Encode(outcomes)
}

// handleGoals handles the /api/goals endpoint
func (ws *WebServer) handleGoals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var projectID *int64
	var taskID *int64
	var goalType *string

	if pidStr := r.URL.Query().Get("project_id"); pidStr != "" {
		if pid, err := strconv.ParseInt(pidStr, 10, 64); err == nil {
			projectID = &pid
		}
	}

	if tidStr := r.URL.Query().Get("task_id"); tidStr != "" {
		if tid, err := strconv.ParseInt(tidStr, 10, 64); err == nil {
			taskID = &tid
		}
	}

	if g := r.URL.Query().Get("goal_type"); g != "" {
		goalType = &g
	}

	goals, err := ws.db.ListGoals(projectID, taskID, goalType)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	if goals == nil {
		goals = []*Goal{}
	}

	json.NewEncoder(w).Encode(goals)
}

// handleDashboard serves the main dashboard HTML
func (ws *WebServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Loom Dashboard</title>
    <style>
        :root {
            --bg-primary: #0f1419;
            --bg-secondary: #1a1f2e;
            --bg-card: #242b3d;
            --bg-hover: #2d364a;
            --text-primary: #e7e9ea;
            --text-secondary: #8899a6;
            --accent-blue: #1d9bf0;
            --accent-green: #00ba7c;
            --accent-yellow: #ffd93d;
            --accent-red: #f4212e;
            --accent-purple: #9b59b6;
            --border-color: #2f3336;
            --shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            line-height: 1.5;
            min-height: 100vh;
        }

        .app-container {
            display: grid;
            grid-template-columns: 280px 1fr;
            min-height: 100vh;
        }

        /* Sidebar */
        .sidebar {
            background: var(--bg-secondary);
            border-right: 1px solid var(--border-color);
            padding: 24px;
            display: flex;
            flex-direction: column;
            gap: 24px;
        }

        .logo {
            display: flex;
            align-items: center;
            gap: 12px;
            font-size: 24px;
            font-weight: 700;
            color: var(--accent-blue);
            padding-bottom: 16px;
            border-bottom: 1px solid var(--border-color);
        }

        .logo-icon {
            width: 40px;
            height: 40px;
            background: linear-gradient(135deg, var(--accent-blue), var(--accent-purple));
            border-radius: 10px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 20px;
        }

        .nav-section {
            display: flex;
            flex-direction: column;
            gap: 8px;
        }

        .nav-title {
            font-size: 12px;
            font-weight: 600;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 8px;
        }

        .nav-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px 16px;
            border-radius: 8px;
            cursor: pointer;
            transition: all 0.2s ease;
            color: var(--text-secondary);
        }

        .nav-item:hover, .nav-item.active {
            background: var(--bg-card);
            color: var(--text-primary);
        }

        .nav-item.active {
            border-left: 3px solid var(--accent-blue);
        }

        .nav-badge {
            margin-left: auto;
            background: var(--accent-blue);
            color: white;
            font-size: 12px;
            padding: 2px 8px;
            border-radius: 12px;
            font-weight: 600;
        }

        /* Main Content */
        .main-content {
            padding: 32px;
            overflow-y: auto;
            max-height: 100vh;
        }

        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 32px;
        }

        .header h1 {
            font-size: 28px;
            font-weight: 700;
        }

        .header-actions {
            display: flex;
            gap: 12px;
            align-items: center;
        }

        .search-box {
            display: flex;
            align-items: center;
            gap: 8px;
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 10px 16px;
            width: 300px;
        }

        .search-box input {
            border: none;
            background: none;
            color: var(--text-primary);
            font-size: 14px;
            width: 100%;
            outline: none;
        }

        .search-box input::placeholder {
            color: var(--text-secondary);
        }

        .connection-status {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 8px 16px;
            border-radius: 20px;
            font-size: 13px;
            font-weight: 500;
        }

        .connection-status.connected {
            background: rgba(0, 186, 124, 0.15);
            color: var(--accent-green);
        }

        .connection-status.disconnected {
            background: rgba(244, 33, 46, 0.15);
            color: var(--accent-red);
        }

        .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: currentColor;
        }

        .refresh-btn {
            background: var(--accent-blue);
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .refresh-btn:hover {
            background: #1a8cd8;
            transform: translateY(-1px);
        }

        /* Stats Cards */
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 32px;
        }

        .stat-card {
            background: var(--bg-card);
            border-radius: 12px;
            padding: 20px;
            border: 1px solid var(--border-color);
            transition: transform 0.2s ease, box-shadow 0.2s ease;
        }

        .stat-card:hover {
            transform: translateY(-2px);
            box-shadow: var(--shadow);
        }

        .stat-label {
            font-size: 13px;
            color: var(--text-secondary);
            margin-bottom: 8px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .stat-value {
            font-size: 32px;
            font-weight: 700;
        }

        .stat-card.projects .stat-value { color: var(--accent-blue); }
        .stat-card.tasks .stat-value { color: var(--accent-green); }
        .stat-card.problems .stat-value { color: var(--accent-red); }
        .stat-card.goals .stat-value { color: var(--accent-yellow); }

        /* Filters */
        .filters {
            display: flex;
            gap: 12px;
            margin-bottom: 24px;
            flex-wrap: wrap;
        }

        .filter-select {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 10px 16px;
            color: var(--text-primary);
            font-size: 14px;
            cursor: pointer;
            min-width: 150px;
        }

        .filter-select:focus {
            outline: none;
            border-color: var(--accent-blue);
        }

        /* Content Sections */
        .content-section {
            display: none;
            animation: fadeIn 0.3s ease;
        }

        .content-section.active {
            display: block;
        }

        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }

        .section-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
        }

        .section-title {
            font-size: 20px;
            font-weight: 600;
        }

        /* Cards Grid */
        .cards-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
            gap: 20px;
        }

        .card {
            background: var(--bg-card);
            border-radius: 12px;
            border: 1px solid var(--border-color);
            overflow: hidden;
            transition: all 0.2s ease;
        }

        .card:hover {
            border-color: var(--accent-blue);
            box-shadow: var(--shadow);
        }

        .card-header {
            padding: 16px 20px;
            border-bottom: 1px solid var(--border-color);
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
        }

        .card-title {
            font-size: 16px;
            font-weight: 600;
            margin-bottom: 4px;
        }

        .card-id {
            font-size: 12px;
            color: var(--text-secondary);
        }

        .card-body {
            padding: 16px 20px;
        }

        .card-description {
            color: var(--text-secondary);
            font-size: 14px;
            margin-bottom: 16px;
            display: -webkit-box;
            -webkit-line-clamp: 3;
            -webkit-box-orient: vertical;
            overflow: hidden;
        }

        .card-meta {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
        }

        /* Badges */
        .badge {
            display: inline-flex;
            align-items: center;
            gap: 4px;
            padding: 4px 10px;
            border-radius: 6px;
            font-size: 12px;
            font-weight: 500;
        }

        .badge.status-pending { background: rgba(139, 153, 166, 0.2); color: var(--text-secondary); }
        .badge.status-in_progress { background: rgba(29, 155, 240, 0.2); color: var(--accent-blue); }
        .badge.status-completed { background: rgba(0, 186, 124, 0.2); color: var(--accent-green); }
        .badge.status-blocked { background: rgba(244, 33, 46, 0.2); color: var(--accent-red); }
        .badge.status-open { background: rgba(255, 217, 61, 0.2); color: var(--accent-yellow); }
        .badge.status-resolved { background: rgba(0, 186, 124, 0.2); color: var(--accent-green); }

        .badge.priority-low { background: rgba(139, 153, 166, 0.2); color: var(--text-secondary); }
        .badge.priority-medium { background: rgba(255, 217, 61, 0.2); color: var(--accent-yellow); }
        .badge.priority-high { background: rgba(244, 33, 46, 0.2); color: var(--accent-red); }
        .badge.priority-urgent { background: rgba(244, 33, 46, 0.4); color: #ff6b6b; }

        .badge.type-general { background: rgba(139, 153, 166, 0.2); color: var(--text-secondary); }
        .badge.type-feature { background: rgba(155, 89, 182, 0.2); color: var(--accent-purple); }
        .badge.type-bugfix { background: rgba(244, 33, 46, 0.2); color: var(--accent-red); }
        .badge.type-chore { background: rgba(29, 155, 240, 0.2); color: var(--accent-blue); }
        .badge.type-investigation { background: rgba(255, 217, 61, 0.2); color: var(--accent-yellow); }

        .badge.goal-short_term { background: rgba(29, 155, 240, 0.2); color: var(--accent-blue); }
        .badge.goal-career { background: rgba(155, 89, 182, 0.2); color: var(--accent-purple); }
        .badge.goal-values { background: rgba(0, 186, 124, 0.2); color: var(--accent-green); }
        .badge.goal-requirement { background: rgba(255, 217, 61, 0.2); color: var(--accent-yellow); }

        /* External Link */
        .external-link {
            display: inline-flex;
            align-items: center;
            gap: 4px;
            color: var(--accent-blue);
            font-size: 13px;
            text-decoration: none;
            margin-top: 8px;
        }

        .external-link:hover {
            text-decoration: underline;
        }

        /* Card Timestamps */
        .card-footer {
            padding: 12px 20px;
            border-top: 1px solid var(--border-color);
            font-size: 12px;
            color: var(--text-secondary);
            display: flex;
            justify-content: space-between;
        }

        /* Empty State */
        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: var(--text-secondary);
        }

        .empty-state-icon {
            font-size: 48px;
            margin-bottom: 16px;
            opacity: 0.5;
        }

        .empty-state h3 {
            font-size: 18px;
            margin-bottom: 8px;
            color: var(--text-primary);
        }

        /* Task list for projects */
        .task-list {
            margin-top: 12px;
            padding-top: 12px;
            border-top: 1px solid var(--border-color);
        }

        .task-list-title {
            font-size: 13px;
            font-weight: 600;
            color: var(--text-secondary);
            margin-bottom: 8px;
        }

        .task-item {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 6px 0;
            font-size: 13px;
        }

        .task-item .badge {
            font-size: 10px;
            padding: 2px 6px;
        }

        /* Scrollbar */
        ::-webkit-scrollbar {
            width: 8px;
            height: 8px;
        }

        ::-webkit-scrollbar-track {
            background: var(--bg-secondary);
        }

        ::-webkit-scrollbar-thumb {
            background: var(--border-color);
            border-radius: 4px;
        }

        ::-webkit-scrollbar-thumb:hover {
            background: var(--text-secondary);
        }

        /* Loading State */
        .loading {
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 40px;
        }

        .spinner {
            width: 40px;
            height: 40px;
            border: 3px solid var(--border-color);
            border-top-color: var(--accent-blue);
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        /* Responsive */
        @media (max-width: 1024px) {
            .app-container {
                grid-template-columns: 1fr;
            }
            
            .sidebar {
                display: none;
            }

            .cards-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="app-container">
        <!-- Sidebar -->
        <aside class="sidebar">
            <div class="logo">
                <div class="logo-icon">🧵</div>
                <span>Loom</span>
            </div>

            <nav class="nav-section">
                <div class="nav-title">Dashboard</div>
                <div class="nav-item active" data-section="overview" onclick="switchSection('overview')">
                    <span>📊</span>
                    <span>Overview</span>
                </div>
            </nav>

            <nav class="nav-section">
                <div class="nav-title">Data</div>
                <div class="nav-item" data-section="projects" onclick="switchSection('projects')">
                    <span>📁</span>
                    <span>Projects</span>
                    <span class="nav-badge" id="projects-count">0</span>
                </div>
                <div class="nav-item" data-section="tasks" onclick="switchSection('tasks')">
                    <span>✅</span>
                    <span>Tasks</span>
                    <span class="nav-badge" id="tasks-count">0</span>
                </div>
                <div class="nav-item" data-section="problems" onclick="switchSection('problems')">
                    <span>⚠️</span>
                    <span>Problems</span>
                    <span class="nav-badge" id="problems-count">0</span>
                </div>
                <div class="nav-item" data-section="outcomes" onclick="switchSection('outcomes')">
                    <span>🎯</span>
                    <span>Outcomes</span>
                    <span class="nav-badge" id="outcomes-count">0</span>
                </div>
                <div class="nav-item" data-section="goals" onclick="switchSection('goals')">
                    <span>🏆</span>
                    <span>Goals</span>
                    <span class="nav-badge" id="goals-count">0</span>
                </div>
            </nav>
        </aside>

        <!-- Main Content -->
        <main class="main-content">
            <header class="header">
                <h1 id="page-title">Overview</h1>
                <div class="header-actions">
                    <div class="search-box">
                        <span>🔍</span>
                        <input type="text" id="search-input" placeholder="Search..." oninput="handleSearch(this.value)">
                    </div>
                    <div class="connection-status disconnected" id="connection-status">
                        <span class="status-dot"></span>
                        <span>Disconnected</span>
                    </div>
                    <button class="refresh-btn" onclick="refreshData()">⟳ Refresh</button>
                </div>
            </header>

            <!-- Stats Overview -->
            <div class="stats-grid" id="stats-grid">
                <div class="stat-card projects">
                    <div class="stat-label">Total Projects</div>
                    <div class="stat-value" id="stat-projects">0</div>
                </div>
                <div class="stat-card tasks">
                    <div class="stat-label">Total Tasks</div>
                    <div class="stat-value" id="stat-tasks">0</div>
                </div>
                <div class="stat-card problems">
                    <div class="stat-label">Open Problems</div>
                    <div class="stat-value" id="stat-problems">0</div>
                </div>
                <div class="stat-card goals">
                    <div class="stat-label">Active Goals</div>
                    <div class="stat-value" id="stat-goals">0</div>
                </div>
            </div>

            <!-- Overview Section -->
            <section class="content-section active" id="section-overview">
                <div class="section-header">
                    <h2 class="section-title">Recent Activity</h2>
                </div>
                <div class="cards-grid" id="recent-activity"></div>
            </section>

            <!-- Projects Section -->
            <section class="content-section" id="section-projects">
                <div class="section-header">
                    <h2 class="section-title">Projects</h2>
                </div>
                <div class="cards-grid" id="projects-grid"></div>
            </section>

            <!-- Tasks Section -->
            <section class="content-section" id="section-tasks">
                <div class="section-header">
                    <h2 class="section-title">Tasks</h2>
                </div>
                <div class="filters">
                    <select class="filter-select" id="task-status-filter" onchange="filterTasks()">
                        <option value="">All Statuses</option>
                        <option value="pending">Pending</option>
                        <option value="in_progress">In Progress</option>
                        <option value="completed">Completed</option>
                        <option value="blocked">Blocked</option>
                    </select>
                    <select class="filter-select" id="task-priority-filter" onchange="filterTasks()">
                        <option value="">All Priorities</option>
                        <option value="low">Low</option>
                        <option value="medium">Medium</option>
                        <option value="high">High</option>
                        <option value="urgent">Urgent</option>
                    </select>
                    <select class="filter-select" id="task-type-filter" onchange="filterTasks()">
                        <option value="">All Types</option>
                        <option value="general">General</option>
                        <option value="feature">Feature</option>
                        <option value="bugfix">Bugfix</option>
                        <option value="chore">Chore</option>
                        <option value="investigation">Investigation</option>
                    </select>
                    <select class="filter-select" id="task-project-filter" onchange="filterTasks()">
                        <option value="">All Projects</option>
                    </select>
                </div>
                <div class="cards-grid" id="tasks-grid"></div>
            </section>

            <!-- Problems Section -->
            <section class="content-section" id="section-problems">
                <div class="section-header">
                    <h2 class="section-title">Problems</h2>
                </div>
                <div class="filters">
                    <select class="filter-select" id="problem-status-filter" onchange="filterProblems()">
                        <option value="">All Statuses</option>
                        <option value="open">Open</option>
                        <option value="in_progress">In Progress</option>
                        <option value="resolved">Resolved</option>
                        <option value="blocked">Blocked</option>
                    </select>
                </div>
                <div class="cards-grid" id="problems-grid"></div>
            </section>

            <!-- Outcomes Section -->
            <section class="content-section" id="section-outcomes">
                <div class="section-header">
                    <h2 class="section-title">Outcomes</h2>
                </div>
                <div class="filters">
                    <select class="filter-select" id="outcome-status-filter" onchange="filterOutcomes()">
                        <option value="">All Statuses</option>
                        <option value="open">Open</option>
                        <option value="in_progress">In Progress</option>
                        <option value="completed">Completed</option>
                        <option value="blocked">Blocked</option>
                    </select>
                </div>
                <div class="cards-grid" id="outcomes-grid"></div>
            </section>

            <!-- Goals Section -->
            <section class="content-section" id="section-goals">
                <div class="section-header">
                    <h2 class="section-title">Goals</h2>
                </div>
                <div class="filters">
                    <select class="filter-select" id="goal-type-filter" onchange="filterGoals()">
                        <option value="">All Types</option>
                        <option value="short_term">Short Term</option>
                        <option value="career">Career</option>
                        <option value="values">Values</option>
                        <option value="requirement">Requirement</option>
                    </select>
                </div>
                <div class="cards-grid" id="goals-grid"></div>
            </section>
        </main>
    </div>

    <script>
        // Data storage
        let data = {
            projects: [],
            tasks: [],
            problems: [],
            outcomes: [],
            goals: []
        };

        let projectsMap = {};
        let searchQuery = '';
        let eventSource = null;

        // Initialize
        document.addEventListener('DOMContentLoaded', () => {
            refreshData();
            connectSSE();
        });

        // Connect to SSE endpoint
        function connectSSE() {
            if (eventSource) {
                eventSource.close();
            }

            eventSource = new EventSource('/events');

            eventSource.onopen = () => {
                updateConnectionStatus(true);
            };

            eventSource.onerror = () => {
                updateConnectionStatus(false);
                // Attempt to reconnect after 5 seconds
                setTimeout(connectSSE, 5000);
            };

            eventSource.addEventListener('connected', () => {
                updateConnectionStatus(true);
            });

            eventSource.addEventListener('refresh', () => {
                refreshData();
            });

            eventSource.addEventListener('heartbeat', () => {
                // Keep-alive heartbeat
            });
        }

        function updateConnectionStatus(connected) {
            const status = document.getElementById('connection-status');
            if (connected) {
                status.className = 'connection-status connected';
                status.innerHTML = '<span class="status-dot"></span><span>Connected</span>';
            } else {
                status.className = 'connection-status disconnected';
                status.innerHTML = '<span class="status-dot"></span><span>Disconnected</span>';
            }
        }

        // Fetch all data
        async function refreshData() {
            try {
                const [projects, tasks, problems, outcomes, goals] = await Promise.all([
                    fetch('/api/projects').then(r => r.json()),
                    fetch('/api/tasks').then(r => r.json()),
                    fetch('/api/problems').then(r => r.json()),
                    fetch('/api/outcomes').then(r => r.json()),
                    fetch('/api/goals').then(r => r.json())
                ]);

                data.projects = projects || [];
                data.tasks = tasks || [];
                data.problems = problems || [];
                data.outcomes = outcomes || [];
                data.goals = goals || [];

                // Build project map
                projectsMap = {};
                data.projects.forEach(p => projectsMap[p.id] = p);

                updateStats();
                updateProjectFilter();
                renderCurrentSection();
            } catch (err) {
                console.error('Error fetching data:', err);
            }
        }

        // Update stats
        function updateStats() {
            document.getElementById('stat-projects').textContent = data.projects.length;
            document.getElementById('stat-tasks').textContent = data.tasks.length;
            document.getElementById('stat-problems').textContent = data.problems.filter(p => p.status !== 'resolved').length;
            document.getElementById('stat-goals').textContent = data.goals.length;

            document.getElementById('projects-count').textContent = data.projects.length;
            document.getElementById('tasks-count').textContent = data.tasks.length;
            document.getElementById('problems-count').textContent = data.problems.length;
            document.getElementById('outcomes-count').textContent = data.outcomes.length;
            document.getElementById('goals-count').textContent = data.goals.length;
        }

        // Update project filter dropdown
        function updateProjectFilter() {
            const select = document.getElementById('task-project-filter');
            const currentValue = select.value;
            select.innerHTML = '<option value="">All Projects</option>';
            data.projects.forEach(p => {
                const option = document.createElement('option');
                option.value = p.id;
                option.textContent = p.name;
                select.appendChild(option);
            });
            select.value = currentValue;
        }

        // Switch section
        function switchSection(section) {
            // Update nav
            document.querySelectorAll('.nav-item').forEach(item => {
                item.classList.toggle('active', item.dataset.section === section);
            });

            // Update sections
            document.querySelectorAll('.content-section').forEach(sec => {
                sec.classList.remove('active');
            });
            document.getElementById('section-' + section).classList.add('active');

            // Update title
            const titles = {
                'overview': 'Overview',
                'projects': 'Projects',
                'tasks': 'Tasks',
                'problems': 'Problems',
                'outcomes': 'Outcomes',
                'goals': 'Goals'
            };
            document.getElementById('page-title').textContent = titles[section];

            renderCurrentSection();
        }

        // Render current section
        function renderCurrentSection() {
            const activeSection = document.querySelector('.content-section.active');
            if (!activeSection) return;

            const sectionId = activeSection.id.replace('section-', '');
            switch(sectionId) {
                case 'overview':
                    renderOverview();
                    break;
                case 'projects':
                    renderProjects();
                    break;
                case 'tasks':
                    filterTasks();
                    break;
                case 'problems':
                    filterProblems();
                    break;
                case 'outcomes':
                    filterOutcomes();
                    break;
                case 'goals':
                    filterGoals();
                    break;
            }
        }

        // Search handler
        function handleSearch(query) {
            searchQuery = query.toLowerCase();
            renderCurrentSection();
        }

        // Filter by search
        function filterBySearch(items, fields) {
            if (!searchQuery) return items;
            return items.filter(item => {
                return fields.some(field => {
                    const value = item[field];
                    return value && value.toString().toLowerCase().includes(searchQuery);
                });
            });
        }

        // Format date
        function formatDate(dateStr) {
            if (!dateStr) return '';
            const date = new Date(dateStr);
            return date.toLocaleDateString('en-US', { 
                year: 'numeric', 
                month: 'short', 
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
            });
        }

        // Render Overview
        function renderOverview() {
            const grid = document.getElementById('recent-activity');
            
            // Combine all items and sort by updated_at
            const allItems = [
                ...data.projects.map(p => ({ ...p, type: 'project' })),
                ...data.tasks.map(t => ({ ...t, type: 'task' })),
                ...data.problems.map(p => ({ ...p, type: 'problem' })),
                ...data.outcomes.map(o => ({ ...o, type: 'outcome' })),
                ...data.goals.map(g => ({ ...g, type: 'goal' }))
            ].sort((a, b) => new Date(b.updated_at) - new Date(a.updated_at))
             .slice(0, 12);

            if (allItems.length === 0) {
                grid.innerHTML = renderEmptyState('No data yet', 'Use the MCP tools to create projects and tasks.');
                return;
            }

            grid.innerHTML = allItems.map(item => {
                switch(item.type) {
                    case 'project':
                        return renderProjectCard(item);
                    case 'task':
                        return renderTaskCard(item);
                    case 'problem':
                        return renderProblemCard(item);
                    case 'outcome':
                        return renderOutcomeCard(item);
                    case 'goal':
                        return renderGoalCard(item);
                }
            }).join('');
        }

        // Render Projects
        function renderProjects() {
            const grid = document.getElementById('projects-grid');
            const filtered = filterBySearch(data.projects, ['name', 'description']);

            if (filtered.length === 0) {
                grid.innerHTML = renderEmptyState('No projects found', 'Create a project using create_project MCP tool.');
                return;
            }

            grid.innerHTML = filtered.map(renderProjectCard).join('');
        }

        function renderProjectCard(project) {
            const tasks = data.tasks.filter(t => t.project_id === project.id);
            const tasksByStatus = {
                pending: tasks.filter(t => t.status === 'pending').length,
                in_progress: tasks.filter(t => t.status === 'in_progress').length,
                completed: tasks.filter(t => t.status === 'completed').length,
                blocked: tasks.filter(t => t.status === 'blocked').length
            };

            return ` + "`" + `
                <div class="card">
                    <div class="card-header">
                        <div>
                            <div class="card-title">📁 ${escapeHtml(project.name)}</div>
                            <div class="card-id">ID: ${project.id}</div>
                        </div>
                    </div>
                    <div class="card-body">
                        <div class="card-description">${escapeHtml(project.description) || 'No description'}</div>
                        <div class="card-meta">
                            <span class="badge status-pending">Pending: ${tasksByStatus.pending}</span>
                            <span class="badge status-in_progress">In Progress: ${tasksByStatus.in_progress}</span>
                            <span class="badge status-completed">Completed: ${tasksByStatus.completed}</span>
                        </div>
                        ${project.external_link ? ` + "`" + `<a href="${escapeHtml(project.external_link)}" target="_blank" class="external-link">🔗 External Link</a>` + "`" + ` : ''}
                    </div>
                    <div class="card-footer">
                        <span>Created: ${formatDate(project.created_at)}</span>
                        <span>Updated: ${formatDate(project.updated_at)}</span>
                    </div>
                </div>
            ` + "`" + `;
        }

        // Filter and render tasks
        function filterTasks() {
            const status = document.getElementById('task-status-filter').value;
            const priority = document.getElementById('task-priority-filter').value;
            const type = document.getElementById('task-type-filter').value;
            const projectId = document.getElementById('task-project-filter').value;

            let filtered = [...data.tasks];
            if (status) filtered = filtered.filter(t => t.status === status);
            if (priority) filtered = filtered.filter(t => t.priority === priority);
            if (type) filtered = filtered.filter(t => t.task_type === type);
            if (projectId) filtered = filtered.filter(t => String(t.project_id) === projectId);

            filtered = filterBySearch(filtered, ['title', 'description']);

            const grid = document.getElementById('tasks-grid');
            if (filtered.length === 0) {
                grid.innerHTML = renderEmptyState('No tasks found', 'Create a task using create_task MCP tool.');
                return;
            }

            grid.innerHTML = filtered.map(renderTaskCard).join('');
        }

        function renderTaskCard(task) {
            const project = projectsMap[task.project_id];
            return ` + "`" + `
                <div class="card">
                    <div class="card-header">
                        <div>
                            <div class="card-title">✅ ${escapeHtml(task.title)}</div>
                            <div class="card-id">ID: ${task.id} • Project: ${project ? escapeHtml(project.name) : 'Unknown'}</div>
                        </div>
                    </div>
                    <div class="card-body">
                        <div class="card-description">${escapeHtml(task.description) || 'No description'}</div>
                        <div class="card-meta">
                            <span class="badge status-${task.status}">${task.status.replace('_', ' ')}</span>
                            <span class="badge priority-${task.priority}">Priority: ${task.priority}</span>
                            <span class="badge type-${task.task_type}">${task.task_type}</span>
                        </div>
                        ${task.external_link ? ` + "`" + `<a href="${escapeHtml(task.external_link)}" target="_blank" class="external-link">🔗 External Link</a>` + "`" + ` : ''}
                    </div>
                    <div class="card-footer">
                        <span>Created: ${formatDate(task.created_at)}</span>
                        <span>Updated: ${formatDate(task.updated_at)}</span>
                    </div>
                </div>
            ` + "`" + `;
        }

        // Filter and render problems
        function filterProblems() {
            const status = document.getElementById('problem-status-filter').value;
            let filtered = [...data.problems];
            if (status) filtered = filtered.filter(p => p.status === status);
            filtered = filterBySearch(filtered, ['title', 'description']);

            const grid = document.getElementById('problems-grid');
            if (filtered.length === 0) {
                grid.innerHTML = renderEmptyState('No problems found', 'Create a problem using create_problem MCP tool.');
                return;
            }

            grid.innerHTML = filtered.map(renderProblemCard).join('');
        }

        function renderProblemCard(problem) {
            const project = problem.project_id ? projectsMap[problem.project_id] : null;
            return ` + "`" + `
                <div class="card">
                    <div class="card-header">
                        <div>
                            <div class="card-title">⚠️ ${escapeHtml(problem.title)}</div>
                            <div class="card-id">ID: ${problem.id}${project ? ' • Project: ' + escapeHtml(project.name) : ''}</div>
                        </div>
                    </div>
                    <div class="card-body">
                        <div class="card-description">${escapeHtml(problem.description) || 'No description'}</div>
                        <div class="card-meta">
                            <span class="badge status-${problem.status}">${problem.status.replace('_', ' ')}</span>
                        </div>
                    </div>
                    <div class="card-footer">
                        <span>Created: ${formatDate(problem.created_at)}</span>
                        <span>Updated: ${formatDate(problem.updated_at)}</span>
                    </div>
                </div>
            ` + "`" + `;
        }

        // Filter and render outcomes
        function filterOutcomes() {
            const status = document.getElementById('outcome-status-filter').value;
            let filtered = [...data.outcomes];
            if (status) filtered = filtered.filter(o => o.status === status);
            filtered = filterBySearch(filtered, ['title', 'description']);

            const grid = document.getElementById('outcomes-grid');
            if (filtered.length === 0) {
                grid.innerHTML = renderEmptyState('No outcomes found', 'Create an outcome using create_outcome MCP tool.');
                return;
            }

            grid.innerHTML = filtered.map(renderOutcomeCard).join('');
        }

        function renderOutcomeCard(outcome) {
            const project = projectsMap[outcome.project_id];
            return ` + "`" + `
                <div class="card">
                    <div class="card-header">
                        <div>
                            <div class="card-title">🎯 ${escapeHtml(outcome.title)}</div>
                            <div class="card-id">ID: ${outcome.id} • Project: ${project ? escapeHtml(project.name) : 'Unknown'}</div>
                        </div>
                    </div>
                    <div class="card-body">
                        <div class="card-description">${escapeHtml(outcome.description) || 'No description'}</div>
                        <div class="card-meta">
                            <span class="badge status-${outcome.status}">${outcome.status.replace('_', ' ')}</span>
                        </div>
                    </div>
                    <div class="card-footer">
                        <span>Created: ${formatDate(outcome.created_at)}</span>
                        <span>Updated: ${formatDate(outcome.updated_at)}</span>
                    </div>
                </div>
            ` + "`" + `;
        }

        // Filter and render goals
        function filterGoals() {
            const type = document.getElementById('goal-type-filter').value;
            let filtered = [...data.goals];
            if (type) filtered = filtered.filter(g => g.goal_type === type);
            filtered = filterBySearch(filtered, ['title', 'description']);

            const grid = document.getElementById('goals-grid');
            if (filtered.length === 0) {
                grid.innerHTML = renderEmptyState('No goals found', 'Create a goal using create_goal MCP tool.');
                return;
            }

            grid.innerHTML = filtered.map(renderGoalCard).join('');
        }

        function renderGoalCard(goal) {
            const project = goal.project_id ? projectsMap[goal.project_id] : null;
            return ` + "`" + `
                <div class="card">
                    <div class="card-header">
                        <div>
                            <div class="card-title">🏆 ${escapeHtml(goal.title)}</div>
                            <div class="card-id">ID: ${goal.id}${project ? ' • Project: ' + escapeHtml(project.name) : ''}</div>
                        </div>
                    </div>
                    <div class="card-body">
                        <div class="card-description">${escapeHtml(goal.description) || 'No description'}</div>
                        <div class="card-meta">
                            <span class="badge goal-${goal.goal_type}">${goal.goal_type.replace('_', ' ')}</span>
                        </div>
                    </div>
                    <div class="card-footer">
                        <span>Created: ${formatDate(goal.created_at)}</span>
                        <span>Updated: ${formatDate(goal.updated_at)}</span>
                    </div>
                </div>
            ` + "`" + `;
        }

        // Empty state
        function renderEmptyState(title, message) {
            return ` + "`" + `
                <div class="empty-state">
                    <div class="empty-state-icon">📭</div>
                    <h3>${title}</h3>
                    <p>${message}</p>
                </div>
            ` + "`" + `;
        }

        // Escape HTML to prevent XSS
        function escapeHtml(text) {
            if (!text) return '';
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
    </script>
</body>
</html>`
