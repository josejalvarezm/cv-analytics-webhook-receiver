package handlers

import (
	"io"
	"net/http"

	"github.com/josejalvarezm/cv-analytics-webhook-receiver/internal/domain"
)

// WebhookHandler handles incoming webhook requests (HTTP transport layer)
type WebhookHandler struct {
	processor domain.WebhookProcessor
	logger    domain.Logger
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(processor domain.WebhookProcessor, logger domain.Logger) *WebhookHandler {
	return &WebhookHandler{
		processor: processor,
		logger:    logger,
	}
}

// ServeHTTP handles HTTP requests to the webhook endpoint
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		h.logger.Error("failed to read request body", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// Extract signature from headers
	signature := r.Header.Get("X-Webhook-Signature")
	if signature == "" {
		h.logger.Info("missing webhook signature header")
		http.Error(w, "Missing X-Webhook-Signature header", http.StatusBadRequest)
		return
	}

	// Process webhook
	if err := h.processor.Process(r.Context(), body, signature); err != nil {
		h.logger.Error("failed to process webhook", err)
		http.Error(w, "Failed to process webhook", http.StatusUnauthorized)
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true,"status":"ok"}`))
}
