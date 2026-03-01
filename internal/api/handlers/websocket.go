package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"moonshine/internal/api/ws"
	"moonshine/internal/config"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketHandler struct {
	hub    *ws.Hub
	config *config.Config
}

func NewWebSocketHandler(hub *ws.Hub, cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		hub:    hub,
		config: cfg,
	}
}

func (h *WebSocketHandler) HandleConnection(c echo.Context) error {
	tokenString := c.QueryParam("token")
	if tokenString == "" {
		fmt.Println("[WS] Missing token")
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing token"})
	}

	userID, err := h.validateToken(tokenString)
	if err != nil {
		fmt.Printf("[WS] Invalid token: %v\n", err)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		fmt.Printf("[WS] Upgrade error: %v\n", err)
		return err
	}

	fmt.Printf("[WS] Connection upgraded for user %s\n", userID)
	h.hub.Register(userID, conn)

	h.handleConnection(userID, conn)

	return nil
}

func (h *WebSocketHandler) handleConnection(userID uuid.UUID, conn *websocket.Conn) {
	defer func() {
		h.hub.Unregister(userID)
		conn.Close()
	}()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := h.hub.SendPing(userID); err != nil {
				return
			}
		}
	}
}

func (h *WebSocketHandler) validateToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.config.JWTKey), nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if idStr, ok := claims["id"].(string); ok {
			return uuid.Parse(idStr)
		}
	}

	return uuid.Nil, jwt.ErrInvalidKey
}
