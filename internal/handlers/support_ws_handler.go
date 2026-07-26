package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

type SupportWSHandler struct {
	Hub     *ws.SupportHub
	Queries *gen.Queries
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // adjust for production
}

func validateToken(tokenStr string) (uuid.UUID, string, error) {
	// Load .env (same as auth service)
	godotenv.Load()

	// ✅ Use same secret as GenerateAccessToken
	secret := os.Getenv("JWT_ACCESS_SECRET")

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Ensure token method is HMAC (same as auth service)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil // ✅ Convert to []byte same as auth service
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
    // 1. Extract token
    token := c.Query("token")
    if token == "" {
        authHeader := c.GetHeader("Authorization")
        if strings.HasPrefix(authHeader, "Bearer ") {
            token = strings.TrimPrefix(authHeader, "Bearer ")
        }
    }
    if token == "" {
        service.AbortWithError(c, http.StatusUnauthorized, "missing token")
        return
    }

    // 2. Validate token AND capture the role
    userID, role, err := validateToken(token)   // ✅ capture role
    if err != nil {
        service.AbortWithError(c, http.StatusUnauthorized, "invalid token")
        return
    }

    sessionID := c.Query("session_id")

    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }

    client := &ws.SupportClient{
        Hub:       h.Hub,
        UserID:    userID.String(),
        IsAgent:   role == "agent" || role == "admin",   // ✅ directly from JWT
        SessionID: sessionID,
        Conn:      conn,
        Send:      make(chan *ws.WebSocketMessage, 256),
    }

    h.Hub.Register <- client

    go client.WritePump()
    go client.ReadPump()
}