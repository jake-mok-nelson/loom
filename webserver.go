package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

// WebServer handles HTTP requests and SSE for the Loom dashboard
type WebServer struct {
	db         *Database
	addr       string
	webAddr    string
	clients    map[chan string]bool
	clientsMux sync.RWMutex
}

// NewWebServer creates a new web server instance
func NewWebServer(db *Database, addr string, webAddr string) *WebServer {
	return &WebServer{
		db:      db,
		addr:    addr,
		webAddr: webAddr,
		clients: make(map[chan string]bool),
	}
}

// Start begins the API and website servers on separate ports
func (ws *WebServer) Start() error {
	// API server mux - serves REST API and SSE endpoints
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/projects", ws.handleProjects)
	apiMux.HandleFunc("/api/tasks", ws.handleTasks)
	apiMux.HandleFunc("/api/problems", ws.handleProblems)
	apiMux.HandleFunc("/api/outcomes", ws.handleOutcomes)
	apiMux.HandleFunc("/api/goals", ws.handleGoals)
	apiMux.HandleFunc("/api/voice", ws.handleVoice)
	apiMux.HandleFunc("/events", ws.handleSSE)

	// Website server mux - serves the dashboard UI
	webMux := http.NewServeMux()
	webMux.HandleFunc("/", ws.handleDashboard)

	// Start the website server in a goroutine
	go func() {
		log.Printf("Starting Loom website server at http://%s", ws.webAddr)
		if err := http.ListenAndServe(ws.webAddr, webMux); err != nil {
			log.Fatalf("Website server failed: %v", err)
		}
	}()

	// Start the API server (blocking)
	log.Printf("Starting Loom API server at http://%s", ws.addr)
	return http.ListenAndServe(ws.addr, apiMux)
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
	var assignee *string

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

	if a := r.URL.Query().Get("assignee"); a != "" {
		assignee = &a
	}

	problems, err := ws.db.ListProblems(projectID, taskID, status, assignee)
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
	var assignee *string

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

	if a := r.URL.Query().Get("assignee"); a != "" {
		assignee = &a
	}

	goals, err := ws.db.ListGoals(projectID, taskID, goalType, assignee)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	if goals == nil {
		goals = []*Goal{}
	}

	json.NewEncoder(w).Encode(goals)
}

// handleVoice handles text-to-speech conversion
// Accepts POST requests with JSON body containing "text" field
// Returns WAV audio file
func (ws *WebServer) handleVoice(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the JSON request body
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "Text field is required", http.StatusBadRequest)
		return
	}

	// Validate text length to prevent abuse
	if len(req.Text) > 5000 {
		http.Error(w, "Text too long (max 5000 characters)", http.StatusBadRequest)
		return
	}

	// Create temporary file securely
	tmpFile, err := os.CreateTemp("", "loom-tts-*.wav")
	if err != nil {
		log.Printf("Failed to create temporary file: %v", err)
		http.Error(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}
	tmpFilePath := tmpFile.Name()
	tmpFile.Close()
	defer func() {
		// Clean up temporary file
		if err := os.Remove(tmpFilePath); err != nil {
			log.Printf("Warning: Failed to remove temporary file %s: %v", tmpFilePath, err)
		}
	}()

	// Use echogarden to synthesize speech
	// Note: Kokoro is the preferred engine but requires model download from HuggingFace
	// Using espeak as a working alternative with British English voice
	// The text is passed as a command argument - echogarden handles escaping internally
	cmd := exec.Command("echogarden", "speak", req.Text, tmpFilePath, "--engine=espeak", "--language=en-GB")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("TTS generation failed: %v\nOutput: %s", err, string(output))
		http.Error(w, "Failed to generate speech", http.StatusInternalServerError)
		return
	}

	// Echogarden appends _001 to the output filename
	actualPath := tmpFilePath[:len(tmpFilePath)-4] + "_001.wav"
	defer func() {
		// Clean up the actual output file
		if err := os.Remove(actualPath); err != nil {
			log.Printf("Warning: Failed to remove actual file %s: %v", actualPath, err)
		}
	}()

	// Read the generated audio file
	audioData, err := os.ReadFile(actualPath)
	if err != nil {
		log.Printf("Failed to read audio file: %v", err)
		http.Error(w, "Failed to read generated audio", http.StatusInternalServerError)
		return
	}

	// Send the audio file as response
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(audioData)))
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(audioData)
}

// apiBaseURL returns the base URL for the API server based on the request host and API address.
func (ws *WebServer) apiBaseURL(r *http.Request) string {
	host := r.Host
	// Extract the hostname (without port) from the request
	hostname := host
	if colonIdx := findLastColon(hostname); colonIdx != -1 {
		hostname = hostname[:colonIdx]
	}
	// Extract the port from the API address
	apiPort := ws.addr
	if colonIdx := findLastColon(apiPort); colonIdx != -1 {
		apiPort = apiPort[colonIdx+1:]
	}
	return fmt.Sprintf("http://%s:%s", hostname, apiPort)
}

// findLastColon returns the index of the last colon in a string, or -1 if not found.
func findLastColon(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}

