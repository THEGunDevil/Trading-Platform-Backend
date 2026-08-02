package ws

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	gen "github.com/internal/db/gen"
	"github.com/internal/service"
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
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Content   string `json:"content,omitempty"`
	ImageURL  string `json:"imageUrl,omitempty"`
}

// ---- Internal message types ----
type ChatMessage struct {
	SessionID uuid.UUID
	SenderID  uuid.UUID
	Content   string
	ImageURL  string
	IsAgent   bool
}

type TypingEvent struct {
	SessionID string
	UserID    string
	IsTyping  bool
}

type NewSessionEvent struct {
	Session  gen.SupportSession
	UserID   string
	UserName string
	Subject  string
}

type AssignAgentRequest struct {
	SessionID   uuid.UUID
	AgentID     uuid.UUID
	AgentUserID string
}

// ---- Hub ----
type SupportHub struct {
	clients        map[string]*SupportClient
	sessionClients map[string]map[string]*SupportClient
	Register       chan *SupportClient
	unregister     chan *SupportClient
	chatMsg        chan *ChatMessage
	newSession     chan *NewSessionEvent
	assignReq      chan AssignAgentRequest
	typingMsg      chan *TypingEvent
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
		typingMsg:      make(chan *TypingEvent),
		Queries:        queries,
	}
	go h.Run()
	return h
}

// Run is the main event loop. It recovers from panics so one bad message
// cannot bring down the entire hub.
func (h *SupportHub) Run() {
	log.Println("🔄 SupportHub event loop started")
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("🔥 PANIC in SupportHub.Run: %v\n%s", r, debug.Stack())
				}
			}()
			select {
			case client := <-h.Register:
				h.registerClient(client)
			case client := <-h.unregister:
				h.unregisterClient(client)
			case msg := <-h.chatMsg:
				h.processChatMessage(msg)
			case event := <-h.newSession:
				h.notifyAgentsOfNewSession(event)
			case event := <-h.typingMsg:
				h.processTypingEvent(event)
			case req := <-h.assignReq:
				h.assignAgent(req)
			}
		}()
	}
}

// ---- Client management ----
func (h *SupportHub) registerClient(client *SupportClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	log.Printf("📥 registerClient: user=%s session=%s", client.UserID, client.SessionID)
	h.clients[client.UserID] = client

	if client.SessionID != "" {
		if h.sessionClients[client.SessionID] == nil {
			h.sessionClients[client.SessionID] = make(map[string]*SupportClient)
		}
		h.sessionClients[client.SessionID][client.UserID] = client
		go h.sendMessageHistory(client)
	}

	if client.IsAgent {
		go h.sendPendingNotifications(client)
	}
	log.Printf("📊 Total clients: %d", len(h.clients))
}

func (h *SupportHub) unregisterClient(client *SupportClient) {
	h.mu.Lock()
	log.Printf("📤 unregisterClient: user=%s session=%s", client.UserID, client.SessionID)
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

	client.mu.Lock()
	client.closed = true
	client.mu.Unlock()

	close(client.Send)
	log.Printf("📊 Total clients after remove: %d", len(h.clients))
}

