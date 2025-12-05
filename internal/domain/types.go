// Package domain contains domain models and interfaces following SOLID principles
package domain

import "context"

// AnalyticsRecord represents a complete analytics record from the chatbot
type AnalyticsRecord struct {
	RequestID     string `json:"requestId"`
	Query         string `json:"query"`
	MatchType     string `json:"matchType"`
	MatchScore    int    `json:"matchScore"`
	Reasoning     string `json:"reasoning"`
	VectorMatches int    `json:"vectorMatches"`
	SessionID     string `json:"sessionId"`
	Week          string `json:"week"`
	Timestamp     int64  `json:"timestamp"`
}

// WebhookPayload represents the incoming webhook payload from AWS Lambda
type WebhookPayload struct {
	EventType string          `json:"eventType"`
	Timestamp int64           `json:"timestamp"`
	Data      AnalyticsRecord `json:"data"`
}

// AnalyticsWriter interface (Dependency Inversion Principle)
// Allows swapping Firebase for other storage implementations
type AnalyticsWriter interface {
	Write(ctx context.Context, record AnalyticsRecord) error
}

// SignatureValidator interface (Dependency Inversion Principle)
// Separates validation logic from transport layer
type SignatureValidator interface {
	Validate(payload []byte, signature string) error
}

// Logger interface (Dependency Inversion Principle)
// Allows swapping logging implementations
type Logger interface {
	Error(msg string, err error)
	Info(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// WebhookProcessor interface (Dependency Inversion Principle)
// Main business logic abstraction
type WebhookProcessor interface {
	Process(ctx context.Context, payload []byte, signature string) error
}
