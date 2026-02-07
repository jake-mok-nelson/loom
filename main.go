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

	// Create MCP handler to be mounted on the API server
	mcpServer := NewMCPServer(db)
	mcpHandler := NewMCPHandler(mcpServer)

	// Start the API (with MCP) and dashboard servers
	log.Printf("Loom starting - API at http://%s, MCP at http://%s/sse, Dashboard at http://%s, database at: %s", *webAddr, *webAddr, *dashboardAddr, dbPath)
	ws := NewWebServer(db, *webAddr, *dashboardAddr, mcpHandler)
	if err := ws.Start(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