// ---- Chat processing ----
func (h *SupportHub) processChatMessage(msg *ChatMessage) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in processChatMessage: %v\n%s", r, debug.Stack())
		}
	}()
	log.Printf("💬 processChatMessage: session=%s sender=%s", msg.SessionID, msg.SenderID)

	if msg.SessionID == uuid.Nil {
		log.Println("⚠️  Dropping chat message with no session ID")
		return
	}

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
	log.Printf("💾 Message saved, id=%s", service.PGTypeToUUID(saved.ID))

	sessionIDStr := msg.SessionID.String()
	out := &WebSocketMessage{
		Type:      "chat",
		ID:        service.PGTypeToUUID(saved.ID).String(),
		SessionID: sessionIDStr,
		Content:   service.PGTextToString(saved.Content),
		ImageURL:  service.PGTextToString(saved.ImageUrl),
		SenderID:  msg.SenderID.String(),
		IsAgent:   msg.IsAgent,
		Timestamp: saved.CreatedAt.Time,
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	sClients, inRoom := h.sessionClients[sessionIDStr]
	if inRoom {
		log.Printf("📡 Broadcasting to %d session clients", len(sClients))
		for _, c := range sClients {
			select {
			case c.Send <- out:
			default:
				log.Printf("⚠️  Failed to send to client %s (buffer full)", c.UserID)
			}
		}
	}

	// Notify agents watching the dashboard (not necessarily in this session's
	// room) so they can update last_message previews. Skip agents already in
	// the room — they just got the full "chat" message above.
	dashboardMsg := &WebSocketMessage{
		Type:      "new_message",
		SessionID: sessionIDStr,
		Content:   out.Content,
		Timestamp: out.Timestamp,
	}
	for _, c := range h.clients {
		if !c.IsAgent {
			continue
		}
		if inRoom {
			if _, alreadyNotified := sClients[c.UserID]; alreadyNotified {
				continue
			}
		}
		select {
		case c.Send <- dashboardMsg:
		default:
			log.Printf("⚠️  Dashboard new_message push failed for agent %s (buffer full)", c.UserID)
		}
	}
}

// ---- Typing event ----
func (h *SupportHub) processTypingEvent(event *TypingEvent) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🔥 PANIC in processTypingEvent: %v\n%s", r, debug.Stack())
		}
	}()

	content := "false"
	if event.IsTyping {
		content = "true"
	}
	msg := &WebSocketMessage{
		Type:      "typing",
		SessionID: event.SessionID,
		SenderID:  event.UserID,
		Content:   content,
		Timestamp: time.Now(),
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	sClients, ok := h.sessionClients[event.SessionID]
	if !ok {
		return
	}
	for _, c := range sClients {
		if c.UserID == event.UserID {
			continue // don't echo back to sender
		}
		select {
		case c.Send <- msg:
		default:
			log.Printf("⚠️  Typing send failed for client %s (buffer full)", c.UserID)
		}
	}
}

// ---- New session notification ----
func (h *SupportHub) TriggerNewSessionNotification(session gen.SupportSession, userID, userName, subject string) {
	log.Printf("📢 TriggerNewSessionNotification: session=%s user=%s", session.ID, userID)
	h.newSession <- &NewSessionEvent{
		Session:  session,
		UserID:   userID,
		UserName: userName,
		Subject:  subject,
	}
}

func (h *SupportHub) notifyAgentsOfNewSession(event *NewSessionEvent) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🔥 PANIC in notifyAgentsOfNewSession: %v\n%s", r, debug.Stack())
		}
	}()

	ctx := context.Background()
	sessionID := service.PGTypeToUUID(event.Session.ID)

	agentIDs, err := h.Queries.ListAgentUserIDs(ctx)
	if err != nil {
		log.Printf("❌ ListAgentUserIDs error: %v", err)
	} else {
		for _, agentID := range agentIDs {
			if _, err := h.Queries.CreateSessionNotification(ctx, gen.CreateSessionNotificationParams{
				SessionID: event.Session.ID,
				AgentID:   agentID,
			}); err != nil {
				log.Printf("❌ CreateSessionNotification error for agent %v: %v", agentID, err)
			}
		}
	}

	msg := &WebSocketMessage{
		Type:      "new_session",
		ID:        uuid.New().String(),
		SessionID: sessionID.String(),
		Content:   fmt.Sprintf("New request from %s", event.UserName),
		Timestamp: time.Now(),
		ExtraData: map[string]interface{}{
			"id":                sessionID.String(),
			"user_id":           event.UserID,
			"user_name":         event.UserName,
			"subject":           event.Subject,
			"status":            "open",
			"assigned_agent_id": nil,
			"created_at":        event.Session.CreatedAt.Time,
			"updated_at":        event.Session.CreatedAt.Time,
		},
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		if !c.IsAgent {
			continue
		}
		select {
		case c.Send <- msg:
		default:
			log.Printf("⚠️  new_session push failed for agent %s (buffer full)", c.UserID)
		}
	}
}

