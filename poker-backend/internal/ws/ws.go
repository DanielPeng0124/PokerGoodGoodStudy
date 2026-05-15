package ws

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"poker-backend/internal/auth"
	"poker-backend/internal/game"
	"poker-backend/internal/room"
)

type Hub struct {
	Manager *room.Manager
	mu      sync.Mutex
	conns   map[string]map[*websocket.Conn]*clientConn
	timers  map[string]bool
}

type clientConn struct {
	uid string
	mu  sync.Mutex
}

func NewHub(m *room.Manager) *Hub {
	return &Hub{Manager: m, conns: map[string]map[*websocket.Conn]*clientConn{}, timers: map[string]bool{}}
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type Message struct {
	Type       string      `json:"type"`
	Seat       int         `json:"seat,omitempty"`
	BuyIn      int64       `json:"buyIn,omitempty"`
	Name       string      `json:"name,omitempty"`
	Away       bool        `json:"away,omitempty"`
	HandNumber int         `json:"handNumber,omitempty"`
	Action     game.Action `json:"action,omitempty"`
	Text       string      `json:"text,omitempty"`
}

func (h *Hub) Serve(roomID string, w http.ResponseWriter, r *http.Request) {
	uid, name := auth.UserFromRequest(r)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	h.ensureRoomTimer(roomID)
	client := &clientConn{uid: uid}
	h.mu.Lock()
	if h.conns[roomID] == nil {
		h.conns[roomID] = map[*websocket.Conn]*clientConn{}
	}
	h.conns[roomID][conn] = client
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.conns[roomID], conn)
		h.mu.Unlock()
	}()
	h.sendState(roomID, conn, client)
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		rm, err := h.Manager.Get(roomID)
		if err != nil {
			writeErr(conn, client, err)
			continue
		}
		switch msg.Type {
		case "sit_down":
			sitName := strings.TrimSpace(msg.Name)
			if sitName == "" {
				sitName = name
			}
			err = rm.Sit(uid, sitName, msg.Seat, msg.BuyIn)
			if err == nil {
				name = sitName
			}
		case "set_away":
			err = rm.SetAway(uid, msg.Away)
		case "leave_seat":
			err = rm.LeaveSeat(uid)
		case "start_game":
			err = rm.Start(uid)
		case "pause_game":
			err = rm.Pause(uid, true)
		case "resume_game":
			err = rm.Pause(uid, false)
		case "end_game":
			err = rm.End(uid, msg.HandNumber)
		case "action":
			err = rm.Action(uid, msg.Action)
		case "skip_turn":
			err = rm.SkipCurrentTurn()
		case "add_time":
			err = rm.AddTurnTime(uid)
		case "chat":
			h.broadcast(roomID, map[string]any{"type": "chat", "userId": uid, "name": name, "text": msg.Text})
			continue
		default:
			err = nil
		}
		if err != nil {
			writeErr(conn, client, err)
			continue
		}
		h.broadcastState(roomID, rm)
	}
}

func (h *Hub) broadcastState(roomID string, rm *room.Room) {
	for _, recipient := range h.recipients(roomID) {
		writeJSON(recipient.conn, recipient.client, map[string]any{"type": "room_state", "payload": rm.Snapshot(recipient.client.uid)})
	}
}
func (h *Hub) sendState(roomID string, c *websocket.Conn, client *clientConn) {
	if rm, err := h.Manager.Get(roomID); err == nil {
		writeJSON(c, client, map[string]any{"type": "room_state", "payload": rm.Snapshot(client.uid)})
	}
}
func (h *Hub) broadcast(roomID string, v any) {
	b, _ := json.Marshal(v)
	for _, recipient := range h.recipients(roomID) {
		recipient.client.mu.Lock()
		_ = recipient.conn.WriteMessage(websocket.TextMessage, b)
		recipient.client.mu.Unlock()
	}
}
func (h *Hub) ensureRoomTimer(roomID string) {
	h.mu.Lock()
	if h.timers[roomID] {
		h.mu.Unlock()
		return
	}
	h.timers[roomID] = true
	h.mu.Unlock()

	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for now := range ticker.C {
			rm, err := h.Manager.Get(roomID)
			if err != nil {
				return
			}
			if rm.AutoActExpired(now) {
				h.broadcastState(roomID, rm)
			}
		}
	}()
}

type recipient struct {
	conn   *websocket.Conn
	client *clientConn
}

func (h *Hub) recipients(roomID string) []recipient {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]recipient, 0, len(h.conns[roomID]))
	for conn, client := range h.conns[roomID] {
		out = append(out, recipient{conn: conn, client: client})
	}
	return out
}

func writeJSON(c *websocket.Conn, client *clientConn, v any) {
	client.mu.Lock()
	_ = c.WriteJSON(v)
	client.mu.Unlock()
}

func writeErr(c *websocket.Conn, client *clientConn, err error) {
	writeJSON(c, client, map[string]any{"type": "error", "error": err.Error()})
}
