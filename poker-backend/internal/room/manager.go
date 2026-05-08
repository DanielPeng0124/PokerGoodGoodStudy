package room

import (
	"errors"
	"sync"

	"poker-backend/internal/model"
)

type Manager struct { mu sync.RWMutex; rooms map[string]*Room }
func NewManager() *Manager { return &Manager{rooms: map[string]*Room{}} }
func (m *Manager) Create(ownerID string, settings model.RoomSettings) *Room { m.mu.Lock(); defer m.mu.Unlock(); r:=New(ownerID,settings); m.rooms[r.ID]=r; return r }
func (m *Manager) Get(id string) (*Room,error) { m.mu.RLock(); defer m.mu.RUnlock(); r:=m.rooms[id]; if r==nil { return nil, errors.New("room not found") }; return r,nil }
