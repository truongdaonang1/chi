package middleware

import (
	"context"
)

// RequestIDHeader is the HTTP header for request tracking.
var RequestIDHeader = "X-Request-Id"

type reqIDKey struct{}

// GetReqID returns request ID from context.
func GetReqID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if reqID, ok := ctx.Value(reqIDKey{}).(string); ok {
		return reqID
	}
	return ""
}

// WithReqID sets request ID on context.
func WithReqID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, reqIDKey{}, reqID)
}
