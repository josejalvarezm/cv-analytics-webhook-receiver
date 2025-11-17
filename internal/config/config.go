package config

import (
	"fmt"
	"os"
)

// Config holds application configuration
type Config struct {
	WebhookSecret       string
	FirebaseProjectID   string
	FirebaseDatabaseURL string
	Port                string
	Environment         string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		WebhookSecret:       os.Getenv("WEBHOOK_SECRET"),
		FirebaseProjectID:   os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseDatabaseURL: os.Getenv("FIREBASE_DATABASE_URL"),
		Port:                getEnvOrDefault("PORT", "8080"),
		Environment:         getEnvOrDefault("ENVIRONMENT", "development"),
	}

	// Validate required fields
	if cfg.WebhookSecret == "" {
		return nil, fmt.Errorf("WEBHOOK_SECRET environment variable is required")
	}
	if cfg.FirebaseDatabaseURL == "" {
		return nil, fmt.Errorf("FIREBASE_DATABASE_URL environment variable is required")
	}

	return cfg, nil
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
