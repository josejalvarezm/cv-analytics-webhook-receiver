package services

import (
	"fmt"
	"log"
)

// SimpleLogger implements domain.Logger interface
type SimpleLogger struct{}

// NewSimpleLogger creates a new simple logger
func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{}
}

// Error logs an error message
func (l *SimpleLogger) Error(msg string, err error) {
	log.Printf("[ERROR] %s: %v\n", msg, err)
}

// Info logs an info message
func (l *SimpleLogger) Info(msg string, args ...interface{}) {
	log.Printf("[INFO] %s %v\n", msg, fmt.Sprint(args...))
}

// Debug logs a debug message
func (l *SimpleLogger) Debug(msg string, args ...interface{}) {
	log.Printf("[DEBUG] %s %v\n", msg, fmt.Sprint(args...))
}
