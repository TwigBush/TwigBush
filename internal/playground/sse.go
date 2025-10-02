package playground

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type GrantEvent struct {
	ID        string    `json:"id"`
	State     string    `json:"state"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan GrantEvent]struct{}
}

func NewSSEHub() *SSEHub { return &SSEHub{clients: map[chan GrantEvent]struct{}{}} }

func (h *SSEHub) Subscribe(ctx context.Context) <-chan GrantEvent {
	ch := make(chan GrantEvent, 128)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	go func() { <-ctx.Done(); h.mu.Lock(); delete(h.clients, ch); close(ch); h.mu.Unlock() }()
	return ch
}

func (h *SSEHub) Broadcast(ev GrantEvent) {
	h.mu.RLock()
	for ch := range h.clients {
		select {
		case ch <- ev:
		default: /* drop */
		}
	}
	h.mu.RUnlock()
}

func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("checking for events")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	enc := json.NewEncoder(w)
	w.Write([]byte("event: ping\ndata: {}\n\n"))
	flusher.Flush()

	events := h.Subscribe(r.Context())
	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-events:
			log.Printf("sending event: %+v", ev)
			w.Write([]byte("event: grant\ndata: "))
			_ = enc.Encode(ev)
			w.Write([]byte("\n"))
			flusher.Flush()
		}
	}
}
