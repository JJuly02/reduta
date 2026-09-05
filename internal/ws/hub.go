// Package ws is an in-process WebSocket fan-out hub. Cross-instance fan-out is
// done by the server bridging Redis pub/sub into Broadcast (spec 5.7).
package ws

import (
	"context"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type Client struct {
	send chan []byte
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]struct{} // eventID -> clients
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]map[*Client]struct{})}
}

func (h *Hub) add(eventID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[eventID] == nil {
		h.rooms[eventID] = make(map[*Client]struct{})
	}
	h.rooms[eventID][c] = struct{}{}
}

func (h *Hub) remove(eventID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room := h.rooms[eventID]; room != nil {
		delete(room, c)
		if len(room) == 0 {
			delete(h.rooms, eventID)
		}
	}
}

// Broadcast delivers msg to every client subscribed to eventID (non-blocking;
// slow clients drop the message rather than stall the hub).
func (h *Hub) Broadcast(eventID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[eventID] {
		select {
		case c.send <- msg:
		default:
		}
	}
}

// Serve upgrades the connection and pumps broadcast messages until it closes.
func (h *Hub) Serve(ctx context.Context, w http.ResponseWriter, r *http.Request, eventID string) error {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return err
	}
	c := &Client{send: make(chan []byte, 32)}
	h.add(eventID, c)
	defer func() {
		h.remove(eventID, c)
		conn.Close(websocket.StatusNormalClosure, "")
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Reader: we don't expect client messages; use it to detect disconnect.
	go func() {
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				cancel()
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-c.send:
			wctx, wcancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(wctx, websocket.MessageText, msg)
			wcancel()
			if err != nil {
				return err
			}
		}
	}
}
