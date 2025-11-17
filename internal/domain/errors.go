package domain

import "errors"

// Domain errors
var (
	// ErrInvalidSignature returned when HMAC signature validation fails
	ErrInvalidSignature = errors.New("invalid webhook signature")

	// ErrDatabaseWrite returned when Firebase write fails
	ErrDatabaseWrite = errors.New("failed to write to database")

	// ErrInvalidPayload returned when webhook payload validation fails
	ErrInvalidPayload = errors.New("invalid webhook payload")

	// ErrMissingField returned when required field is missing
	ErrMissingField = errors.New("missing required field")
)
