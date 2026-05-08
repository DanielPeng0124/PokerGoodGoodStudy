package ws

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"poker-backend/internal/auth"
	"poker-backend/internal/game"
	"poker-backend/internal/room"
)

type Hub struct {
	Manager *room.Manager
	conns   map[string]map[*websocket.Conn]string
}

func NewHub(m *room.Manager) *Hub {
	return &Hub{Manager: m, conns: map[string]map[*websocket.Conn]string{}}
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type Message struct {
	Type   string      `json:"type"`
	Seat   int         `json:"seat,omitempty"`
	BuyIn  int64       `json:"buyIn,omitempty"`
	Name   string      `json:"name,omitempty"`
	Action game.Action `json:"action,omitempty"`
	Text   string      `json:"text,omitempty"`
}

func (h *Hub) Serve(roomID string, w http.ResponseWriter, r *http.Request) {
	uid, name := auth.UserFromRequest(r)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	if h.conns[roomID] == nil {
		h.conns[roomID] = map[*websocket.Conn]string{}
	}
	h.conns[roomID][conn] = uid
	defer delete(h.conns[roomID], conn)
	h.sendState(roomID, conn, uid)
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		rm, err := h.Manager.Get(roomID)
		if err != nil {
			writeErr(conn, err)
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
		case "start_game":
			err = rm.Start()
		case "action":
			err = rm.Action(uid, msg.Action)
		case "skip_turn":
			err = rm.SkipCurrentTurn()
		case "chat":
			h.broadcast(roomID, map[string]any{"type": "chat", "userId": uid, "name": name, "text": msg.Text})
			continue
		default:
			err = nil
		}
		if err != nil {
			writeErr(conn, err)
			continue
		}
		h.broadcastState(roomID, rm)
	}
}

func (h *Hub) broadcastState(roomID string, rm *room.Room) {
	for c, uid := range h.conns[roomID] {
		_ = c.WriteJSON(map[string]any{"type": "room_state", "payload": rm.Snapshot(uid)})
	}
}
func (h *Hub) sendState(roomID string, c *websocket.Conn, uid string) {
	if rm, err := h.Manager.Get(roomID); err == nil {
		_ = c.WriteJSON(map[string]any{"type": "room_state", "payload": rm.Snapshot(uid)})
	}
}
func (h *Hub) broadcast(roomID string, v any) {
	b, _ := json.Marshal(v)
	for c := range h.conns[roomID] {
		_ = c.WriteMessage(websocket.TextMessage, b)
	}
}
func writeErr(c *websocket.Conn, err error) {
	_ = c.WriteJSON(map[string]any{"type": "error", "error": err.Error()})
}
