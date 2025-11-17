// Package main contains the Cloud Function entry point
package main

import (
	"context"
	"log"
	"net/http"

	firebase "firebase.google.com/go/v4"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"

	"github.com/josejalvarezm/cv-analytics-webhook-receiver/internal/config"
	"github.com/josejalvarezm/cv-analytics-webhook-receiver/internal/domain"
	"github.com/josejalvarezm/cv-analytics-webhook-receiver/internal/handlers"
	"github.com/josejalvarezm/cv-analytics-webhook-receiver/internal/repositories"
	"github.com/josejalvarezm/cv-analytics-webhook-receiver/internal/services"
)

var webhookHandler http.Handler

// init initializes the Cloud Function (runs once during startup)
func init() {
	functions.HTTP("AnalyticsWebhook", AnalyticsWebhook)

	// Initialize dependencies once
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	webhookHandler = initializeHandler(cfg)
}

// AnalyticsWebhook is the Cloud Function entry point
func AnalyticsWebhook(w http.ResponseWriter, r *http.Request) {
	webhookHandler.ServeHTTP(w, r)
}

// initializeHandler initializes all dependencies following SOLID principles
// (Dependency Injection, Interface Segregation, Dependency Inversion)
func initializeHandler(cfg *config.Config) http.Handler {
	ctx := context.Background()

	// Initialize Firebase
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

	// Create concrete implementations
	logger := services.NewSimpleLogger()
	validator := domain.NewHMACValidator(cfg.WebhookSecret)
	writer := repositories.NewFirebaseRepository(dbClient)

	// Compose service (Dependency Injection)
	webhookService := services.NewWebhookService(validator, writer, logger)

	// Create HTTP handler
	handler := handlers.NewWebhookHandler(webhookService, logger)

	logger.Info("webhook handler initialized", "environment", cfg.Environment)

	return handler
}
