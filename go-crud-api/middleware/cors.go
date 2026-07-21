package middleware

import (
	"net/http"
)

// CORS handles cross-origin requests.
// In the proctoring system the Go server will receive requests from
// a web dashboard (different origin) and from the Rust desktop agent —
// both need the right headers or browsers will block the requests.

func CORS(next http.Handler) http.Handler {
	return  http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")

		// OPTIONS is a preflight request — browsers send it before the real request
        // to check if CORS is allowed. We respond immediately with 204 No Content.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return 
		}

		next.ServeHTTP(w, r)
	})
}