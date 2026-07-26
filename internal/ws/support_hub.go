package ws

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ---- Outgoing WebSocket message (to frontend) ----
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id,omitempty"`
	SessionID string                 `json:"sessionId,omitempty"`
	Content   string                 `json:"content,omitempty"`
	ImageURL  string                 `json:"imageUrl,omitempty"`
	SenderID  string                 `json:"senderId,omitempty"`
	IsAgent   bool                   `json:"isAgent,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	ExtraData map[string]interface{} `json:"extraData,omitempty"`
}

// ---- Incoming WebSocket message (from frontend) ----
type IncomingChatMessage struct {
	Type      string `json:"type"`      // "chat" or "agent_take"
	SessionID string `json:"sessionId"` // string from JSON
	Content   string `json:"content,omitempty"`
	ImageURL  string `json:"imageUrl,omitempty"`
}

// Parsed version used internally
type ChatMessage struct {
	SessionID uuid.UUID
	SenderID  uuid.UUID
	Content   string
	ImageURL  string
	IsAgent   bool
}

// ---- Events for the hub ----
type NewSessionEvent struct {
	Session  gen.SupportSession
	UserID   string
	UserName string
	Subject  string
}

type AssignAgentRequest struct {
	SessionID   uuid.UUID
	AgentID     uuid.UUID
	AgentUserID string // user id of the agent (for identification)
}

// ---- Hub ----
type SupportHub struct {
	clients        map[string]*SupportClient
	sessionClients map[string]map[string]*SupportClient
	Register       chan *SupportClient // ← exported
	unregister     chan *SupportClient
	chatMsg        chan *ChatMessage
	newSession     chan *NewSessionEvent
	assignReq      chan AssignAgentRequest
	mu             sync.RWMutex
	Queries        *gen.Queries
}

func NewSupportHub(queries *gen.Queries) *SupportHub {
	h := &SupportHub{
		clients:        make(map[string]*SupportClient),
		sessionClients: make(map[string]map[string]*SupportClient),
		Register:       make(chan *SupportClient),
		unregister:     make(chan *SupportClient),
		chatMsg:        make(chan *ChatMessage),
		newSession:     make(chan *NewSessionEvent),
		assignReq:      make(chan AssignAgentRequest),
		Queries:        queries,
	}
	go h.Run()
	return h
}

func (h *SupportHub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case msg := <-h.chatMsg:
			h.processChatMessage(msg)
		case event := <-h.newSession:
			h.notifyAgentsOfNewSession(event)
		case req := <-h.assignReq:
			h.assignAgent(req)
		}
	}
}

// ---- Client management ----
func (h *SupportHub) registerClient(client *SupportClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.UserID] = client

	if client.SessionID != "" {
		if h.sessionClients[client.SessionID] == nil {
			h.sessionClients[client.SessionID] = make(map[string]*SupportClient)
		}
		h.sessionClients[client.SessionID][client.UserID] = client
		// Send history
		go h.sendMessageHistory(client)
	}

	if client.IsAgent {
		go h.sendPendingNotifications(client)
	}
}
func (h *SupportHub) unregisterClient(client *SupportClient) {
	h.mu.Lock()
	delete(h.clients, client.UserID)
	if client.SessionID != "" {
		if sClients, ok := h.sessionClients[client.SessionID]; ok {
			delete(sClients, client.UserID)
			if len(sClients) == 0 {
				delete(h.sessionClients, client.SessionID)
			}
		}
	}
	h.mu.Unlock()

	// Mark client as closed to stop writes
	client.mu.Lock()
	client.closed = true
	client.mu.Unlock()

	close(client.Send)
}

// ---- Chat processing ----
func (h *SupportHub) processChatMessage(msg *ChatMessage) {
	ctx := context.Background()

	saved, err := h.Queries.SendSupportMessage(ctx, gen.SendSupportMessageParams{
		SessionID: service.UUIDToPGType(msg.SessionID),
		SenderID:  service.UUIDToPGType(msg.SenderID),
		Content:   service.StringToPGText(msg.Content),
		ImageUrl:  service.StringToPGText(msg.ImageURL),
		IsAgent:   msg.IsAgent,
	})
	if err != nil {
		log.Printf("Failed to save message: %v", err)
		return
	}

	out := &WebSocketMessage{
		Type:      "chat",
		ID:        service.PGTypeToUUID(saved.ID).String(),
		SessionID: msg.SessionID.String(),
		Content:   service.PGTextToString(saved.Content),
		ImageURL:  service.PGTextToString(saved.ImageUrl),
		SenderID:  msg.SenderID.String(),
		IsAgent:   msg.IsAgent,
		Timestamp: saved.CreatedAt.Time,
	}

	h.mu.RLock()
	if sClients, ok := h.sessionClients[msg.SessionID.String()]; ok {
		for _, c := range sClients {
			select {
			case c.Send <- out:
			default:
			}
		}
	}
	h.mu.RUnlock()
}

// ---- New session notification ----
func (h *SupportHub) TriggerNewSessionNotification(session gen.SupportSession, userID, userName, subject string) {
	h.newSession <- &NewSessionEvent{
		Session:  session,
		UserID:   userID,
		UserName: userName,
		Subject:  subject,
	}
}

func (h *SupportHub) notifyAgentsOfNewSession(event *NewSessionEvent) {
	// 1. Persist notifications for all online agents
	h.mu.RLock()
	// ✅ এই অংশে লগ যোগ করুন (totalClients ও agentCount দেখার জন্য)
	totalClients := len(h.clients)
	agentCount := 0
	for _, client := range h.clients {
		if client.IsAgent {
			agentCount++
		}
	}
	log.Printf("📢 Total clients: %d, Agents: %d", totalClients, agentCount)
	// ⬆️ উপরের ৬ লাইন যোগ করুন
	
	for uid, client := range h.clients {
		if !client.IsAgent {
			continue
		}
		agentUUID, _ := uuid.Parse(uid)
		_, err := h.Queries.CreateSessionNotification(context.Background(), gen.CreateSessionNotificationParams{
			SessionID: event.Session.ID,
			AgentID:   service.UUIDToPGType(agentUUID),
		})
		if err != nil {
			log.Printf("Failed to save notif for agent %s: %v", uid, err)
		}
	}
	h.mu.RUnlock()

	// 2. Broadcast to online agents immediately
	msg := &WebSocketMessage{
		Type:      "new_session",
		SessionID: event.Session.ID.String(),
		Content:   fmt.Sprintf("New request from %s", event.UserName),
		Timestamp: time.Now(),
		ExtraData: map[string]interface{}{
			"userName":  event.UserName,
			"subject":   event.Subject,
			"createdAt": event.Session.CreatedAt,
		},
	}

	h.mu.RLock()
	for _, client := range h.clients {
		if client.IsAgent {
			select {
			case client.Send <- msg:
			default:
			}
		}
	}
	h.mu.RUnlock()
}

// ---- Agent assignment ----
func (h *SupportHub) assignAgent(req AssignAgentRequest) {
	ctx := context.Background()

	// 1. Atomic assignment
	assigned, err := h.Queries.AssignAgentToSession(ctx, gen.AssignAgentToSessionParams{
		AssignedAgentID: service.UUIDToPGType(req.AgentID),
		ID:              service.UUIDToPGType(req.SessionID),
	})
	if err != nil {
		// Session no longer available
		h.sendToAgent(req.AgentUserID, &WebSocketMessage{
			Type:      "error",
			SessionID: req.SessionID.String(),
			Content:   "Session is no longer available.",
			Timestamp: time.Now(),
		})
		return
	}

	// 2. Expire all pending notifications
	_, _ = h.Queries.ExpireAllNotifications(ctx, service.UUIDToPGType(req.SessionID))

	userID := service.PGTypeToUUID(assigned.UserID).String()

	// 3. Notify the user
	h.sendToSessionParticipant(req.SessionID.String(), userID, &WebSocketMessage{
		Type:      "assigned",
		SessionID: req.SessionID.String(),
		Content:   "An agent has joined your session.",
		Timestamp: time.Now(),
	})

	// 4. Notify other agents
	takenMsg := &WebSocketMessage{
		Type:      "session_taken",
		SessionID: req.SessionID.String(),
		Content:   "Session taken by another agent.",
		Timestamp: time.Now(),
	}
	h.broadcastToOtherAgents(req.AgentUserID, takenMsg)

	// 5. Confirm to the taking agent
	h.sendToAgent(req.AgentUserID, &WebSocketMessage{
		Type:      "assigned_success",
		SessionID: req.SessionID.String(),
		Content:   "You have been assigned to this session.",
		Timestamp: time.Now(),
	})

	// 6. Add agent to session participants
	h.mu.Lock()
	if sClients, ok := h.sessionClients[req.SessionID.String()]; ok {
		if agentClient, exists := h.clients[req.AgentUserID]; exists {
			agentClient.SessionID = req.SessionID.String()
			sClients[req.AgentUserID] = agentClient
		}
	}
	h.mu.Unlock()
}

// ---- Helper send methods ----
func (h *SupportHub) sendToSessionParticipant(sessionID, userID string, msg *WebSocketMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if sClients, ok := h.sessionClients[sessionID]; ok {
		if c, ok := sClients[userID]; ok {
			select {
			case c.Send <- msg:
			default:
			}
		}
	}
}

func (h *SupportHub) sendToAgent(agentID string, msg *WebSocketMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if c, ok := h.clients[agentID]; ok && c.IsAgent {
		select {
		case c.Send <- msg:
		default:
		}
	}
}

func (h *SupportHub) broadcastToOtherAgents(excludeID string, msg *WebSocketMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for id, c := range h.clients {
		if c.IsAgent && id != excludeID {
			select {
			case c.Send <- msg:
			default:
			}
		}
	}
}

func (h *SupportHub) sendPendingNotifications(client *SupportClient) {
	ctx := context.Background()
	agentUUID, _ := uuid.Parse(client.UserID)

	notifs, err := h.Queries.GetAgentNotifications(ctx, service.UUIDToPGType(agentUUID))
	if err != nil {
		return
	}

	for _, n := range notifs {
		msg := &WebSocketMessage{
			Type:      "new_session",
			ID:        service.PGTypeToUUID(n.ID).String(),
			SessionID: service.PGTypeToUUID(n.SessionID).String(),
			Content:   fmt.Sprintf("New request from %s", n.UserName),
			Timestamp: n.CreatedAt.Time,
			ExtraData: map[string]interface{}{
				"userName":  n.UserName,
				"subject":   n.Subject,
				"createdAt": n.SessionCreatedAt,
			},
		}
		select {
		case client.Send <- msg:
			// Mark as read so it won't be sent again on reconnect
			_ = h.Queries.MarkNotificationAsRead(ctx, gen.MarkNotificationAsReadParams{
				ID:      n.ID,
				AgentID: service.UUIDToPGType(agentUUID),
			})
		default:
		}
	}
}

func (h *SupportHub) sendMessageHistory(client *SupportClient) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered in sendMessageHistory: %v", r)
		}
	}()

	ctx := context.Background()
	sessionUUID, _ := uuid.Parse(client.SessionID)

	messages, err := h.Queries.ListSessionMessages(ctx, service.UUIDToPGType(sessionUUID))
	if err != nil {
		return
	}

	for _, m := range messages {
		out := &WebSocketMessage{
			Type:      "chat",
			ID:        service.PGTypeToUUID(m.ID).String(),
			SessionID: client.SessionID,
			Content:   service.PGTextToString(m.Content),
			ImageURL:  service.PGTextToString(m.ImageUrl),
			SenderID:  service.PGTypeToUUID(m.SenderID).String(),
			IsAgent:   m.IsAgent,
			Timestamp: m.CreatedAt.Time,
		}

		// Non-blocking send: if the channel is full or closed, stop sending
		select {
		case client.Send <- out:
		default:
			// Client might be slow or disconnected, stop sending history
			return
		}
	}
}

// ---- SupportClient ----
type SupportClient struct {
	Hub       *SupportHub
	UserID    string
	IsAgent   bool
	SessionID string
	Conn      *websocket.Conn
	Send      chan *WebSocketMessage
	closed    bool
	mu        sync.Mutex
}

func (c *SupportClient) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		var raw IncomingChatMessage
		err := c.Conn.ReadJSON(&raw)
		if err != nil {
			break
		}

		switch raw.Type {
		case "chat":
			sessionUUID, err1 := uuid.Parse(raw.SessionID)
			senderUUID, err2 := uuid.Parse(c.UserID)
			if err1 != nil || err2 != nil {
				continue
			}
			c.Hub.chatMsg <- &ChatMessage{
				SessionID: sessionUUID,
				SenderID:  senderUUID,
				Content:   raw.Content,
				ImageURL:  raw.ImageURL,
				IsAgent:   c.IsAgent,
			}

		case "agent_take":
			sessionUUID, _ := uuid.Parse(raw.SessionID)
			agentUUID, _ := uuid.Parse(c.UserID)
			c.Hub.assignReq <- AssignAgentRequest{
				SessionID:   sessionUUID,
				AgentID:     agentUUID,
				AgentUserID: c.UserID,
			}
		}
	}
}

func (c *SupportClient) WritePump() {
	defer c.Conn.Close()
	for msg := range c.Send {
		if err := c.Conn.WriteJSON(msg); err != nil {
			break
		}
	}
}
