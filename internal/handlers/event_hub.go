package handlers

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type EventHub struct {
	mu      sync.Mutex
	clients map[chan string]struct{}
	closed  bool
}

var defaultEventHub = NewEventHub()

func NewEventHub() *EventHub {
	return &EventHub{clients: make(map[chan string]struct{})}
}

func (hub *EventHub) Subscribe() (<-chan string, func()) {
	client := make(chan string, 8)
	hub.mu.Lock()
	if hub.closed {
		close(client)
	} else {
		hub.clients[client] = struct{}{}
	}
	hub.mu.Unlock()
	return client, sync.OnceFunc(func() { hub.unsubscribe(client) })
}

func (hub *EventHub) Publish(message string) bool {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if hub.closed {
		return false
	}
	for client := range hub.clients {
		select {
		case client <- message:
		default:
		}
	}
	return true
}

func (hub *EventHub) Close() error {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if hub.closed {
		return nil
	}
	hub.closed = true
	for client := range hub.clients {
		delete(hub.clients, client)
		close(client)
	}
	return nil
}

func (hub *EventHub) unsubscribe(client chan string) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if _, ok := hub.clients[client]; !ok {
		return
	}
	delete(hub.clients, client)
	close(client)
}

func (hub *EventHub) HandleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	client, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	fmt.Fprint(w, "data: connected\n\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	ticker := time.NewTicker(9 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case message, ok := <-client:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", message)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-ticker.C:
			fmt.Fprint(w, ": keepalive\n\n")
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}
