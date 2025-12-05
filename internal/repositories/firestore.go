package repositories

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"example.com/webhook-receiver/internal/domain"
)

// FirestoreRepository implements domain.AnalyticsWriter using Firestore
// Uses requestId as document ID for guaranteed idempotency
type FirestoreRepository struct {
	client *firestore.Client
}

// NewFirestoreRepository creates a new Firestore repository
func NewFirestoreRepository(client *firestore.Client) *FirestoreRepository {
	return &FirestoreRepository{
		client: client,
	}
}

// Write stores an analytics record in Firestore
// Uses requestId as document ID to prevent duplicates (idempotent)
func (r *FirestoreRepository) Write(ctx context.Context, record domain.AnalyticsRecord) error {
	// Use requestId as document ID for idempotency
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

	// Set overwrites if document exists (idempotent operation)
	if _, err := docRef.Set(ctx, data); err != nil {
		return fmt.Errorf("failed to write analytics to Firestore: %w", err)
	}

	return nil
}
