// This file is for local development.
// For Cloud Functions, the function.go file is used instead.

package main

import (
	"fmt"
	"log"
	"net/http"

	"example.com/webhook-receiver/internal/config"
	"example.com/webhook-receiver/internal/domain"
	"example.com/webhook-receiver/internal/handlers"
	"example.com/webhook-receiver/internal/repositories"
	"example.com/webhook-receiver/internal/services"

	"context"

	firebase "firebase.google.com/go/v4"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Firebase
	ctx := context.Background()
	firebaseApp, err := firebase.NewApp(ctx, &firebase.Config{
		DatabaseURL: cfg.FirebaseDatabaseURL,
	})
	if err != nil {
		log.Fatalf("Failed to initialize Firebase: %v", err)
	}

	dbClient, err := firebaseApp.Database(ctx)
	if err != nil {
		log.Fatalf("Failed to get Firebase database client: %v", err)
	}

	// Create dependencies
	logger := services.NewSimpleLogger()
	validator := domain.NewHMACValidator(cfg.WebhookSecret)
	writer := repositories.NewFirebaseRepository(dbClient)

	// Compose service
	webhookService := services.NewWebhookService(validator, writer, logger)

	// Create handler
	handler := handlers.NewWebhookHandler(webhookService, logger)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("Starting webhook server", "addr", addr)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
