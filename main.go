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
	webAddr := flag.String("addr", ":8080", "Web server address (default :8080)")
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

	// Start the web server
	log.Printf("Loom web dashboard starting at http://%s with database at: %s", *webAddr, dbPath)
	ws := NewWebServer(db, *webAddr)
	if err := ws.Start(); err != nil {
		log.Fatal("Failed to start web server:", err)
	}
}
