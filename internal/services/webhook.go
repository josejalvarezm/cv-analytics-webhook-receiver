package services

import (
	"context"
	"encoding/json"
	"fmt"

	"example.com/webhook-receiver/internal/domain"
)

// WebhookService implements domain.WebhookProcessor
// Orchestrates validation and storage (Business Logic Layer)
type WebhookService struct {
	validator domain.SignatureValidator
	writer    domain.AnalyticsWriter
	logger    domain.Logger
}

// NewWebhookService creates a new webhook service with dependency injection
func NewWebhookService(
	validator domain.SignatureValidator,
	writer domain.AnalyticsWriter,
	logger domain.Logger,
) *WebhookService {
	return &WebhookService{
		validator: validator,
		writer:    writer,
		logger:    logger,
	}
}

// Process validates and stores the webhook payload
func (s *WebhookService) Process(ctx context.Context, payload []byte, signature string) error {
	// Step 1: Validate signature
	if err := s.validator.Validate(payload, signature); err != nil {
		s.logger.Error("webhook validation failed", err)
		return fmt.Errorf("webhook validation failed: %w", err)
	}

	// Step 2: Parse payload
	var webhookPayload domain.WebhookPayload
	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
		s.logger.Error("failed to parse webhook payload", err)
		return fmt.Errorf("failed to parse webhook: %w", err)
	}

	// Step 3: Validate parsed data
	if err := validateAnalyticsRecord(&webhookPayload.Data); err != nil {
		s.logger.Error("analytics record validation failed", err)
		return fmt.Errorf("invalid analytics record: %w", err)
	}

	// Step 4: Store in Firebase
	if err := s.writer.Write(ctx, webhookPayload.Data); err != nil {
		s.logger.Error("failed to write analytics", err)
		return fmt.Errorf("failed to store analytics: %w", err)
	}

	s.logger.Info("webhook processed successfully", "requestId", webhookPayload.Data.RequestID)
	return nil
}

// validateAnalyticsRecord ensures required fields are present
func validateAnalyticsRecord(record *domain.AnalyticsRecord) error {
	if record.RequestID == "" {
		return fmt.Errorf("requestId is required")
	}
	if record.Query == "" {
		return fmt.Errorf("query is required")
	}
	if record.Timestamp == 0 {
		return fmt.Errorf("timestamp is required")
	}
	return nil
}
