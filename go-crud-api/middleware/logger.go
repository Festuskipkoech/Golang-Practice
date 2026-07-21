package middleware

import (
    "bufio"
    "log/slog"
    "net"
    "net/http"
    "time"
)

type responseWriter struct {
    http.ResponseWriter
    status int
}

func (rw *responseWriter) WriteHeader(status int) {
    rw.status = status
    rw.ResponseWriter.WriteHeader(status)
}

// Hijack delegates to the underlying ResponseWriter's Hijacker implementation.
// WebSocket upgrades require this — the coder/websocket library calls Hijack
// to take over the raw TCP connection. Without forwarding it here,
// our wrapper silently drops the interface and Accept() returns 501.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
    hijacker, ok := rw.ResponseWriter.(http.Hijacker)
    if !ok {
        return nil, nil, http.ErrNotSupported
    }
    return hijacker.Hijack()
}

func Logger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        wrapped := &responseWriter{
            ResponseWriter: w,
            status:         http.StatusOK,
        }

        next.ServeHTTP(wrapped, r)

        slog.Info("request",
            "method", r.Method,
            "path", r.URL.Path,
            "status", wrapped.status,
            "duration", time.Since(start).String(),
            "remote_addr", r.RemoteAddr,
        )
    })
}