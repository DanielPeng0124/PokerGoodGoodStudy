package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"poker-backend/internal/auth"
	"poker-backend/internal/model"
	"poker-backend/internal/room"
	"poker-backend/internal/ws"
)

func NewRouter(m *room.Manager) http.Handler {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET","POST","OPTIONS"}, AllowedHeaders: []string{"*"}}))
	hub := ws.NewHub(m)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request){ writeJSON(w, map[string]string{"status":"ok"}) })
	r.Post("/rooms", func(w http.ResponseWriter, r *http.Request){
		uid,_ := auth.UserFromRequest(r)
		var req struct{ Settings model.RoomSettings `json:"settings"` }
		_ = json.NewDecoder(r.Body).Decode(&req)
		rm := m.Create(uid, req.Settings)
		writeJSON(w, rm.Snapshot(uid))
	})
	r.Get("/rooms/{roomID}", func(w http.ResponseWriter, r *http.Request){
		uid,_ := auth.UserFromRequest(r)
		rm, err := m.Get(chi.URLParam(r,"roomID")); if err != nil { http.Error(w,err.Error(),404); return }
		writeJSON(w, rm.Snapshot(uid))
	})
	r.Get("/rooms/{roomID}/ws", func(w http.ResponseWriter, r *http.Request){ hub.Serve(chi.URLParam(r,"roomID"), w, r) })
	return r
}

func writeJSON(w http.ResponseWriter, v any) { w.Header().Set("Content-Type","application/json"); _=json.NewEncoder(w).Encode(v) }
