package services

import (
	"context"
	"encoding/json"
	"testing"

	"example.com/webhook-receiver/internal/domain"
)

// MockSignatureValidator for testing
type MockSignatureValidator struct {
	ShouldValidate bool
	Error          error
}

func (m *MockSignatureValidator) Validate(payload []byte, signature string) error {
	if m.Error != nil {
		return m.Error
	}
	return nil
}

// MockAnalyticsWriter for testing
type MockAnalyticsWriter struct {
	WrittenRecords []domain.AnalyticsRecord
	Error          error
}

func (m *MockAnalyticsWriter) Write(ctx context.Context, record domain.AnalyticsRecord) error {
	if m.Error != nil {
		return m.Error
	}
	m.WrittenRecords = append(m.WrittenRecords, record)
	return nil
}

// MockLogger for testing
type MockLogger struct {
	ErrorLogs []string
	InfoLogs  []string
	DebugLogs []string
}

func (m *MockLogger) Error(msg string, err error) {
	m.ErrorLogs = append(m.ErrorLogs, msg)
}

func (m *MockLogger) Info(msg string, args ...interface{}) {
	m.InfoLogs = append(m.InfoLogs, msg)
}

func (m *MockLogger) Debug(msg string, args ...interface{}) {
	m.DebugLogs = append(m.DebugLogs, msg)
}

func TestWebhookServiceProcessSuccess(t *testing.T) {
	// Arrange
	validator := &MockSignatureValidator{ShouldValidate: true}
	writer := &MockAnalyticsWriter{}
	logger := &MockLogger{}
	service := NewWebhookService(validator, writer, logger)

	payload := domain.WebhookPayload{
		EventType: "analytics_event",
		Timestamp: 1700000000,
		Data: domain.AnalyticsRecord{
			RequestID: "req_123",
			Query:     "test query",
			SessionID: "sess_789",
			Timestamp: 1700000000,
		},
	}
	payloadJSON, _ := json.Marshal(payload)

	// Act
	err := service.Process(context.Background(), payloadJSON, "valid_signature")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(writer.WrittenRecords) != 1 {
		t.Errorf("Expected 1 written record, got %d", len(writer.WrittenRecords))
	}
	if writer.WrittenRecords[0].RequestID != "req_123" {
		t.Errorf("Expected RequestID req_123, got %s", writer.WrittenRecords[0].RequestID)
	}
	if len(logger.InfoLogs) == 0 {
		t.Errorf("Expected info logs, got none")
	}
}

func TestWebhookServiceProcessInvalidJSON(t *testing.T) {
	// Arrange
	validator := &MockSignatureValidator{ShouldValidate: true}
	writer := &MockAnalyticsWriter{}
	logger := &MockLogger{}
	service := NewWebhookService(validator, writer, logger)

	invalidJSON := []byte("{invalid json")

	// Act
	err := service.Process(context.Background(), invalidJSON, "valid_signature")

	// Assert
	if err == nil {
		t.Errorf("Expected JSON parsing error, got nil")
	}
	if len(writer.WrittenRecords) != 0 {
		t.Errorf("Expected 0 written records, got %d", len(writer.WrittenRecords))
	}
}

func TestWebhookServiceProcessMissingRequired(t *testing.T) {
	// Arrange
	validator := &MockSignatureValidator{ShouldValidate: true}
	writer := &MockAnalyticsWriter{}
	logger := &MockLogger{}
	service := NewWebhookService(validator, writer, logger)

	// Payload missing RequestID (required)
	payload := domain.WebhookPayload{
		EventType: "analytics_event",
		Timestamp: 1700000000,
		Data: domain.AnalyticsRecord{
			Query:     "test query",
			SessionID: "sess_789",
			Timestamp: 1700000000,
		},
	}
	payloadJSON, _ := json.Marshal(payload)

	// Act
	err := service.Process(context.Background(), payloadJSON, "valid_signature")

	// Assert
	if err == nil {
		t.Errorf("Expected validation error, got nil")
	}
	if len(writer.WrittenRecords) != 0 {
		t.Errorf("Expected 0 written records, got %d", len(writer.WrittenRecords))
	}
}

func TestWebhookServiceProcessSignatureValidationFailure(t *testing.T) {
	// Arrange
	validator := &MockSignatureValidator{Error: domain.ErrInvalidSignature}
	writer := &MockAnalyticsWriter{}
	logger := &MockLogger{}
	service := NewWebhookService(validator, writer, logger)

	payload := domain.WebhookPayload{
		EventType: "analytics_event",
		Timestamp: 1700000000,
		Data: domain.AnalyticsRecord{
			RequestID: "req_123",
			Query:     "test query",
			SessionID: "sess_789",
			Timestamp: 1700000000,
		},
	}
	payloadJSON, _ := json.Marshal(payload)

	// Act
	err := service.Process(context.Background(), payloadJSON, "invalid_signature")

	// Assert
	if err == nil {
		t.Errorf("Expected signature validation error, got nil")
	}
	if len(writer.WrittenRecords) != 0 {
		t.Errorf("Expected 0 written records, got %d", len(writer.WrittenRecords))
	}
}

func TestWebhookServiceProcessWriterFailure(t *testing.T) {
	// Arrange
	validator := &MockSignatureValidator{ShouldValidate: true}
	writer := &MockAnalyticsWriter{Error: domain.ErrDatabaseWrite}
	logger := &MockLogger{}
	service := NewWebhookService(validator, writer, logger)

	payload := domain.WebhookPayload{
		EventType: "analytics_event",
		Timestamp: 1700000000,
		Data: domain.AnalyticsRecord{
			RequestID: "req_123",
			Query:     "test query",
			SessionID: "sess_789",
			Timestamp: 1700000000,
		},
	}
	payloadJSON, _ := json.Marshal(payload)

	// Act
	err := service.Process(context.Background(), payloadJSON, "valid_signature")

	// Assert
	if err == nil {
		t.Errorf("Expected database write error, got nil")
	}
	if len(logger.ErrorLogs) == 0 {
		t.Errorf("Expected error logs, got none")
	}
}
