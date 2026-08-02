package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gen "github.com/internal/db/gen"
	"github.com/internal/service"
)

type AgentHandler struct {
	Queries *gen.Queries
}

func NewAgentHandler(queries *gen.Queries) *AgentHandler {
	return &AgentHandler{Queries: queries}
}

// SessionResponse is the explicit, frontend-facing shape for a support session.
// Defined here (not left to sqlc's generated struct) so JSON keys are
// guaranteed snake_case regardless of sqlc.yaml config.
type SessionResponse struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	Subject         string     `json:"subject"`
	Status          string     `json:"status"`
	AssignedAgentID *string    `json:"assigned_agent_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	AssignedAt      *time.Time `json:"assigned_at,omitempty"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	UserName        string     `json:"user_name"`
	UserEmail       string     `json:"user_email"`
}

// toSessionResponse maps a sqlc-generated ListAllSessionsWithUser row into
// the explicit response shape.
func toSessionResponse(row gen.ListAllSessionsWithUserRow) SessionResponse {
	resp := SessionResponse{
		ID:        service.PGTypeToUUID(row.ID).String(),
		UserID:    service.PGTypeToUUID(row.UserID).String(),
		Subject:   row.Subject,
		Status:    row.Status,
		UserName:  row.UserName,
		UserEmail: row.UserEmail,
	}

	if row.CreatedAt.Valid {
		resp.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		resp.UpdatedAt = row.UpdatedAt.Time
	}
	if row.AssignedAgentID.Valid {
		id := service.PGTypeToUUID(row.AssignedAgentID).String()
		resp.AssignedAgentID = &id
	}
	if row.AssignedAt.Valid {
		t := row.AssignedAt.Time
		resp.AssignedAt = &t
	}
	if row.ClosedAt.Valid {
		t := row.ClosedAt.Time
		resp.ClosedAt = &t
	}

	return resp
}

// GET /agent/conversations
func (h *AgentHandler) ListConversations(c *gin.Context) {
	sessions, err := h.Queries.ListAllSessionsWithUser(c.Request.Context())
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to fetch sessions")
		return
	}

	out := make([]SessionResponse, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, toSessionResponse(s))
	}

	c.JSON(http.StatusOK, out)
}

// GET /agent/conversations/:id
func (h *AgentHandler) GetConversation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid id")
		return
	}
	row, err := h.Queries.GetSessionWithUserByID(c.Request.Context(), service.UUIDToPGType(id))
	if err != nil {
		service.AbortWithError(c, http.StatusNotFound, "session not found")
		return
	}

	resp := SessionResponse{
		ID:        service.PGTypeToUUID(row.ID).String(),
		UserID:    service.PGTypeToUUID(row.UserID).String(),
		Subject:   row.Subject,
		Status:    row.Status,
		UserName:  row.UserName,
		UserEmail: row.UserEmail,
	}
	if row.CreatedAt.Valid {
		resp.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		resp.UpdatedAt = row.UpdatedAt.Time
	}
	if row.AssignedAgentID.Valid {
		aid := service.PGTypeToUUID(row.AssignedAgentID).String()
		resp.AssignedAgentID = &aid
	}
	if row.AssignedAt.Valid {
		t := row.AssignedAt.Time
		resp.AssignedAt = &t
	}
	if row.ClosedAt.Valid {
		t := row.ClosedAt.Time
		resp.ClosedAt = &t
	}

	c.JSON(http.StatusOK, resp)
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
