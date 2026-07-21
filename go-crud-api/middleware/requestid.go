package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// contextKey is an unexported type for context keys in this package.
// Using a custom type prevents key collisions with other packages
// that also store values in context — a subtle but important Go pattern.

type contextKey string

const RequestIDKey contextKey = "request_id"

// RequestID generates a unique ID for every request and attaches it to the context.
// In the proctoring system, this becomes the session trace ID —
// every log line, every gRPC call, every database write will carry this ID
// so you can reconstruct exactly what happened in a session.

func RequestID(next http.Handler) http.Handler{
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()

       // Store in context so any handler downstream can read it
	   ctx := context.WithValue(r.Context(), RequestIDKey, id)

	    // Also send it back in the response header so clients can reference it
		w.Header().Set("X-Request-ID", id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
// GetRequestID is a helper handlers use to extract the ID from context.
// This is the clean API pattern — never access context keys directly outside this package.

func GateRequestID(r *http.Request) string {
	id, _ := r.Context().Value(RequestIDKey).(string)
	return id
}