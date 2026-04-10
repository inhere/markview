package handlers

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/inhere/markview/internal/utils"
)

var (
	clients   = make(map[chan string]bool)
	clientsMu sync.Mutex
)

// HandleSSE handles the SSE endpoint.
func HandleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	utils.Debugf("Request: %s handle SSE, clientIp: %s, clientNum: %d", r.URL.Path, r.RemoteAddr, len(clients))
	clientChan := make(chan string)

	clientsMu.Lock()
	clients[clientChan] = true
	clientsMu.Unlock()

	// 每连接独立创建 ticker，确保不被其他连接共享
	// 间隔 9s < WriteTimeout 10s，保持连接活跃
	ticker := time.NewTicker(9 * time.Second)
	defer ticker.Stop()

	defer func() {
		clientsMu.Lock()
		delete(clients, clientChan)
		clientsMu.Unlock()
		close(clientChan)
	}()

	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			return
		case msg := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-ticker.C:
			// Keep alive: 发送空注释行维持连接
			fmt.Fprintf(w, ": keepalive\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}
