package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Span represents a trace span for observability
type Span struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Name       string
	Attributes map[string]interface{}
	StartTime  time.Time
}

// Tracer provides distributed tracing capabilities
type Tracer struct {
	serviceName string
}

// NewTracer creates a new tracer
func NewTracer(serviceName string) *Tracer {
	return &Tracer{serviceName: serviceName}
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	span := &Span{
		TraceID:    generateID(16),
		SpanID:     generateID(8),
		Name:       name,
		Attributes: make(map[string]interface{}),
		StartTime:  time.Now(), // Add StartTime to Span struct
	}

	// Check for parent span in context
	if parentSpan := SpanFromContext(ctx); parentSpan != nil {
		span.TraceID = parentSpan.TraceID
		span.ParentID = parentSpan.SpanID
	}

	return ContextWithSpan(ctx, span), span
}

// End ends a span and records it
func (t *Tracer) End(span *Span) {
	duration := time.Since(span.StartTime)

	// Structured Log for Tracing (Stdout - can be scraped by Fluentd/Promtail)
	entry := map[string]interface{}{
		"level":       "INFO",
		"ts":          time.Now().Format(time.RFC3339),
		"logger":      "tracer",
		"msg":         "Span Finished",
		"trace_id":    span.TraceID,
		"span_id":     span.SpanID,
		"parent_id":   span.ParentID,
		"span_name":   span.Name,
		"duration_ms": duration.Milliseconds(),
		"attributes":  span.Attributes,
	}

	// Direct JSON write to avoid cyclic dependency with observability package
	// Or just use fmt/log
	jsonBytes, _ := json.Marshal(entry)
	fmt.Println(string(jsonBytes))
}

// SetAttribute sets an attribute on the span
func (s *Span) SetAttribute(key string, value interface{}) {
	s.Attributes[key] = value
}

// Context key type
type spanContextKey struct{}

// ContextWithSpan returns a new context with the span
func ContextWithSpan(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, spanContextKey{}, span)
}

// SpanFromContext retrieves the span from context
func SpanFromContext(ctx context.Context) *Span {
	if span, ok := ctx.Value(spanContextKey{}).(*Span); ok {
		return span
	}
	return nil
}

// TraceIDFromContext gets the trace ID from context
func TraceIDFromContext(ctx context.Context) string {
	if span := SpanFromContext(ctx); span != nil {
		return span.TraceID
	}
	return ""
}

func generateID(bytes int) string {
	b := make([]byte, bytes)
	rand.Read(b)
	return hex.EncodeToString(b)
}
