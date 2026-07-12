package application

import "context"

type contextKey string

const correlationIDKey contextKey = "correlation_id"

func WithCorrelationID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, correlationIDKey, value)
}

func CorrelationID(ctx context.Context) string {
	value, _ := ctx.Value(correlationIDKey).(string)
	return value
}