// handleDashboard serves the main dashboard HTML
func (ws *WebServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	apiBase := ws.apiBaseURL(r)
	// Inject the API base URL as a JavaScript variable before the dashboard script
	html := fmt.Sprintf(`<script>var API_BASE_URL = %q;</script>`, apiBase) + dashboardHTML
	w.Write([]byte(html))
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
            --node-project: #1d9bf0;
            --node-task: #00ba7c;
            --node-problem: #f4212e;
            --node-outcome: #ffd93d;
            --node-goal: #9b59b6;
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

        .voice-toggle-btn {
            background: var(--bg-card);
            color: var(--text-primary);
            border: 1px solid var(--border-color);
            padding: 10px 16px;
            border-radius: 8px;
            font-size: 18px;
            cursor: pointer;
            transition: all 0.2s ease;
            display: flex;
            align-items: center;
            justify-content: center;
        }

        .voice-toggle-btn:hover {
            background: var(--bg-hover);
            transform: translateY(-1px);
        }

        .voice-toggle-btn.muted {
            opacity: 0.5;
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
        .badge.status-active { background: rgba(0, 186, 124, 0.2); color: var(--accent-green); }
        .badge.status-planning { background: rgba(29, 155, 240, 0.2); color: var(--accent-blue); }
        .badge.status-on_hold { background: rgba(255, 217, 61, 0.2); color: var(--accent-yellow); }
        .badge.status-archived { background: rgba(139, 153, 166, 0.2); color: var(--text-secondary); }

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

        /* Graph View Styles */
        .graph-container {
            width: 100%;
            height: calc(100vh - 250px);
            min-height: 500px;
            background: var(--bg-card);
            border-radius: 12px;
            border: 1px solid var(--border-color);
            position: relative;
            overflow: hidden;
        }

        #graph-canvas {
            width: 100%;
            height: 100%;
            cursor: grab;
        }

        #graph-canvas:active {
            cursor: grabbing;
        }

        .graph-controls {
            display: flex;
            gap: 12px;
            margin-bottom: 16px;
            flex-wrap: wrap;
            align-items: center;
        }

        .layout-select {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 10px 16px;
            color: var(--text-primary);
            font-size: 14px;
            cursor: pointer;
            min-width: 180px;
        }

        .layout-select:focus {
            outline: none;
            border-color: var(--accent-blue);
        }

        .graph-btn {
            background: var(--bg-card);
            color: var(--text-primary);
            border: 1px solid var(--border-color);
            padding: 10px 16px;
            border-radius: 8px;
            font-size: 14px;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .graph-btn:hover {
            background: var(--bg-hover);
            border-color: var(--accent-blue);
        }

        .graph-btn.active {
            background: var(--accent-blue);
            border-color: var(--accent-blue);
        }

        .graph-legend {
            display: flex;
            gap: 16px;
            flex-wrap: wrap;
            margin-top: 16px;
            padding: 12px 16px;
            background: var(--bg-card);
            border-radius: 8px;
            border: 1px solid var(--border-color);
        }

        .legend-item {
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 13px;
            color: var(--text-secondary);
        }

        .legend-dot {
            width: 12px;
            height: 12px;
            border-radius: 50%;
        }

        .legend-dot.project { background: var(--node-project); }
        .legend-dot.task { background: var(--node-task); }
        .legend-dot.problem { background: var(--node-problem); }
        .legend-dot.outcome { background: var(--node-outcome); }
        .legend-dot.goal { background: var(--node-goal); }

        .node-tooltip {
            position: absolute;
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 12px 16px;
            font-size: 13px;
            pointer-events: none;
            z-index: 1000;
            max-width: 300px;
            box-shadow: var(--shadow);
            display: none;
        }

        .node-tooltip.visible {
            display: block;
        }

        .tooltip-title {
            font-weight: 600;
            margin-bottom: 4px;
            color: var(--text-primary);
        }

        .tooltip-type {
            font-size: 11px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 8px;
        }

        .tooltip-type.project { color: var(--node-project); }
        .tooltip-type.task { color: var(--node-task); }
        .tooltip-type.problem { color: var(--node-problem); }
        .tooltip-type.outcome { color: var(--node-outcome); }
        .tooltip-type.goal { color: var(--node-goal); }

        .tooltip-desc {
            color: var(--text-secondary);
            font-size: 12px;
        }

        .graph-filter-group {
            display: flex;
            gap: 8px;
            align-items: center;
        }

        .graph-filter-label {
            font-size: 13px;
            color: var(--text-secondary);
        }

        .filter-toggle {
            display: flex;
            align-items: center;
            gap: 4px;
            padding: 6px 12px;
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 6px;
            font-size: 12px;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .filter-toggle.active {
            background: var(--bg-hover);
            border-color: var(--accent-blue);
        }

        .filter-toggle input {
            margin: 0;
            cursor: pointer;
        }

        /* Clickable Card Styles */
        .card.clickable {
            cursor: pointer;
        }

        .card.clickable:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 25px rgba(0, 0, 0, 0.4);
        }

        /* Related Items Modal */
        .related-modal {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.8);
            z-index: 1000;
            overflow-y: auto;
        }

        .related-modal.active {
            display: flex;
            justify-content: center;
            padding: 40px 20px;
        }

        .related-modal-content {
            background: var(--bg-primary);
            border-radius: 16px;
            width: 100%;
            max-width: 900px;
            max-height: calc(100vh - 80px);
            overflow-y: auto;
            border: 1px solid var(--border-color);
        }

        .related-modal-header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            padding: 24px;
            border-bottom: 1px solid var(--border-color);
            position: sticky;
            top: 0;
            background: var(--bg-primary);
            z-index: 10;
        }

        .related-modal-title {
            font-size: 22px;
            font-weight: 600;
            margin-bottom: 8px;
        }

        .related-modal-subtitle {
            font-size: 14px;
            color: var(--text-secondary);
        }

        .related-modal-close {
            background: none;
            border: none;
            color: var(--text-secondary);
            font-size: 28px;
            cursor: pointer;
            padding: 0 8px;
            line-height: 1;
        }

        .related-modal-close:hover {
            color: var(--text-primary);
        }

        .related-modal-body {
            padding: 24px;
        }

        .related-section {
            margin-bottom: 32px;
        }

        .related-section:last-child {
            margin-bottom: 0;
        }

        .related-section-header {
            display: flex;
            align-items: center;
            gap: 10px;
            margin-bottom: 16px;
        }

        .related-section-title {
            font-size: 16px;
            font-weight: 600;
            color: var(--text-primary);
        }

        .related-section-count {
            background: var(--bg-hover);
            color: var(--text-secondary);
            padding: 2px 10px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: 500;
        }

        .related-items-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
            gap: 16px;
        }

        .related-item {
            background: var(--bg-card);
            border-radius: 10px;
            padding: 16px;
            border: 1px solid var(--border-color);
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .related-item:hover {
            border-color: var(--accent-blue);
            background: var(--bg-hover);
        }

        .related-item-header {
            display: flex;
            align-items: flex-start;
            gap: 10px;
            margin-bottom: 8px;
        }

        .related-item-icon {
            font-size: 18px;
        }

        .related-item-title {
            font-size: 14px;
            font-weight: 500;
            color: var(--text-primary);
            flex: 1;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }

        .related-item-desc {
            font-size: 13px;
            color: var(--text-secondary);
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
            overflow: hidden;
            margin-bottom: 10px;
        }

        .related-item-meta {
            display: flex;
            flex-wrap: wrap;
            gap: 6px;
        }

        .related-empty {
            text-align: center;
            padding: 24px;
            color: var(--text-secondary);
            font-size: 14px;
        }

        .detail-field {
            margin-bottom: 16px;
        }

        .detail-label {
            font-size: 12px;
            color: var(--text-secondary);
            margin-bottom: 4px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .detail-value {
            font-size: 14px;
            color: var(--text-primary);
        }

        .detail-badges {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
            margin-bottom: 16px;
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
                <div class="nav-item" data-section="graph" onclick="switchSection('graph')">
                    <span>🔗</span>
                    <span>Graph View</span>
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
                    <button class="voice-toggle-btn" id="voice-toggle" onclick="toggleVoice()" title="Toggle voice notifications">
                        <span id="voice-icon">🔊</span>
                    </button>
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

            <!-- Graph View Section -->
            <section class="content-section" id="section-graph">
                <div class="section-header">
                    <h2 class="section-title">Entity Relationships</h2>
                </div>
                <div class="graph-controls">
                    <select class="layout-select" id="graph-layout" onchange="changeLayout()">
                        <option value="force">Force-Directed Layout</option>
                        <option value="hierarchical">Hierarchical Layout</option>
                        <option value="radial">Radial Layout</option>
                        <option value="grid">Grid Layout</option>
                    </select>
                    <button class="graph-btn" onclick="resetGraphView()">⟳ Reset View</button>
                    <button class="graph-btn" onclick="zoomIn()">+ Zoom In</button>
                    <button class="graph-btn" onclick="zoomOut()">- Zoom Out</button>
                    <div class="graph-filter-group">
                        <span class="graph-filter-label">Show:</span>
                        <label class="filter-toggle active">
                            <input type="checkbox" checked onchange="toggleEntityType(event, 'project')"> Projects
                        </label>
                        <label class="filter-toggle active">
                            <input type="checkbox" checked onchange="toggleEntityType(event, 'task')"> Tasks
                        </label>
                        <label class="filter-toggle active">
                            <input type="checkbox" checked onchange="toggleEntityType(event, 'problem')"> Problems
                        </label>
                        <label class="filter-toggle active">
                            <input type="checkbox" checked onchange="toggleEntityType(event, 'outcome')"> Outcomes
                        </label>
                        <label class="filter-toggle active">
                            <input type="checkbox" checked onchange="toggleEntityType(event, 'goal')"> Goals
                        </label>
                    </div>
                </div>
                <div class="graph-container">
                    <canvas id="graph-canvas"></canvas>
                    <div class="node-tooltip" id="node-tooltip">
                        <div class="tooltip-type" id="tooltip-type"></div>
                        <div class="tooltip-title" id="tooltip-title"></div>
                        <div class="tooltip-desc" id="tooltip-desc"></div>
                    </div>
                </div>
                <div class="graph-legend">
                    <div class="legend-item">
                        <span class="legend-dot project"></span>
                        <span>Project</span>
                    </div>
                    <div class="legend-item">
                        <span class="legend-dot task"></span>
                        <span>Task</span>
                    </div>
                    <div class="legend-item">
                        <span class="legend-dot problem"></span>
                        <span>Problem</span>
                    </div>
                    <div class="legend-item">
                        <span class="legend-dot outcome"></span>
                        <span>Outcome</span>
                    </div>
                    <div class="legend-item">
                        <span class="legend-dot goal"></span>
                        <span>Goal</span>
                    </div>
                </div>
            </section>

            <!-- Projects Section -->
            <section class="content-section" id="section-projects">
                <div class="section-header">
                    <h2 class="section-title">Projects</h2>
                </div>
                <div class="filters">
                    <select class="filter-select" id="project-status-filter" onchange="filterProjects()">
                        <option value="">All Statuses</option>
                        <option value="active">Active</option>
                        <option value="planning">Planning</option>
                        <option value="on_hold">On Hold</option>
                        <option value="completed">Completed</option>
                        <option value="archived">Archived</option>
                    </select>
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

    <!-- Related Items Modal -->
    <div class="related-modal" id="related-modal">
        <div class="related-modal-content">
            <div class="related-modal-header">
                <div>
                    <div class="related-modal-title" id="modal-title"></div>
                    <div class="related-modal-subtitle" id="modal-subtitle"></div>
                </div>
                <button class="related-modal-close" onclick="closeRelatedModal()">&times;</button>
            </div>
            <div class="related-modal-body" id="modal-body">
            </div>
        </div>
    </div>

    <script>
        // Constants
        const MODAL_TRANSITION_DELAY = 100; // Delay in ms for modal close/open transitions

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
        let voiceMuted = localStorage.getItem('voiceMuted') === 'true';

        // Initialize
        document.addEventListener('DOMContentLoaded', () => {
            refreshData();
            connectSSE();
            updateVoiceIcon();
        });

        // Voice functionality
        function toggleVoice() {
            voiceMuted = !voiceMuted;
            localStorage.setItem('voiceMuted', voiceMuted);
            updateVoiceIcon();
        }

        function updateVoiceIcon() {
            const btn = document.getElementById('voice-toggle');
            const icon = document.getElementById('voice-icon');
            if (voiceMuted) {
                icon.textContent = '🔇';
                btn.classList.add('muted');
                btn.title = 'Voice notifications muted (click to unmute)';
            } else {
                icon.textContent = '🔊';
                btn.classList.remove('muted');
                btn.title = 'Voice notifications enabled (click to mute)';
            }
        }

        async function speakText(text) {
            if (voiceMuted) {
                return;
            }

            try {
                const response = await fetch(API_BASE_URL + '/api/voice', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ text: text })
                });

                if (!response.ok) {
                    console.error('Voice API error:', response.statusText);
                    return;
                }

                const audioBlob = await response.blob();
                const audioUrl = URL.createObjectURL(audioBlob);
                const audio = new Audio(audioUrl);
                
                audio.onended = () => {
                    URL.revokeObjectURL(audioUrl);
                };

                audio.onerror = (error) => {
                    console.error('Audio playback error:', error);
                    URL.revokeObjectURL(audioUrl);
                };

                await audio.play();
            } catch (error) {
                console.error('Failed to play voice:', error);
            }
        }

        // Connect to SSE endpoint
        function connectSSE() {
            if (eventSource) {
                eventSource.close();
            }

            eventSource = new EventSource(API_BASE_URL + '/events');

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
                    fetch(API_BASE_URL + '/api/projects').then(r => r.json()),
                    fetch(API_BASE_URL + '/api/tasks').then(r => r.json()),
                    fetch(API_BASE_URL + '/api/problems').then(r => r.json()),
                    fetch(API_BASE_URL + '/api/outcomes').then(r => r.json()),
                    fetch(API_BASE_URL + '/api/goals').then(r => r.json())
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
                'graph': 'Graph View',
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
                case 'graph':
                    initGraph();
                    break;
                case 'projects':
                    filterProjects();
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
        // Filter and render projects
        function filterProjects() {
            const status = document.getElementById('project-status-filter').value;
            
            let filtered = [...data.projects];
            if (status) filtered = filtered.filter(p => p.status === status);
            
            filtered = filterBySearch(filtered, ['name', 'description']);

            const grid = document.getElementById('projects-grid');
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
                <div class="card clickable" onclick="showRelatedItems('project', ${project.id})">
                    <div class="card-header">
                        <div>
                            <div class="card-title">📁 ${escapeHtml(project.name)}</div>
                            <div class="card-id">ID: ${project.id}</div>
                        </div>
                    </div>
                    <div class="card-body">
                        <div class="card-description">${escapeHtml(project.description) || 'No description'}</div>
                        <div class="card-meta">
                            <span class="badge status-${project.status}">${project.status.replace('_', ' ')}</span>
                            <span class="badge status-pending">Pending: ${tasksByStatus.pending}</span>
                            <span class="badge status-in_progress">In Progress: ${tasksByStatus.in_progress}</span>
                            <span class="badge status-completed">Completed: ${tasksByStatus.completed}</span>
                        </div>
                        ${project.external_link ? ` + "`" + `<a href="${escapeHtml(project.external_link)}" target="_blank" class="external-link" onclick="event.stopPropagation()">🔗 External Link</a>` + "`" + ` : ''}
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
                <div class="card clickable" onclick="showRelatedItems('task', ${task.id})">
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
                        ${task.external_link ? ` + "`" + `<a href="${escapeHtml(task.external_link)}" target="_blank" class="external-link" onclick="event.stopPropagation()">🔗 External Link</a>` + "`" + ` : ''}
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
                <div class="card clickable" onclick="showRelatedItems('problem', ${problem.id})">
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
                <div class="card clickable" onclick="showRelatedItems('outcome', ${outcome.id})">
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
                <div class="card clickable" onclick="showRelatedItems('goal', ${goal.id})">
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

        // ==================== RELATED ITEMS MODAL ====================

        // Show related items modal when clicking on a card
        function showRelatedItems(type, id) {
            const modal = document.getElementById('related-modal');
            const titleEl = document.getElementById('modal-title');
            const subtitleEl = document.getElementById('modal-subtitle');
            const bodyEl = document.getElementById('modal-body');

            let item, title, subtitle, bodyHtml;
            const icons = { project: '📁', task: '✅', problem: '⚠️', outcome: '🎯', goal: '🏆' };

            switch(type) {
                case 'project':
                    item = data.projects.find(p => p.id === id);
                    if (!item) return;
                    title = icons.project + ' ' + escapeHtml(item.name);
                    subtitle = 'Project Details & Related Items';
                    bodyHtml = renderProjectRelatedItems(item);
                    break;
                case 'task':
                    item = data.tasks.find(t => t.id === id);
                    if (!item) return;
                    title = icons.task + ' ' + escapeHtml(item.title);
                    subtitle = 'Task Details & Related Items';
                    bodyHtml = renderTaskRelatedItems(item);
                    break;
                case 'problem':
                    item = data.problems.find(p => p.id === id);
                    if (!item) return;
                    title = icons.problem + ' ' + escapeHtml(item.title);
                    subtitle = 'Problem Details & Related Items';
                    bodyHtml = renderProblemRelatedItems(item);
                    break;
                case 'outcome':
                    item = data.outcomes.find(o => o.id === id);
                    if (!item) return;
                    title = icons.outcome + ' ' + escapeHtml(item.title);
                    subtitle = 'Outcome Details & Related Items';
                    bodyHtml = renderOutcomeRelatedItems(item);
                    break;
                case 'goal':
                    item = data.goals.find(g => g.id === id);
                    if (!item) return;
                    title = icons.goal + ' ' + escapeHtml(item.title);
                    subtitle = 'Goal Details & Related Items';
                    bodyHtml = renderGoalRelatedItems(item);
                    break;
                default:
                    return;
            }

            titleEl.innerHTML = title;
            subtitleEl.textContent = subtitle;
            bodyEl.innerHTML = bodyHtml;
            modal.classList.add('active');
            document.body.style.overflow = 'hidden';
        }

        // Close modal
        function closeRelatedModal() {
            const modal = document.getElementById('related-modal');
            modal.classList.remove('active');
            document.body.style.overflow = '';
        }

        // Close modal on escape key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') closeRelatedModal();
        });

        // Close modal when clicking outside
        document.getElementById('related-modal').addEventListener('click', (e) => {
            if (e.target.id === 'related-modal') closeRelatedModal();
        });

        // Render related items for a project
        function renderProjectRelatedItems(project) {
            const tasks = data.tasks.filter(t => t.project_id === project.id);
            const problems = data.problems.filter(p => p.project_id === project.id);
            const outcomes = data.outcomes.filter(o => o.project_id === project.id);
            const goals = data.goals.filter(g => g.project_id === project.id);

            let html = ` + "`" + `
                <div class="detail-field">
                    <div class="detail-label">Description</div>
                    <div class="detail-value">${escapeHtml(project.description) || 'No description'}</div>
                </div>
                ${project.external_link ? ` + "`" + `
                <div class="detail-field">
                    <div class="detail-label">External Link</div>
                    <div class="detail-value"><a href="${escapeHtml(project.external_link)}" target="_blank" style="color: var(--accent-blue);">🔗 ${escapeHtml(project.external_link)}</a></div>
                </div>
                ` + "`" + ` : ''}
                <div class="detail-field">
                    <div class="detail-label">Timestamps</div>
                    <div class="detail-value">Created: ${formatDate(project.created_at)} • Updated: ${formatDate(project.updated_at)}</div>
                </div>
            ` + "`" + `;

            html += renderRelatedSection('Tasks', '✅', tasks, renderRelatedTask);
            html += renderRelatedSection('Problems', '⚠️', problems, renderRelatedProblem);
            html += renderRelatedSection('Outcomes', '🎯', outcomes, renderRelatedOutcome);
            html += renderRelatedSection('Goals', '🏆', goals, renderRelatedGoal);

            return html;
        }

        // Render related items for a task
        function renderTaskRelatedItems(task) {
            const project = projectsMap[task.project_id];
            const problems = data.problems.filter(p => p.task_id === task.id);
            const outcomes = data.outcomes.filter(o => o.task_id === task.id);
            const goals = data.goals.filter(g => g.task_id === task.id);

            let html = ` + "`" + `
                <div class="detail-badges">
                    <span class="badge status-${task.status}">${task.status.replace('_', ' ')}</span>
                    <span class="badge priority-${task.priority}">Priority: ${task.priority}</span>
                    <span class="badge type-${task.task_type}">${task.task_type}</span>
                </div>
                <div class="detail-field">
                    <div class="detail-label">Description</div>
                    <div class="detail-value">${escapeHtml(task.description) || 'No description'}</div>
                </div>
                ${task.external_link ? ` + "`" + `
                <div class="detail-field">
                    <div class="detail-label">External Link</div>
                    <div class="detail-value"><a href="${escapeHtml(task.external_link)}" target="_blank" style="color: var(--accent-blue);">🔗 ${escapeHtml(task.external_link)}</a></div>
                </div>
                ` + "`" + ` : ''}
                <div class="detail-field">
                    <div class="detail-label">Timestamps</div>
                    <div class="detail-value">Created: ${formatDate(task.created_at)} • Updated: ${formatDate(task.updated_at)}</div>
                </div>
            ` + "`" + `;

            if (project) {
                html += renderRelatedSection('Parent Project', '📁', [project], renderRelatedProject);
            }
            html += renderRelatedSection('Problems', '⚠️', problems, renderRelatedProblem);
            html += renderRelatedSection('Outcomes', '🎯', outcomes, renderRelatedOutcome);
            html += renderRelatedSection('Goals', '🏆', goals, renderRelatedGoal);

            return html;
        }

        // Render related items for a problem
        function renderProblemRelatedItems(problem) {
            const project = problem.project_id ? projectsMap[problem.project_id] : null;
            const task = problem.task_id ? data.tasks.find(t => t.id === problem.task_id) : null;

            let html = ` + "`" + `
                <div class="detail-badges">
                    <span class="badge status-${problem.status}">${problem.status.replace('_', ' ')}</span>
                </div>
                <div class="detail-field">
                    <div class="detail-label">Description</div>
                    <div class="detail-value">${escapeHtml(problem.description) || 'No description'}</div>
                </div>
                <div class="detail-field">
                    <div class="detail-label">Timestamps</div>
                    <div class="detail-value">Created: ${formatDate(problem.created_at)} • Updated: ${formatDate(problem.updated_at)}</div>
                </div>
            ` + "`" + `;

            if (project) {
                html += renderRelatedSection('Linked Project', '📁', [project], renderRelatedProject);
            }
            if (task) {
                html += renderRelatedSection('Linked Task', '✅', [task], renderRelatedTask);
            }

            return html;
        }

        // Render related items for an outcome
        function renderOutcomeRelatedItems(outcome) {
            const project = projectsMap[outcome.project_id];
            const task = outcome.task_id ? data.tasks.find(t => t.id === outcome.task_id) : null;

            let html = ` + "`" + `
                <div class="detail-badges">
                    <span class="badge status-${outcome.status}">${outcome.status.replace('_', ' ')}</span>
                </div>
                <div class="detail-field">
                    <div class="detail-label">Description</div>
                    <div class="detail-value">${escapeHtml(outcome.description) || 'No description'}</div>
                </div>
                <div class="detail-field">
                    <div class="detail-label">Timestamps</div>
                    <div class="detail-value">Created: ${formatDate(outcome.created_at)} • Updated: ${formatDate(outcome.updated_at)}</div>
                </div>
            ` + "`" + `;

            if (project) {
                html += renderRelatedSection('Linked Project', '📁', [project], renderRelatedProject);
            }
            if (task) {
                html += renderRelatedSection('Linked Task', '✅', [task], renderRelatedTask);
            }

            return html;
        }

        // Render related items for a goal
        function renderGoalRelatedItems(goal) {
            const project = goal.project_id ? projectsMap[goal.project_id] : null;
            const task = goal.task_id ? data.tasks.find(t => t.id === goal.task_id) : null;

            let html = ` + "`" + `
                <div class="detail-badges">
                    <span class="badge goal-${goal.goal_type}">${goal.goal_type.replace('_', ' ')}</span>
                </div>
                <div class="detail-field">
                    <div class="detail-label">Description</div>
                    <div class="detail-value">${escapeHtml(goal.description) || 'No description'}</div>
                </div>
                <div class="detail-field">
                    <div class="detail-label">Timestamps</div>
                    <div class="detail-value">Created: ${formatDate(goal.created_at)} • Updated: ${formatDate(goal.updated_at)}</div>
                </div>
            ` + "`" + `;

            if (project) {
                html += renderRelatedSection('Linked Project', '📁', [project], renderRelatedProject);
            }
            if (task) {
                html += renderRelatedSection('Linked Task', '✅', [task], renderRelatedTask);
            }

            return html;
        }

        // Helper function to render a related section
        function renderRelatedSection(title, icon, items, renderFn) {
            if (!items || items.length === 0) {
                return ` + "`" + `
                    <div class="related-section">
                        <div class="related-section-header">
                            <span class="related-section-title">${icon} ${title}</span>
                            <span class="related-section-count">0</span>
                        </div>
                        <div class="related-empty">No ${title.toLowerCase()} linked</div>
                    </div>
                ` + "`" + `;
            }

            return ` + "`" + `
                <div class="related-section">
                    <div class="related-section-header">
                        <span class="related-section-title">${icon} ${title}</span>
                        <span class="related-section-count">${items.length}</span>
                    </div>
                    <div class="related-items-grid">
                        ${items.map(renderFn).join('')}
                    </div>
                </div>
            ` + "`" + `;
        }

        // Render a related project item
        function renderRelatedProject(project) {
            const taskCount = data.tasks.filter(t => t.project_id === project.id).length;
            return ` + "`" + `
                <div class="related-item" onclick="event.stopPropagation(); closeRelatedModal(); setTimeout(() => showRelatedItems('project', ${project.id}), MODAL_TRANSITION_DELAY);">
                    <div class="related-item-header">
                        <span class="related-item-icon">📁</span>
                        <span class="related-item-title">${escapeHtml(project.name)}</span>
                    </div>
                    <div class="related-item-desc">${escapeHtml(project.description) || 'No description'}</div>
                    <div class="related-item-meta">
                        <span class="badge" style="background: rgba(29, 155, 240, 0.2); color: var(--accent-blue);">${taskCount} tasks</span>
                    </div>
                </div>
            ` + "`" + `;
        }

        // Render a related task item
        function renderRelatedTask(task) {
            return ` + "`" + `
                <div class="related-item" onclick="event.stopPropagation(); closeRelatedModal(); setTimeout(() => showRelatedItems('task', ${task.id}), MODAL_TRANSITION_DELAY);">
                    <div class="related-item-header">
                        <span class="related-item-icon">✅</span>
                        <span class="related-item-title">${escapeHtml(task.title)}</span>
                    </div>
                    <div class="related-item-desc">${escapeHtml(task.description) || 'No description'}</div>
                    <div class="related-item-meta">
                        <span class="badge status-${task.status}">${task.status.replace('_', ' ')}</span>
                        <span class="badge priority-${task.priority}">${task.priority}</span>
                    </div>
                </div>
            ` + "`" + `;
        }

        // Render a related problem item
        function renderRelatedProblem(problem) {
            return ` + "`" + `
                <div class="related-item" onclick="event.stopPropagation(); closeRelatedModal(); setTimeout(() => showRelatedItems('problem', ${problem.id}), MODAL_TRANSITION_DELAY);">
                    <div class="related-item-header">
                        <span class="related-item-icon">⚠️</span>
                        <span class="related-item-title">${escapeHtml(problem.title)}</span>
                    </div>
                    <div class="related-item-desc">${escapeHtml(problem.description) || 'No description'}</div>
                    <div class="related-item-meta">
                        <span class="badge status-${problem.status}">${problem.status.replace('_', ' ')}</span>
                    </div>
                </div>
            ` + "`" + `;
        }

        // Render a related outcome item
        function renderRelatedOutcome(outcome) {
            return ` + "`" + `
                <div class="related-item" onclick="event.stopPropagation(); closeRelatedModal(); setTimeout(() => showRelatedItems('outcome', ${outcome.id}), MODAL_TRANSITION_DELAY);">
                    <div class="related-item-header">
                        <span class="related-item-icon">🎯</span>
                        <span class="related-item-title">${escapeHtml(outcome.title)}</span>
                    </div>
                    <div class="related-item-desc">${escapeHtml(outcome.description) || 'No description'}</div>
                    <div class="related-item-meta">
                        <span class="badge status-${outcome.status}">${outcome.status.replace('_', ' ')}</span>
                    </div>
                </div>
            ` + "`" + `;
        }

        // Render a related goal item
        function renderRelatedGoal(goal) {
            return ` + "`" + `
                <div class="related-item" onclick="event.stopPropagation(); closeRelatedModal(); setTimeout(() => showRelatedItems('goal', ${goal.id}), MODAL_TRANSITION_DELAY);">
                    <div class="related-item-header">
                        <span class="related-item-icon">🏆</span>
                        <span class="related-item-title">${escapeHtml(goal.title)}</span>
                    </div>
                    <div class="related-item-desc">${escapeHtml(goal.description) || 'No description'}</div>
                    <div class="related-item-meta">
                        <span class="badge goal-${goal.goal_type}">${goal.goal_type.replace('_', ' ')}</span>
                    </div>
                </div>
            ` + "`" + `;
        }

        // ==================== GRAPH VIEW ====================
        
        // Graph state
        let graphCanvas = null;
        let graphCtx = null;
        let graphNodes = [];
        let graphEdges = [];
        let graphScale = 1;
        let graphOffset = { x: 0, y: 0 };
        let isDragging = false;
        let dragStart = { x: 0, y: 0 };
        let selectedNode = null;
        let hoveredNode = null;
        let animationId = null;
        let currentLayout = 'force';
        let visibleTypes = { project: true, task: true, problem: true, outcome: true, goal: true };
        
        // Node colors
        const nodeColors = {
            project: '#1d9bf0',
            task: '#00ba7c',
            problem: '#f4212e',
            outcome: '#ffd93d',
            goal: '#9b59b6'
        };
        
        // Node sizes
        const nodeSizes = {
            project: 24,
            task: 18,
            problem: 16,
            outcome: 16,
            goal: 16
        };
        
        // Force simulation constants
        const REPULSION_STRENGTH = 2000;      // Repulsion force between nodes
        const IDEAL_EDGE_LENGTH = 100;        // Ideal distance between connected nodes
        const SPRING_STRENGTH = 0.05;         // Spring attraction strength for edges
        const MAX_LABEL_LENGTH = 20;          // Maximum characters before truncation
        const TRUNCATE_LENGTH = 18;           // Length to truncate label to

        // Initialize graph
        function initGraph() {
            graphCanvas = document.getElementById('graph-canvas');
            if (!graphCanvas) return;
            
            graphCtx = graphCanvas.getContext('2d');
            
            // Set canvas size
            resizeCanvas();
            window.addEventListener('resize', resizeCanvas);
            
            // Build graph data
            buildGraphData();
            
            // Apply initial layout
            applyLayout(currentLayout);
            
            // Add event listeners
            graphCanvas.addEventListener('mousedown', handleMouseDown);
            graphCanvas.addEventListener('mousemove', handleMouseMove);
            graphCanvas.addEventListener('mouseup', handleMouseUp);
            graphCanvas.addEventListener('mouseleave', handleMouseLeave);
            graphCanvas.addEventListener('wheel', handleWheel);
            graphCanvas.addEventListener('dblclick', handleDoubleClick);
            
            // Start render loop
            if (animationId) cancelAnimationFrame(animationId);
            renderGraph();
        }
        
        function resizeCanvas() {
            if (!graphCanvas) return;
            const container = graphCanvas.parentElement;
            graphCanvas.width = container.clientWidth;
            graphCanvas.height = container.clientHeight;
        }
        
        function buildGraphData() {
            graphNodes = [];
            graphEdges = [];
            
            // Create nodes for each entity type
            data.projects.forEach(p => {
                graphNodes.push({
                    id: 'project-' + p.id,
                    type: 'project',
                    entityId: p.id,
                    label: p.name,
                    description: p.description || '',
                    x: 0,
                    y: 0,
                    vx: 0,
                    vy: 0
                });
            });
            
            data.tasks.forEach(t => {
                graphNodes.push({
                    id: 'task-' + t.id,
                    type: 'task',
                    entityId: t.id,
                    label: t.title,
                    description: t.description || '',
                    projectId: t.project_id,
                    x: 0,
                    y: 0,
                    vx: 0,
                    vy: 0
                });
                // Link to project
                graphEdges.push({
                    source: 'project-' + t.project_id,
                    target: 'task-' + t.id
                });
            });
            
            data.problems.forEach(p => {
                graphNodes.push({
                    id: 'problem-' + p.id,
                    type: 'problem',
                    entityId: p.id,
                    label: p.title,
                    description: p.description || '',
                    projectId: p.project_id,
                    taskId: p.task_id,
                    x: 0,
                    y: 0,
                    vx: 0,
                    vy: 0
                });
                // Link to project or task
                if (p.task_id) {
                    graphEdges.push({
                        source: 'task-' + p.task_id,
                        target: 'problem-' + p.id
                    });
                } else if (p.project_id) {
                    graphEdges.push({
                        source: 'project-' + p.project_id,
                        target: 'problem-' + p.id
                    });
                }
            });
            
            data.outcomes.forEach(o => {
                graphNodes.push({
                    id: 'outcome-' + o.id,
                    type: 'outcome',
                    entityId: o.id,
                    label: o.title,
                    description: o.description || '',
                    projectId: o.project_id,
                    taskId: o.task_id,
                    x: 0,
                    y: 0,
                    vx: 0,
                    vy: 0
                });
                // Link to project or task
                if (o.task_id) {
                    graphEdges.push({
                        source: 'task-' + o.task_id,
                        target: 'outcome-' + o.id
                    });
                } else if (o.project_id) {
                    graphEdges.push({
                        source: 'project-' + o.project_id,
                        target: 'outcome-' + o.id
                    });
                }
            });
            
            data.goals.forEach(g => {
                graphNodes.push({
                    id: 'goal-' + g.id,
                    type: 'goal',
                    entityId: g.id,
                    label: g.title,
                    description: g.description || '',
                    projectId: g.project_id,
                    taskId: g.task_id,
                    x: 0,
                    y: 0,
                    vx: 0,
                    vy: 0
                });
                // Link to project or task
                if (g.task_id) {
                    graphEdges.push({
                        source: 'task-' + g.task_id,
                        target: 'goal-' + g.id
                    });
                } else if (g.project_id) {
                    graphEdges.push({
                        source: 'project-' + g.project_id,
                        target: 'goal-' + g.id
                    });
                }
            });
        }
        
        function applyLayout(layout) {
            currentLayout = layout;
            const width = graphCanvas ? graphCanvas.width : 800;
            const height = graphCanvas ? graphCanvas.height : 600;
            const centerX = width / 2;
            const centerY = height / 2;
            
            switch(layout) {
                case 'force':
                    applyForceLayout();
                    break;
                case 'hierarchical':
                    applyHierarchicalLayout(centerX, centerY, width, height);
                    break;
                case 'radial':
                    applyRadialLayout(centerX, centerY);
                    break;
                case 'grid':
                    applyGridLayout(centerX, centerY, width, height);
                    break;
            }
        }
        
        function applyForceLayout() {
            const width = graphCanvas ? graphCanvas.width : 800;
            const height = graphCanvas ? graphCanvas.height : 600;
            
            // Initialize with random positions
            graphNodes.forEach(node => {
                node.x = Math.random() * (width - 100) + 50;
                node.y = Math.random() * (height - 100) + 50;
                node.vx = 0;
                node.vy = 0;
            });
            
            // Run force simulation
            simulateForces(100);
        }
        
        function simulateForces(iterations) {
            const width = graphCanvas ? graphCanvas.width : 800;
            const height = graphCanvas ? graphCanvas.height : 600;
            const nodeMap = {};
            graphNodes.forEach(n => nodeMap[n.id] = n);
            
            for (let i = 0; i < iterations; i++) {
                const alpha = 1 - i / iterations;
                
                // Repulsion between all nodes
                for (let j = 0; j < graphNodes.length; j++) {
                    for (let k = j + 1; k < graphNodes.length; k++) {
                        const nodeA = graphNodes[j];
                        const nodeB = graphNodes[k];
                        const dx = nodeB.x - nodeA.x;
                        const dy = nodeB.y - nodeA.y;
                        const dist = Math.sqrt(dx * dx + dy * dy) || 1;
                        const force = REPULSION_STRENGTH / (dist * dist);
                        const fx = (dx / dist) * force * alpha;
                        const fy = (dy / dist) * force * alpha;
                        nodeA.vx -= fx;
                        nodeA.vy -= fy;
                        nodeB.vx += fx;
                        nodeB.vy += fy;
                    }
                }
                
                // Attraction along edges
                graphEdges.forEach(edge => {
                    const source = nodeMap[edge.source];
                    const target = nodeMap[edge.target];
                    if (!source || !target) return;
                    const dx = target.x - source.x;
                    const dy = target.y - source.y;
                    const dist = Math.sqrt(dx * dx + dy * dy) || 1;
                    const force = (dist - IDEAL_EDGE_LENGTH) * SPRING_STRENGTH * alpha;
                    const fx = (dx / dist) * force;
                    const fy = (dy / dist) * force;
                    source.vx += fx;
                    source.vy += fy;
                    target.vx -= fx;
                    target.vy -= fy;
                });
                
                // Center gravity
                graphNodes.forEach(node => {
                    node.vx += (width / 2 - node.x) * 0.001 * alpha;
                    node.vy += (height / 2 - node.y) * 0.001 * alpha;
                });
                
                // Apply velocities with damping
                graphNodes.forEach(node => {
                    node.x += node.vx * 0.8;
                    node.y += node.vy * 0.8;
                    node.vx *= 0.9;
                    node.vy *= 0.9;
                    // Keep in bounds
                    node.x = Math.max(50, Math.min(width - 50, node.x));
                    node.y = Math.max(50, Math.min(height - 50, node.y));
                });
            }
        }
        
        function applyHierarchicalLayout(centerX, centerY, width, height) {
            // Group by type hierarchy: projects -> tasks -> (problems, outcomes, goals)
            const levels = {
                project: [],
                task: [],
                other: [] // problems, outcomes, goals
            };
            
            graphNodes.forEach(node => {
                if (node.type === 'project') levels.project.push(node);
                else if (node.type === 'task') levels.task.push(node);
                else levels.other.push(node);
            });
            
            const padding = 80;
            const levelHeight = (height - 2 * padding) / 3;
            
            // Position projects at top
            levels.project.forEach((node, i) => {
                const spacing = (width - 2 * padding) / (levels.project.length + 1);
                node.x = padding + spacing * (i + 1);
                node.y = padding;
            });
            
            // Position tasks in middle
            levels.task.forEach((node, i) => {
                const spacing = (width - 2 * padding) / (levels.task.length + 1);
                node.x = padding + spacing * (i + 1);
                node.y = padding + levelHeight;
            });
            
            // Position others at bottom
            levels.other.forEach((node, i) => {
                const spacing = (width - 2 * padding) / (levels.other.length + 1);
                node.x = padding + spacing * (i + 1);
                node.y = padding + levelHeight * 2;
            });
        }
        
        function applyRadialLayout(centerX, centerY) {
            // Projects in center ring, tasks in next ring, others in outer ring
            const rings = {
                project: { nodes: [], radius: 80 },
                task: { nodes: [], radius: 180 },
                other: { nodes: [], radius: 280 }
            };
            
            graphNodes.forEach(node => {
                if (node.type === 'project') rings.project.nodes.push(node);
                else if (node.type === 'task') rings.task.nodes.push(node);
                else rings.other.nodes.push(node);
            });
            
            Object.values(rings).forEach(ring => {
                ring.nodes.forEach((node, i) => {
                    const angle = (2 * Math.PI * i) / ring.nodes.length - Math.PI / 2;
                    node.x = centerX + ring.radius * Math.cos(angle);
                    node.y = centerY + ring.radius * Math.sin(angle);
                });
            });
        }
        
        function applyGridLayout(centerX, centerY, width, height) {
            const padding = 60;
            const cols = Math.ceil(Math.sqrt(graphNodes.length));
            const cellWidth = (width - 2 * padding) / cols;
            const cellHeight = (height - 2 * padding) / Math.ceil(graphNodes.length / cols);
            
            graphNodes.forEach((node, i) => {
                const col = i % cols;
                const row = Math.floor(i / cols);
                node.x = padding + cellWidth * (col + 0.5);
                node.y = padding + cellHeight * (row + 0.5);
            });
        }
        
        function renderGraph() {
            if (!graphCtx || !graphCanvas) return;
            
            const ctx = graphCtx;
            const width = graphCanvas.width;
            const height = graphCanvas.height;
            
            // Clear canvas
            ctx.fillStyle = '#242b3d';
            ctx.fillRect(0, 0, width, height);
            
            ctx.save();
            ctx.translate(graphOffset.x, graphOffset.y);
            ctx.scale(graphScale, graphScale);
            
            // Get visible nodes
            const visibleNodes = graphNodes.filter(n => visibleTypes[n.type]);
            const visibleNodeIds = new Set(visibleNodes.map(n => n.id));
            
            // Draw edges
            ctx.lineWidth = 1.5 / graphScale;
            graphEdges.forEach(edge => {
                if (!visibleNodeIds.has(edge.source) || !visibleNodeIds.has(edge.target)) return;
                const source = graphNodes.find(n => n.id === edge.source);
                const target = graphNodes.find(n => n.id === edge.target);
                if (!source || !target) return;
                
                ctx.beginPath();
                ctx.strokeStyle = 'rgba(139, 153, 166, 0.4)';
                ctx.moveTo(source.x, source.y);
                ctx.lineTo(target.x, target.y);
                ctx.stroke();
                
                // Draw arrow
                const angle = Math.atan2(target.y - source.y, target.x - source.x);
                const targetRadius = nodeSizes[target.type] + 5;
                const arrowX = target.x - Math.cos(angle) * targetRadius;
                const arrowY = target.y - Math.sin(angle) * targetRadius;
                const arrowSize = 8 / graphScale;
                
                ctx.beginPath();
                ctx.fillStyle = 'rgba(139, 153, 166, 0.6)';
                ctx.moveTo(arrowX, arrowY);
                ctx.lineTo(
                    arrowX - arrowSize * Math.cos(angle - Math.PI / 6),
                    arrowY - arrowSize * Math.sin(angle - Math.PI / 6)
                );
                ctx.lineTo(
                    arrowX - arrowSize * Math.cos(angle + Math.PI / 6),
                    arrowY - arrowSize * Math.sin(angle + Math.PI / 6)
                );
                ctx.closePath();
                ctx.fill();
            });
            
            // Draw nodes
            visibleNodes.forEach(node => {
                const radius = nodeSizes[node.type];
                const isHovered = hoveredNode === node;
                const isSelected = selectedNode === node;
                
                // Node shadow
                if (isHovered || isSelected) {
                    ctx.beginPath();
                    ctx.arc(node.x, node.y, radius + 4, 0, Math.PI * 2);
                    ctx.fillStyle = nodeColors[node.type] + '40';
                    ctx.fill();
                }
                
                // Node circle
                ctx.beginPath();
                ctx.arc(node.x, node.y, radius, 0, Math.PI * 2);
                ctx.fillStyle = nodeColors[node.type];
                ctx.fill();
                
                if (isSelected) {
                    ctx.strokeStyle = '#fff';
                    ctx.lineWidth = 3 / graphScale;
                    ctx.stroke();
                }
                
                // Node icon
                ctx.fillStyle = '#fff';
                ctx.font = ` + "`" + `${Math.round(radius * 0.8)}px Arial` + "`" + `;
                ctx.textAlign = 'center';
                ctx.textBaseline = 'middle';
                const icons = { project: '📁', task: '✅', problem: '⚠', outcome: '🎯', goal: '🏆' };
                ctx.fillText(icons[node.type], node.x, node.y);
                
                // Node label
                ctx.font = ` + "`" + `${Math.round(11 / graphScale)}px -apple-system, sans-serif` + "`" + `;
                ctx.fillStyle = '#e7e9ea';
                ctx.textAlign = 'center';
                const label = node.label.length > MAX_LABEL_LENGTH ? node.label.substring(0, TRUNCATE_LENGTH) + '...' : node.label;
                ctx.fillText(label, node.x, node.y + radius + 14 / graphScale);
            });
            
            ctx.restore();
            
            animationId = requestAnimationFrame(renderGraph);
        }
        
        function handleMouseDown(e) {
            const rect = graphCanvas.getBoundingClientRect();
            const x = (e.clientX - rect.left - graphOffset.x) / graphScale;
            const y = (e.clientY - rect.top - graphOffset.y) / graphScale;
            
            // Check if clicking on a node
            const node = findNodeAt(x, y);
            if (node) {
                selectedNode = node;
                isDragging = true;
                dragStart = { x: e.clientX - node.x * graphScale - graphOffset.x, y: e.clientY - node.y * graphScale - graphOffset.y };
            } else {
                // Start panning
                selectedNode = null;
                isDragging = true;
                dragStart = { x: e.clientX - graphOffset.x, y: e.clientY - graphOffset.y };
            }
        }
        
        function handleMouseMove(e) {
            const rect = graphCanvas.getBoundingClientRect();
            const x = (e.clientX - rect.left - graphOffset.x) / graphScale;
            const y = (e.clientY - rect.top - graphOffset.y) / graphScale;
            
            // Update hovered node
            hoveredNode = findNodeAt(x, y);
            
            // Show tooltip
            if (hoveredNode) {
                const tooltip = document.getElementById('node-tooltip');
                const typeEl = document.getElementById('tooltip-type');
                const titleEl = document.getElementById('tooltip-title');
                const descEl = document.getElementById('tooltip-desc');
                
                typeEl.textContent = hoveredNode.type.toUpperCase();
                typeEl.className = 'tooltip-type ' + hoveredNode.type;
                titleEl.textContent = hoveredNode.label;
                descEl.textContent = hoveredNode.description || 'No description';
                
                tooltip.style.left = (e.clientX - rect.left + 15) + 'px';
                tooltip.style.top = (e.clientY - rect.top + 15) + 'px';
                tooltip.classList.add('visible');
            } else {
                document.getElementById('node-tooltip').classList.remove('visible');
            }
            
            if (!isDragging) return;
            
            if (selectedNode) {
                // Drag node
                selectedNode.x = (e.clientX - dragStart.x - graphOffset.x) / graphScale;
                selectedNode.y = (e.clientY - dragStart.y - graphOffset.y) / graphScale;
            } else {
                // Pan view
                graphOffset.x = e.clientX - dragStart.x;
                graphOffset.y = e.clientY - dragStart.y;
            }
        }
        
        function handleMouseUp() {
            isDragging = false;
        }
        
        function handleMouseLeave() {
            isDragging = false;
            hoveredNode = null;
            document.getElementById('node-tooltip').classList.remove('visible');
        }
        
        function handleWheel(e) {
            e.preventDefault();
            const rect = graphCanvas.getBoundingClientRect();
            const mouseX = e.clientX - rect.left;
            const mouseY = e.clientY - rect.top;
            
            const zoomFactor = e.deltaY > 0 ? 0.9 : 1.1;
            const newScale = Math.max(0.2, Math.min(3, graphScale * zoomFactor));
            
            // Zoom towards mouse position
            graphOffset.x = mouseX - (mouseX - graphOffset.x) * (newScale / graphScale);
            graphOffset.y = mouseY - (mouseY - graphOffset.y) * (newScale / graphScale);
            graphScale = newScale;
        }
        
        function handleDoubleClick(e) {
            const rect = graphCanvas.getBoundingClientRect();
            const x = (e.clientX - rect.left - graphOffset.x) / graphScale;
            const y = (e.clientY - rect.top - graphOffset.y) / graphScale;
            
            const node = findNodeAt(x, y);
            if (node) {
                // Navigate to entity section
                const sectionMap = {
                    project: 'projects',
                    task: 'tasks',
                    problem: 'problems',
                    outcome: 'outcomes',
                    goal: 'goals'
                };
                switchSection(sectionMap[node.type]);
            }
        }
        
        function findNodeAt(x, y) {
            const visibleNodes = graphNodes.filter(n => visibleTypes[n.type]);
            for (let i = visibleNodes.length - 1; i >= 0; i--) {
                const node = visibleNodes[i];
                const radius = nodeSizes[node.type];
                const dx = x - node.x;
                const dy = y - node.y;
                if (dx * dx + dy * dy <= radius * radius) {
                    return node;
                }
            }
            return null;
        }
        
        function changeLayout() {
            const layout = document.getElementById('graph-layout').value;
            applyLayout(layout);
        }
        
        function resetGraphView() {
            graphScale = 1;
            graphOffset = { x: 0, y: 0 };
            selectedNode = null;
            applyLayout(currentLayout);
        }
        
        function zoomIn() {
            graphScale = Math.min(3, graphScale * 1.2);
        }
        
        function zoomOut() {
            graphScale = Math.max(0.2, graphScale / 1.2);
        }
        
        function toggleEntityType(event, type) {
            visibleTypes[type] = !visibleTypes[type];
            const toggle = event.target.closest('.filter-toggle');
            toggle.classList.toggle('active', visibleTypes[type]);
        }
    </script>
</body>
</html>`
