package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

var db *Database

func main() {
	// Parse command-line flags
	webAddr := flag.String("addr", ":8080", "API server address (default :8080)")
	dashboardAddr := flag.String("web-addr", ":3000", "Website server address (default :3000)")
	mcpAddr := flag.String("mcp-addr", ":8081", "MCP Streamable HTTP server address (default :8081)")
	flag.Parse()

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

	// Start MCP Streamable HTTP server in a goroutine
	mcpServer := NewMCPServer(db)
	go func() {
		if err := StartMCPServer(mcpServer, *mcpAddr); err != nil {
			log.Fatalf("MCP server failed: %v", err)
		}
	}()

	// Start the API and dashboard servers
	log.Printf("Loom starting - API at http://%s, Dashboard at http://%s, MCP at http://%s/mcp, database at: %s", *webAddr, *dashboardAddr, *mcpAddr, dbPath)
	ws := NewWebServer(db, *webAddr, *dashboardAddr)
	if err := ws.Start(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
