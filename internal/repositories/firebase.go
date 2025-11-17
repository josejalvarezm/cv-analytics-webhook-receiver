package repositories

import (
	"context"
	"fmt"
	"time"

	"firebase.google.com/go/v4/db"
	"github.com/josejalvarezm/cv-analytics-webhook-receiver/internal/domain"
)

// FirebaseRepository implements domain.AnalyticsWriter using Firebase Realtime Database
type FirebaseRepository struct {
	client *db.Client
}

// NewFirebaseRepository creates a new Firebase repository
func NewFirebaseRepository(client *db.Client) *FirebaseRepository {
	return &FirebaseRepository{
		client: client,
	}
}

// Write stores an analytics record in Firebase
func (r *FirebaseRepository) Write(ctx context.Context, record domain.AnalyticsRecord) error {
	ref := r.client.NewRef("analytics/live")

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
		"receivedAt":    time.Now().UnixMilli(),
	}

	// Push creates a new child with auto-generated key
	if _, err := ref.Push(ctx, data); err != nil {
		return fmt.Errorf("failed to write analytics: %w", err)
	}

	return nil
}
