// Package function contains the Cloud Function entry point for GCP Cloud Functions Gen2
package function

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"golang.org/x/time/rate"
)

var webhookHandler http.Handler

// ===== CONFIG LAYER =====

type Config struct {
	WebhookSecret string
	Environment   string
}

func loadConfig() (*Config, error) {
	cfg := &Config{
		WebhookSecret: os.Getenv("WEBHOOK_SECRET"),
		Environment:   getEnvOrDefault("ENVIRONMENT", "production"),
	}

	if cfg.WebhookSecret == "" {
		return nil, fmt.Errorf("WEBHOOK_SECRET environment variable is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ===== DOMAIN LAYER =====

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

type WebhookPayload struct {
	EventType string          `json:"eventType"`
	Timestamp int64           `json:"timestamp"`
	Data      AnalyticsRecord `json:"data"`
}

type Logger interface {
	Error(msg string, err error)
	Info(msg string, args ...interface{})
}

type SignatureValidator interface {
	Validate(payload []byte, signature string) error
}

type AnalyticsWriter interface {
	Write(ctx context.Context, record AnalyticsRecord) error
}

// ===== SERVICE LAYER =====

type SimpleLogger struct{}

func (l *SimpleLogger) Error(msg string, err error) {
	log.Printf("[ERROR] %s: %v", msg, err)
}

func (l *SimpleLogger) Info(msg string, args ...interface{}) {
	log.Printf("[INFO] %s %v", msg, fmt.Sprint(args...))
}

type HMACValidator struct {
	secret string
}

func NewHMACValidator(secret string) *HMACValidator {
	return &HMACValidator{secret: secret}
}

func (v *HMACValidator) Validate(payload []byte, signature string) error {
	mac := hmac.New(sha256.New, []byte(v.secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

type WebhookService struct {
	validator SignatureValidator
	writer    AnalyticsWriter
	logger    Logger
}

func NewWebhookService(validator SignatureValidator, writer AnalyticsWriter, logger Logger) *WebhookService {
	return &WebhookService{validator, writer, logger}
}

func (s *WebhookService) Process(ctx context.Context, payload []byte, signature string) error {
	// Validate signature
	if err := s.validator.Validate(payload, signature); err != nil {
		s.logger.Error("webhook validation failed", err)
		return fmt.Errorf("webhook validation failed: %w", err)
	}

	// Parse payload
	var webhookPayload WebhookPayload
	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
		s.logger.Error("failed to parse webhook payload", err)
		return fmt.Errorf("failed to parse webhook: %w", err)
	}

	// Validate record
	if err := validateAnalyticsRecord(&webhookPayload.Data); err != nil {
		s.logger.Error("analytics record validation failed", err)
		return fmt.Errorf("invalid analytics record: %w", err)
	}

	// Store in Firestore
	if err := s.writer.Write(ctx, webhookPayload.Data); err != nil {
		s.logger.Error("failed to write analytics", err)
		return fmt.Errorf("failed to store analytics: %w", err)
	}

	s.logger.Info("webhook processed successfully", "requestId", webhookPayload.Data.RequestID)
	return nil
}

func validateAnalyticsRecord(record *AnalyticsRecord) error {
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

// ===== REPOSITORY LAYER =====

type FirestoreRepository struct {
	client *firestore.Client
}

func NewFirestoreRepository(client *firestore.Client) *FirestoreRepository {
	return &FirestoreRepository{client: client}
}

func (r *FirestoreRepository) Write(ctx context.Context, record AnalyticsRecord) error {
	docRef := r.client.Collection("analytics").Doc(record.RequestID)

	data := map[string]interface{}{
		"requestId":     record.RequestID,
		"query":         record.Query,
		"matchType":     record.MatchType,
		"matchScore":    record.MatchScore,
		"reasoning":     record.Reasoning,
		"vectorMatches": record.VectorMatches,
		"sessionId":     record.SessionID,
		"week":          record.Week,
		"timestamp":     record.Timestamp,
		"receivedAt":    time.Now().Unix(),
	}

	if _, err := docRef.Set(ctx, data); err != nil {
		return fmt.Errorf("failed to write analytics to Firestore: %w", err)
	}

	return nil
}

// ===== HANDLER LAYER =====

// RateLimiter provides thread-safe rate limiting
type RateLimiter struct {
	limiter *rate.Limiter
	mu      sync.Mutex
}

func NewRateLimiter(requestsPerSecond int, burst int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.limiter.Allow()
}

type WebhookHandler struct {
	processor   *WebhookService
	logger      Logger
	rateLimiter *RateLimiter
}

func NewWebhookHandler(processor *WebhookService, logger Logger, rateLimiter *RateLimiter) *WebhookHandler {
	return &WebhookHandler{processor, logger, rateLimiter}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check rate limit first (before any processing)
	if !h.rateLimiter.Allow() {
		h.logger.Info("rate limit exceeded", r.RemoteAddr)
		w.Header().Set("X-RateLimit-Retry-After", "1")
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		h.logger.Error("failed to read request body", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("X-Webhook-Signature")
	if signature == "" {
		h.logger.Info("missing webhook signature header", nil)
		http.Error(w, "Missing X-Webhook-Signature header", http.StatusBadRequest)
		return
	}

	// Remove "sha256=" prefix if present (Lambda sends "sha256=<hex>")
	signature = strings.TrimPrefix(signature, "sha256=")

	if err := h.processor.Process(r.Context(), body, signature); err != nil {
		h.logger.Error("failed to process webhook", err)
		http.Error(w, "Failed to process webhook", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true,"status":"ok"}`))
}

// ===== CLOUD FUNCTION ENTRY POINT =====

func init() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	webhookHandler = initializeHandler(cfg)
}

// AnalyticsWebhook is the HTTP Cloud Function entry point
func AnalyticsWebhook(w http.ResponseWriter, r *http.Request) {
	if webhookHandler == nil {
		http.Error(w, "Handler not initialized", http.StatusInternalServerError)
		return
	}
	webhookHandler.ServeHTTP(w, r)
}

func initializeHandler(cfg *Config) http.Handler {
	ctx := context.Background()

	firebaseApp, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to initialize Firebase: %v", err)
	}

	firestoreClient, err := firebaseApp.Firestore(ctx)
	if err != nil {
		log.Fatalf("Failed to get Firestore client: %v", err)
	}

	logger := &SimpleLogger{}
	validator := NewHMACValidator(cfg.WebhookSecret)
	writer := NewFirestoreRepository(firestoreClient)
	webhookService := NewWebhookService(validator, writer, logger)

	// Rate limiter: 100 requests per second with burst of 20
	// Protects against DDoS while allowing legitimate traffic spikes
	rateLimiter := NewRateLimiter(100, 20)

	handler := NewWebhookHandler(webhookService, logger, rateLimiter)

	logger.Info("webhook handler initialized", "environment", cfg.Environment, "database", "firestore", "rate_limit", "100 req/s")

	return handler
}
