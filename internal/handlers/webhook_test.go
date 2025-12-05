package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/webhook-receiver/internal/domain"
)

// MockWebhookProcessor for testing
type MockWebhookProcessor struct {
	ProcessCalled bool
	ProcessError  error
}

func (m *MockWebhookProcessor) Process(ctx context.Context, payload []byte, signature string) error {
	m.ProcessCalled = true
	if m.ProcessError != nil {
		return m.ProcessError
	}
	return nil
}

// MockHandlerLogger for testing
type MockHandlerLogger struct {
	ErrorLogs []string
	InfoLogs  []string
	DebugLogs []string
}

func (m *MockHandlerLogger) Error(msg string, err error) {
	m.ErrorLogs = append(m.ErrorLogs, msg)
}

func (m *MockHandlerLogger) Info(msg string, args ...interface{}) {
	m.InfoLogs = append(m.InfoLogs, msg)
}

func (m *MockHandlerLogger) Debug(msg string, args ...interface{}) {
	m.DebugLogs = append(m.DebugLogs, msg)
}

func TestWebhookHandlerServeHTTPSuccess(t *testing.T) {
	// Arrange
	processor := &MockWebhookProcessor{}
	logger := &MockHandlerLogger{}
	handler := NewWebhookHandler(processor, logger)

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

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(payloadJSON))
	req.Header.Set("X-Webhook-Signature", "test_signature")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if !processor.ProcessCalled {
		t.Errorf("Expected Process to be called")
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if success, ok := response["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true in response")
	}
}

func TestWebhookHandlerServeHTTPMissingSignature(t *testing.T) {
	// Arrange
	processor := &MockWebhookProcessor{}
	logger := &MockHandlerLogger{}
	handler := NewWebhookHandler(processor, logger)

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

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(payloadJSON))
	// No signature header
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
	if processor.ProcessCalled {
		t.Errorf("Processor should not be called on missing signature")
	}
}

func TestWebhookHandlerServeHTTPWrongMethod(t *testing.T) {
	// Arrange
	processor := &MockWebhookProcessor{}
	logger := &MockHandlerLogger{}
	handler := NewWebhookHandler(processor, logger)

	req := httptest.NewRequest("GET", "/webhook", nil)
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
	if processor.ProcessCalled {
		t.Errorf("Processor should not be called for GET request")
	}
}

func TestWebhookHandlerServeHTTPProcessorError(t *testing.T) {
	// Arrange
	processor := &MockWebhookProcessor{
		ProcessError: domain.ErrInvalidSignature,
	}
	logger := &MockHandlerLogger{}
	handler := NewWebhookHandler(processor, logger)

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

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(payloadJSON))
	req.Header.Set("X-Webhook-Signature", "invalid_signature")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
	if len(logger.ErrorLogs) == 0 {
		t.Errorf("Expected error to be logged")
	}
}

func TestWebhookHandlerServeHTTPInvalidJSON(t *testing.T) {
	// Arrange
	processor := &MockWebhookProcessor{
		ProcessError: domain.ErrInvalidPayload,
	}
	logger := &MockHandlerLogger{}
	handler := NewWebhookHandler(processor, logger)

	invalidJSON := []byte("{invalid json")

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(invalidJSON))
	req.Header.Set("X-Webhook-Signature", "test_signature")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}
