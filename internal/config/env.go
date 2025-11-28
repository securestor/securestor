package config

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/joho/godotenv"
)

var (
	envOnce   sync.Once
	envLoaded bool
)

// LoadEnvOnce loads the .env file only once during the application lifecycle
// This prevents multiple modules from trying to load the same file
func LoadEnvOnce() {
	envOnce.Do(func() {
		loadEnvironment()
	})
}

// loadEnvironment handles the actual environment loading with proper fallbacks
func loadEnvironment() {
	// Try to load .env from multiple possible locations
	envPaths := []string{
		".env",       // Current directory
		"../.env",    // Parent directory
		"../../.env", // Go up two levels
		filepath.Join(os.Getenv("APP_ROOT"), ".env"), // APP_ROOT env var
	}

	var loaded bool
	for _, path := range envPaths {
		if _, err := os.Stat(path); err == nil {
			if err := godotenv.Load(path); err == nil {
				log.Printf("Environment loaded from: %s", path)
				loaded = true
				break
			}
		}
	}

	// Determine if we're running in a containerized environment
	isContainer := isContainerEnvironment()

	if !loaded {
		if isContainer {
			log.Println("Running in container - using environment variables")
		} else if isDevelopment() {
			log.Println("Warning: .env file not found - using environment variables or defaults")
		}
	}

	envLoaded = true
}

// isContainerEnvironment detects if we're running in a container
func isContainerEnvironment() bool {
	// Check for common container indicators
	indicators := []string{
		"/.dockerenv",        // Docker
		"/run/.containerenv", // Podman
	}

	for _, indicator := range indicators {
		if _, err := os.Stat(indicator); err == nil {
			return true
		}
	}

	// Check for container-specific environment variables
	containerEnvVars := []string{
		"KUBERNETES_SERVICE_HOST", // Kubernetes
		"DOCKER_CONTAINER",        // Docker
		"CONTAINER_ID",            // Generic container
	}

	for _, envVar := range containerEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	// Check if critical env vars are already set (likely container deployment)
	if os.Getenv("DATABASE_URL") != "" && os.Getenv("PORT") != "" {
		return true
	}

	return false
}

// isDevelopment checks if we're in development mode
func isDevelopment() bool {
	env := os.Getenv("ENVIRONMENT")
	return env == "" || env == "development" || env == "dev"
}

// GetEnvWithFallback gets an environment variable with a fallback value
func GetEnvWithFallback(key, fallback string) string {
	LoadEnvOnce() // Ensure env is loaded

	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// MustGetEnv gets an environment variable or panics if not found
func MustGetEnv(key string) string {
	LoadEnvOnce() // Ensure env is loaded

	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Required environment variable %s is not set", key)
	}
	return value
}

// GetEnvBool gets an environment variable as boolean with fallback
func GetEnvBool(key string, fallback bool) bool {
	LoadEnvOnce()

	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value == "true" || value == "1" || value == "yes" || value == "on"
}

// IsEnvLoaded returns whether the environment has been loaded
func IsEnvLoaded() bool {
	return envLoaded
}
