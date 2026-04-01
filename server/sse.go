package server

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/allouis/cctl/session"
)

type Hub struct {
	svc         *session.Service
	notify      <-chan struct{}
	subscribers map[chan []byte]struct{}
	mu          sync.RWMutex
	done        chan struct{}
}

func NewHub(svc *session.Service, notify <-chan struct{}) *Hub {
	return &Hub{
		svc:         svc,
		notify:      notify,
		subscribers: make(map[chan []byte]struct{}),
		done:        make(chan struct{}),
	}
}

func (h *Hub) Run() {
	// Fallback ticker catches tmux window exits that don't fire hooks.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastHash [sha256.Size]byte

	check := func() {
		sessions, err := h.svc.List()
		if err != nil {
			return
		}

		data, err := json.Marshal(sessions)
		if err != nil {
			return
		}

		hash := sha256.Sum256(data)
		if hash == lastHash {
			return
		}
		lastHash = hash

		h.broadcast(data)
	}

	for {
		select {
		case <-h.notify:
			check()
		case <-ticker.C:
			check()
		case <-h.done:
			return
		}
	}
}

func (h *Hub) Stop() {
	close(h.done)
}

func (h *Hub) subscribe() chan []byte {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.subscribers, ch)
	h.mu.Unlock()
	close(ch)
}

func (h *Hub) broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.subscribers {
		select {
		case ch <- data:
		default:
			// drop if subscriber is slow
		}
	}
}

func (h *Hub) ServeSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := h.subscribe()
	defer h.unsubscribe(ch)

	// Send initial state (enriched via service)
	sessions, _ := h.svc.List()
	if data, err := json.Marshal(sessions); err == nil {
		fmt.Fprintf(w, "event: sessions\ndata: %s\n\n", data)
		flusher.Flush()
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	timeout := time.NewTimer(5 * time.Minute)
	defer timeout.Stop()

	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: sessions\ndata: %s\n\n", data)
			flusher.Flush()

		case <-heartbeat.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()

		case <-timeout.C:
			return

		case <-r.Context().Done():
			return
		}
	}
}