// ---- Agent assignment ----
func (h *SupportHub) assignAgent(req AssignAgentRequest) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🔥 PANIC in assignAgent: %v\n%s", r, debug.Stack())
		}
	}()
	log.Printf("👥 assignAgent: session=%s agent=%s", req.SessionID, req.AgentID)

	ctx := context.Background()
	sessionIDStr := req.SessionID.String()

	_, err := h.Queries.AssignAgentToSession(ctx, gen.AssignAgentToSessionParams{
		AssignedAgentID: service.UUIDToPGType(req.AgentID),
		ID:              service.UUIDToPGType(req.SessionID),
	})

	h.mu.Lock()
	defer h.mu.Unlock()

	if err != nil {
		log.Printf("❌ assignAgent: could not assign (already taken?): %v", err)
		if c, ok := h.clients[req.AgentUserID]; ok {
			select {
			case c.Send <- &WebSocketMessage{
				Type:      "error",
				SessionID: sessionIDStr,
				Content:   "Session already taken or not open",
				Timestamp: time.Now(),
			}:
			default:
			}
		}
		return
	}

	agentClient, ok := h.clients[req.AgentUserID]
	if !ok {
		// Agent disconnected between claim and here — DB claim still succeeded,
		// so at minimum notify the room; nothing to send back to the agent.
		log.Printf("⚠️  assignAgent: agent %s no longer connected after claiming session %s", req.AgentUserID, sessionIDStr)
	} else {
		agentClient.mu.Lock()
		agentClient.SessionID = sessionIDStr
		agentClient.mu.Unlock()

		if h.sessionClients[sessionIDStr] == nil {
			h.sessionClients[sessionIDStr] = make(map[string]*SupportClient)
		}
		h.sessionClients[sessionIDStr][agentClient.UserID] = agentClient

		select {
		case agentClient.Send <- &WebSocketMessage{
			Type:      "assigned_success",
			SessionID: sessionIDStr,
			Content:   "You are now handling this conversation",
			Timestamp: time.Now(),
		}:
		default:
		}
	}

	// Tell the room (customer + agent) an agent joined.
	if sClients, ok := h.sessionClients[sessionIDStr]; ok {
		roomMsg := &WebSocketMessage{
			Type:      "assigned",
			SessionID: sessionIDStr,
			SenderID:  req.AgentID.String(),
			Content:   "An agent has joined your session.",
			Timestamp: time.Now(),
		}
		for _, c := range sClients {
			select {
			case c.Send <- roomMsg:
			default:
				log.Printf("⚠️  Failed to notify client %s of assignment", c.UserID)
			}
		}
	}

	// Tell every other agent to drop it from their unassigned list.
	takenMsg := &WebSocketMessage{
		Type:      "session_taken",
		SessionID: sessionIDStr,
		Content:   "Session claimed by another agent",
		Timestamp: time.Now(),
	}
	for _, c := range h.clients {
		if c.IsAgent && c.UserID != req.AgentUserID {
			select {
			case c.Send <- takenMsg:
			default:
			}
		}
	}

	log.Printf("✅ assignAgent: session=%s now owned by agent=%s", sessionIDStr, req.AgentID)
}

// ---- Helper send methods ----
// (add logs if needed)

// ---- Safe background goroutines ----
func (h *SupportHub) sendMessageHistory(client *SupportClient) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🔥 PANIC in sendMessageHistory: %v\n%s", r, debug.Stack())
		}
	}()
	log.Printf("📜 sendMessageHistory: client=%s session=%s", client.UserID, client.SessionID)

	ctx := context.Background()
	sessionUUID, _ := uuid.Parse(client.SessionID)

	messages, err := h.Queries.ListSessionMessages(ctx, service.UUIDToPGType(sessionUUID))
	if err != nil {
		log.Printf("❌ ListSessionMessages error: %v", err)
		return
	}
	log.Printf("📨 History: %d messages for session %s", len(messages), client.SessionID)

	for _, m := range messages {
		client.mu.Lock()
		if client.closed {
			client.mu.Unlock()
			log.Printf("🚫 Client %s closed, stopping history", client.UserID)
			return
		}
		client.mu.Unlock()

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

		select {
		case client.Send <- out:
		default:
			log.Printf("⚠️  History send failed for client %s (buffer full)", client.UserID)
			return
		}
	}
	log.Printf("✅ History sent to client %s", client.UserID)
}

