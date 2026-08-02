package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	gen "github.com/internal/db/gen"
	"github.com/internal/service"
	"github.com/internal/ws"
)

type SupportWSHandler struct {
	Hub       *ws.SupportHub
	Queries   *gen.Queries
	JWTSecret string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *SupportWSHandler) validateToken(tokenStr string) (uuid.UUID, string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(h.JWTSecret), nil
	})
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return uuid.Nil, "", fmt.Errorf("invalid token claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, "", fmt.Errorf("missing sub claim")
	}

	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid user id: %w", err)
	}

	role, _ := claims["role"].(string)
	if role == "" {
		role = "user"
	}

	return userID, role, nil
}

func (h *SupportWSHandler) HandleWebSocket(c *gin.Context) {
	log.Println("━━━━━━━━━━ WS REQUEST START ━━━━━━━━━━")
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		log.Println("❌ WS rejected: missing token")
		service.AbortWithError(c, http.StatusUnauthorized, "missing token")
		return
	}

	masked := token
	if len(masked) > 10 {
		masked = masked[:10] + "..."
	}
	log.Printf("🔑 WS token: %s", masked)

	userID, role, err := h.validateToken(token)
	if err != nil {
		log.Printf("❌ WS auth failed: %v", err)
		service.AbortWithError(c, http.StatusUnauthorized, "invalid token")
		return
	}
	log.Printf("✅ WS authenticated user=%s role=%s", userID, role)

	sessionID := c.Query("session_id")
	log.Printf("🧩 WS session_id=%q", sessionID)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("💥 WS upgrade error: %v", err)
		return
	}
	log.Printf("✅ WebSocket upgraded for user=%s", userID)

	client := &ws.SupportClient{
		Hub:       h.Hub,
		UserID:    userID.String(),
		IsAgent:   role == "agent" || role == "admin",
		SessionID: sessionID,
		Conn:      conn,
		Send:      make(chan *ws.WebSocketMessage, 256),
	}

	h.Hub.Register <- client
	log.Printf("📨 Client %s registered to hub", userID)

	go client.WritePump()
	go client.ReadPump()
	log.Printf("🚀 Goroutines started for user %s", userID)
}
