package handlers

import "net/http"

// HandleSSE handles the SSE endpoint.
func HandleSSE(w http.ResponseWriter, r *http.Request) {
	defaultEventHub.HandleSSE(w, r)
}