func (h *SupportHub) sendPendingNotifications(client *SupportClient) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🔥 PANIC in sendPendingNotifications: %v\n%s", r, debug.Stack())
		}
	}()
	log.Printf("🔔 sendPendingNotifications: agent=%s", client.UserID)

	ctx := context.Background()
	agentUUID, _ := uuid.Parse(client.UserID)

	notifs, err := h.Queries.GetAgentNotifications(ctx, service.UUIDToPGType(agentUUID))
	if err != nil {
		log.Printf("❌ GetAgentNotifications error: %v", err)
		return
	}
	log.Printf("📬 Pending notifications: %d for agent %s", len(notifs), client.UserID)

	for _, n := range notifs {
		client.mu.Lock()
		if client.closed {
			client.mu.Unlock()
			log.Printf("🚫 Agent %s closed, stopping notifications", client.UserID)
			return
		}
		client.mu.Unlock()

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
			_ = h.Queries.MarkNotificationAsRead(ctx, gen.MarkNotificationAsReadParams{
				ID:      n.ID,
				AgentID: service.UUIDToPGType(agentUUID),
			})
		default:
			log.Printf("⚠️  Notification send failed for agent %s (buffer full)", client.UserID)
			return
		}
	}
	log.Printf("✅ Notifications sent to agent %s", client.UserID)
}

func (h *SupportHub) SendTypingEvent(sessionID, userID string, isTyping bool) {
	h.typingMsg <- &TypingEvent{
		SessionID: sessionID,
		UserID:    userID,
		IsTyping:  isTyping,
	}
}

// ---- SupportClient ----
type SupportClient struct {
	Hub    *SupportHub
	ConnID string
	UserID    string
	IsAgent   bool
	SessionID string
	Conn      *websocket.Conn
	Send      chan *WebSocketMessage
	closed    bool
	mu        sync.Mutex
}

func (c *SupportClient) ReadPump() {
	log.Printf("👂 ReadPump started for user %s", c.UserID)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🔥 PANIC in ReadPump: %v\n%s", r, debug.Stack())
		}
		c.Hub.unregister <- c
		c.Conn.Close()
		log.Printf("👋 ReadPump ended for user %s", c.UserID)
	}()

	for {
		var raw IncomingChatMessage
		err := c.Conn.ReadJSON(&raw)
		if err != nil {
			log.Printf("🔌 ReadJSON error for user %s: %v", c.UserID, err)
			break
		}
		log.Printf("📩 Incoming from %s: type=%s session=%s", c.UserID, raw.Type, raw.SessionID)

		switch raw.Type {
		case "chat":
			sessionUUID, err1 := uuid.Parse(raw.SessionID)
			senderUUID, err2 := uuid.Parse(c.UserID)
			if err1 != nil || err2 != nil {
				log.Printf("❌ Parse error in chat: session=%s user=%s", raw.SessionID, c.UserID)
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

		case "typing":
			isTyping := raw.Content == "true"
			c.Hub.SendTypingEvent(raw.SessionID, c.UserID, isTyping)
		}
	}
}

func (c *SupportClient) WritePump() {
	log.Printf("✍️  WritePump started for user %s", c.UserID)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🔥 PANIC in WritePump: %v\n%s", r, debug.Stack())
		}
		c.Conn.Close()
		log.Printf("👋 WritePump ended for user %s", c.UserID)
	}()

	for msg := range c.Send {
		if err := c.Conn.WriteJSON(msg); err != nil {
			log.Printf("📤 WriteJSON error for user %s: %v", c.UserID, err)
			break
		}
		log.Printf("✉️  Sent %s to user %s", msg.Type, c.UserID)
	}
}
// BroadcastToSession sends a message to all connected clients in the given session.
func (h *SupportHub) BroadcastToSession(sessionID string, msg *WebSocketMessage) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    if clients, ok := h.sessionClients[sessionID]; ok {
        for _, c := range clients {
            select {
            case c.Send <- msg:
            default:
            }
        }
    }
}