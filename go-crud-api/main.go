package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-crud-api/handlers"
	"go-crud-api/middleware"
	"go-crud-api/store"
)

// chain applies middleware in order, outermost first.
// The request flows: CORS -> RequestID -> Logger -> your handler
// The response flows back in reverse: handler -> Logger -> RequestID -> CORS
// This is why order matters — Logger must be inside RequestID
// so it can read the request ID from context.

func chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
   // Apply in reverse so the first middleware in the list is outermost
   for i := len(middlewares) - 1; i >= 0; i -- {
	h = middlewares[i](h)
   }
   return h

}
func main() {
	// Build the dependency chain bottom-up:
    // store -> handler -> router -> server
	bookStore := store.NewBookStore()
	sessionStore := store.NewSessionStore()

	// handlers
	bookHander := handlers.NewBookHandler(bookStore)
	sessionHandler := handlers.NewSessionHandler(sessionStore)


	mux := http.NewServeMux()

	// method path routing
	mux.HandleFunc("GET /books", bookHander.GetAll)
	mux.HandleFunc("GET /books/{id}",bookHander.GetById)
	mux.HandleFunc("POST /books", bookHander.Create)
	mux.HandleFunc("PUT /books/{id}", bookHander.Update)
	mux.HandleFunc("DELETE /books/{id}", bookHander.Delete)

    // Session REST routes
    mux.HandleFunc("POST /sessions", sessionHandler.CreateSession)
    mux.HandleFunc("GET /sessions", sessionHandler.GetAllSessions)
    mux.HandleFunc("GET /sessions/{id}", sessionHandler.GetSession)

    // WebSocket route — this is where the Rust agent connects
    mux.HandleFunc("GET /sessions/{id}/stream", sessionHandler.HandleWebSocket)
    // Wrap the entire mux with our middleware stack.
    // Every single request — regardless of route — passes through all three.

	handler := chain(mux,
		middleware.CORS,
		middleware.RequestID,
		middleware.Logger,
	)
	// http.Server gives us control over the server lifecycle.
	// Previously we called http.ListenAndServe which is a convenience wrapper
	// that gives you no handle to shut the server down.
	// Now we own the server struct and can call Shutdown() on it.
	server := &http.Server{
		Addr: ":8080",
		Handler: handler,
		
		// These timeouts protect against slow clients holding connections open.
		// In the proctoring system a student with a bad connection should not
		// be able to block a server goroutine indefinitely.
		ReadTimeout: 30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout: 120 * time.Second,

		// Launch the server in a goroutine so main() can continue
		// to the signal listening block below.
		// If we called server.ListenAndServe() directly it would block forever
		// and we would never reach the shutdown logic.
	}
	go func() {
		fmt.Printf("server running on http://localhost%s\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// ErrServerClosed is expected when Shutdown() is called — not an error.
			// Anything else is a real startup failure.
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()
	// Block here waiting for a shutdown signal.
	// signal.NotifyContext returns a context that is cancelled
	// when the OS sends SIGINT (Ctrl+C) or SIGTERM (what Kubernetes sends
	// when it wants to stop your pod during a rolling deployment).
	// This is the production-correct way to handle both cases identically.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// This blocks until Ctrl+C or SIGTERM arrives.
	<-ctx.Done()
	slog.Info("shutdown signal received — draining connections")

	// Give in-flight requests up to 30 seconds to complete.
	// Shutdown() stops accepting new connections immediately,
	// then waits for active handlers to return.
	// After 30 seconds it gives up and forces exit.
	// For the proctoring system this 30s window is where:
	// - Active WebSocket sessions finish their current frame
	// - Workers drain the jobs channel
	// - Session statuses are written to the database
	shutdownctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Drain the worker pool — wait for all buffered frames to be processed
	// before we let the HTTP server close.
	// This calls the method we are about to add to SessionHandler.
	sessionHandler.Shutdown(shutdownctx)

	if err := server.Shutdown(shutdownctx); err != nil {
		slog.Error("server forced to shut down", "error", err)
	}

	slog.Info("server stopped cleanly")

	
}