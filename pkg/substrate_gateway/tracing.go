package substrategw

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// TraceIDKey is the context key for the trace ID
	TraceIDKey ContextKey = "trace_id"
)

// GetTraceID retrieves the trace ID from the context, or generates a new one if not present
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return generateTraceID()
	}

	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		return traceID
	}

	return generateTraceID()
}

// WithTraceID adds a trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// generateTraceID generates a new unique trace ID
func generateTraceID() string {
	return fmt.Sprintf("trace-%s", uuid.New().String())
}

// LogWithTrace creates a log event with the trace ID attached
func LogWithTrace(ctx context.Context) *zerolog.Event {
	traceID := GetTraceID(ctx)
	return log.Info().Str("trace_id", traceID)
}

// LogDebugWithTrace creates a debug log event with the trace ID attached
func LogDebugWithTrace(ctx context.Context) *zerolog.Event {
	traceID := GetTraceID(ctx)
	return log.Debug().Str("trace_id", traceID)
}

// LogErrorWithTrace creates an error log event with the trace ID attached
func LogErrorWithTrace(ctx context.Context) *zerolog.Event {
	traceID := GetTraceID(ctx)
	return log.Error().Str("trace_id", traceID)
}
