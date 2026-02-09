package ws

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type HPUpdateData struct {
	CurrentHp int  `json:"currentHp"`
	Hp        uint `json:"hp"`
}

type Hub struct {
	connections map[uuid.UUID]*websocket.Conn
	mu          sync.RWMutex
}

var globalHub *Hub
var once sync.Once

func GetHub() *Hub {
	once.Do(func() {
		globalHub = &Hub{
			connections: make(map[uuid.UUID]*websocket.Conn),
		}
	})
	return globalHub
}

func (h *Hub) Register(userID uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.connections[userID] = conn
	fmt.Printf("[Hub] User %s connected. Total connections: %d\n", userID, len(h.connections))
}

func (h *Hub) Unregister(userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conn, exists := h.connections[userID]; exists {
		conn.Close()
		delete(h.connections, userID)
		fmt.Printf("[Hub] User %s disconnected. Total connections: %d\n", userID, len(h.connections))
	}
}

func (h *Hub) SendToUser(userID uuid.UUID, msg Message) error {
	h.mu.RLock()
	conn, exists := h.connections[userID]
	h.mu.RUnlock()

	if !exists {
		return nil
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, data)
}

func (h *Hub) SendHPUpdate(userID uuid.UUID, currentHp int, hp uint) error {
	// Ensure currentHp is not negative before sending
	if currentHp < 0 {
		currentHp = 0
	}

	msg := Message{
		Type: "hp_update",
		Data: HPUpdateData{
			CurrentHp: currentHp,
			Hp:        hp,
		},
	}
	return h.SendToUser(userID, msg)
}

func (h *Hub) IsConnected(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.connections[userID]
	return exists
}

func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}

func (h *Hub) GetConnectedUserIDs() []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	userIDs := make([]uuid.UUID, 0, len(h.connections))
	for userID := range h.connections {
		userIDs = append(userIDs, userID)
	}
	return userIDs
}
