package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
    "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
)

type AgentHandler struct {
    Queries *gen.Queries
}

func NewAgentHandler(queries *gen.Queries) *AgentHandler {
    return &AgentHandler{Queries: queries}
}

// GET /agent/conversations
func (h *AgentHandler) ListConversations(c *gin.Context) {
    sessions, err := h.Queries.ListAllSessionsWithUser(c.Request.Context())
    if err != nil {
        service.AbortWithError(c, http.StatusInternalServerError, "failed to fetch sessions")
        return
    }
    c.JSON(http.StatusOK, sessions)
}

// GET /agent/conversations/:id
func (h *AgentHandler) GetConversation(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        service.AbortWithError(c, http.StatusBadRequest, "invalid id")
        return
    }
    session, err := h.Queries.GetSupportSessionByID(c.Request.Context(), service.UUIDToPGType(id))
    if err != nil {
        service.AbortWithError(c, http.StatusNotFound, "session not found")
        return
    }
    c.JSON(http.StatusOK, session)
}

// POST /agent/conversations/:id/assign
func (h *AgentHandler) AssignConversation(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        service.AbortWithError(c, http.StatusBadRequest, "invalid id")
        return
    }
    userID, ok := service.UserIDFromContext(c)
    if !ok {
        service.AbortWithError(c, http.StatusUnauthorized, "unauthorized")
        return
    }
    // Atomic assignment (status must be 'open' and assigned_agent_id NULL)
    _, err = h.Queries.AssignAgentToSession(c.Request.Context(), gen.AssignAgentToSessionParams{
        AssignedAgentID: service.UUIDToPGType(userID),
        ID:              service.UUIDToPGType(id),
    })
    if err != nil {
        service.AbortWithError(c, http.StatusConflict, "session already taken or not open")
        return
    }
    c.Status(http.StatusOK)
}

// POST /agent/conversations/:id/close
func (h *AgentHandler) CloseConversation(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        service.AbortWithError(c, http.StatusBadRequest, "invalid id")
        return
    }
    // Only close assigned sessions
    err = h.Queries.CloseSession(c.Request.Context(), service.UUIDToPGType(id))
    if err != nil {
        service.AbortWithError(c, http.StatusInternalServerError, "failed to close session")
        return
    }
    c.Status(http.StatusOK)
}