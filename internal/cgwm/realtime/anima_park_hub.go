package realtime

import (
	"log"
	"sync"
	"github.com/gorilla/websocket"
)

// Simple in-memory hub for park real-time (presences, meetings).
// In production use Redis + proper auth. Broadcast only public/anonymized data.

type ParkHub struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}

var ParkHubInstance = &ParkHub{clients: make(map[*websocket.Conn]bool)}

func (h *ParkHub) Register(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()
}

func (h *ParkHub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
}

func (h *ParkHub) Broadcast(msg interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("park hub broadcast err: %v", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

// Similar SyncHub for cloud sync status broadcasts to admin.
type SyncHub struct { /* identical pattern */ }