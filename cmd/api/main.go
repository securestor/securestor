package main

import (
	"log"

	"github.com/securestor/securestor/internal/api"
	"github.com/securestor/securestor/internal/config"
	"github.com/securestor/securestor/internal/database"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize default data (admin user, tenant, roles, scopes)
	// Only runs on first startup when no tenants exist
	if err := database.InitializeDefaultData(db); err != nil {
		log.Fatalf("Failed to initialize default data: %v", err)
	}

	// Initialize API server
	server := api.NewServer(cfg, db)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
