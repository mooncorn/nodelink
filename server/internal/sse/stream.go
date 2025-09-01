package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/common"
)

// StreamBuilder provides a fluent API for creating SSE streams with conventions
type StreamBuilder struct {
	sseManager common.SSEManager
}

// NewStreamBuilder creates a new stream builder
func NewStreamBuilder(sseManager common.SSEManager) *StreamBuilder {
	return &StreamBuilder{
		sseManager: sseManager,
	}
}

// ForAgent creates an agent-specific stream builder
func (b *StreamBuilder) ForAgent(agentID string) *AgentStreamBuilder {
	return &AgentStreamBuilder{
		builder: b,
		agentID: agentID,
	}
}

// ForTerminal creates a terminal-specific stream builder
func (b *StreamBuilder) ForTerminal(sessionID string) *TerminalStreamBuilder {
	return &TerminalStreamBuilder{
		builder:   b,
		sessionID: sessionID,
	}
}

// ForMetrics creates a metrics-specific stream builder
func (b *StreamBuilder) ForMetrics(agentID string) *MetricsStreamBuilder {
	return &MetricsStreamBuilder{
		builder: b,
		agentID: agentID,
	}
}

// Global creates a global stream builder
func (b *StreamBuilder) Global() *GlobalStreamBuilder {
	return &GlobalStreamBuilder{
		builder: b,
	}
}

// AgentStreamBuilder handles agent status stream patterns
type AgentStreamBuilder struct {
	builder       *StreamBuilder
	agentID       string
	statusManager common.StatusManager
	allAgents     bool
}

// WithCurrentStatus includes current agent status in initial messages
func (a *AgentStreamBuilder) WithCurrentStatus(statusManager common.StatusManager) *AgentStreamBuilder {
	a.statusManager = statusManager
	return a
}

// AllAgents configures the stream to listen to all agent events
func (a *AgentStreamBuilder) AllAgents() *AgentStreamBuilder {
	a.allAgents = true
	return a
}

// Handle processes the SSE connection with agent-specific conventions
func (a *AgentStreamBuilder) Handle(c *gin.Context) error {
	// Extract agent ID from URL if not provided
	if a.agentID == "" && !a.allAgents {
		a.agentID = c.Param("agentId")
	}

	// Validate agent ID for specific agent streams
	if !a.allAgents && a.agentID == "" {
		c.JSON(400, gin.H{"error": "agent_id is required"})
		return fmt.Errorf("agent_id is required")
	}

	// Setup headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Generate client ID
	var clientIDComponents []string
	if a.allAgents {
		clientIDComponents = []string{c.Request.RemoteAddr}
	} else {
		clientIDComponents = []string{a.agentID, c.Request.RemoteAddr}
	}
	clientID := fmt.Sprintf("agent_%d", time.Now().UnixNano())
	if len(clientIDComponents) > 0 {
		for _, component := range clientIDComponents {
			if component != "" {
				clientID += "_" + component
			}
		}
	}

	// Add client to SSE manager
	client := a.builder.sseManager.AddClient(clientID)
	if client == nil {
		c.JSON(500, gin.H{"error": "Failed to create SSE client"})
		return fmt.Errorf("failed to create SSE client")
	}

	// Join appropriate rooms
	var rooms []string
	if a.allAgents {
		rooms = []string{"agents"}
	} else {
		rooms = []string{fmt.Sprintf("agent_%s", a.agentID)}
	}

	for _, room := range rooms {
		if err := a.builder.sseManager.JoinRoom(clientID, room); err != nil {
			log.Printf("Error joining room %s: %v", room, err)
		}
	}

	// Handle connection lifecycle
	defer a.builder.sseManager.RemoveClient(clientID)

	// Send initial messages
	a.sendInitialMessages(c, client)

	// Keep connection alive
	return a.handleConnection(c, client)
}

func (a *AgentStreamBuilder) sendInitialMessages(c *gin.Context, client common.SSEClient) {
	// Connection acknowledgment
	connectionMsg := a.formatConnectionMessage()
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", connectionMsg); err == nil {
		c.Writer.Flush()
	}

	// Current status if available and requested
	if !a.allAgents && a.statusManager != nil && a.agentID != "" {
		if agent, exists := a.statusManager.GetAgent(a.agentID); exists {
			statusMsg := a.formatCurrentStatusMessage(agent)
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", statusMsg); err == nil {
				c.Writer.Flush()
			}
		}
	}
}

func (a *AgentStreamBuilder) handleConnection(c *gin.Context, client common.SSEClient) error {
	for {
		select {
		case msg := <-client.GetChannel():
			formattedMsg := a.formatMessage(msg)
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", formattedMsg); err != nil {
				return fmt.Errorf("error writing SSE message: %v", err)
			}
			c.Writer.Flush()

		case <-c.Request.Context().Done():
			return nil
		case <-client.GetContext().Done():
			return nil
		}
	}
}

func (a *AgentStreamBuilder) formatConnectionMessage() string {
	var data map[string]interface{}
	if a.allAgents {
		data = map[string]interface{}{
			"event": "connection",
			"data":  map[string]string{"status": "connected", "scope": "all_agents"},
		}
	} else {
		data = map[string]interface{}{
			"event": "agent_connection",
			"data": map[string]string{
				"status":   "connected",
				"agent_id": a.agentID,
			},
		}
	}
	result, _ := json.Marshal(data)
	return string(result)
}

func (a *AgentStreamBuilder) formatCurrentStatusMessage(agent *common.AgentInfo) string {
	data := map[string]interface{}{
		"event": "current_status",
		"data":  agent,
	}
	result, _ := json.Marshal(data)
	return string(result)
}

func (a *AgentStreamBuilder) formatMessage(msg common.SSEMessage) string {
	data := map[string]interface{}{
		"event": msg.EventType,
		"data":  msg.Data,
		"room":  msg.Room,
	}
	result, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling SSE message: %v", err)
		errorData, _ := json.Marshal(map[string]interface{}{
			"event": msg.EventType,
			"error": "failed to marshal data",
		})
		return string(errorData)
	}
	return string(result)
}

// TerminalStreamBuilder handles terminal stream patterns
type TerminalStreamBuilder struct {
	builder        *StreamBuilder
	sessionID      string
	sessionManager common.TerminalSessionManager
	requireAuth    bool
	includeHistory bool
}

// RequireAuth enables authentication validation
func (t *TerminalStreamBuilder) RequireAuth(sessionManager common.TerminalSessionManager) *TerminalStreamBuilder {
	t.requireAuth = true
	t.sessionManager = sessionManager
	return t
}

// WithHistory includes terminal history in initial messages
func (t *TerminalStreamBuilder) WithHistory() *TerminalStreamBuilder {
	t.includeHistory = true
	return t
}

// Handle processes the SSE connection with terminal-specific conventions
func (t *TerminalStreamBuilder) Handle(c *gin.Context) error {
	// Extract session ID from URL if not provided
	if t.sessionID == "" {
		t.sessionID = c.Param("sessionId")
	}

	if t.sessionID == "" {
		c.JSON(400, gin.H{"error": "Session ID is required"})
		return fmt.Errorf("session ID is required")
	}

	// Authentication if required
	if t.requireAuth {
		userID := t.getUserIDFromContext(c)
		if userID == "" {
			c.JSON(401, gin.H{"error": "User authentication required"})
			return fmt.Errorf("user authentication required")
		}

		if err := t.sessionManager.ValidateSessionAccess(t.sessionID, userID); err != nil {
			switch err {
			case common.ErrTerminalSessionNotFound:
				c.JSON(404, gin.H{"error": "Terminal session not found"})
			case common.ErrUnauthorizedTerminalAccess:
				c.JSON(403, gin.H{"error": "Unauthorized access to terminal session"})
			case common.ErrTerminalSessionClosed:
				c.JSON(410, gin.H{"error": "Terminal session is closed"})
			default:
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return err
		}

		// Update last activity
		t.sessionManager.UpdateLastActivity(t.sessionID)
	}

	// Setup headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Generate client ID
	clientID := fmt.Sprintf("terminal_%s_%d", t.sessionID, time.Now().UnixNano())

	// Add client
	client := t.builder.sseManager.AddClient(clientID)
	if client == nil {
		c.JSON(500, gin.H{"error": "Failed to create SSE client"})
		return fmt.Errorf("failed to create SSE client")
	}

	// Join terminal room
	room := fmt.Sprintf("terminal_%s", t.sessionID)
	if err := t.builder.sseManager.JoinRoom(clientID, room); err != nil {
		log.Printf("Error joining terminal room %s: %v", room, err)
	}

	defer t.builder.sseManager.RemoveClient(clientID)

	// Send initial message
	t.sendInitialMessage(c)

	// Handle connection
	return t.handleConnection(c, client)
}

func (t *TerminalStreamBuilder) sendInitialMessage(c *gin.Context) {
	data := map[string]interface{}{
		"event": "terminal_connected",
		"data": map[string]interface{}{
			"session_id": t.sessionID,
			"message":    "Connected to terminal stream",
		},
	}
	msg, _ := json.Marshal(data)
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(msg)); err == nil {
		c.Writer.Flush()
	}
}

func (t *TerminalStreamBuilder) handleConnection(c *gin.Context, client common.SSEClient) error {
	for {
		select {
		case msg := <-client.GetChannel():
			formattedMsg := t.formatMessage(msg)
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", formattedMsg); err != nil {
				return fmt.Errorf("error writing SSE message: %v", err)
			}
			c.Writer.Flush()

		case <-c.Request.Context().Done():
			return nil
		case <-client.GetContext().Done():
			return nil
		}
	}
}

func (t *TerminalStreamBuilder) formatMessage(msg common.SSEMessage) string {
	data := map[string]interface{}{
		"event": msg.EventType,
		"data":  msg.Data,
		"room":  msg.Room,
	}
	result, _ := json.Marshal(data)
	return string(result)
}

func (t *TerminalStreamBuilder) getUserIDFromContext(c *gin.Context) string {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = c.Query("user_id")
	}
	return userID
}

// MetricsStreamBuilder handles metrics stream patterns
type MetricsStreamBuilder struct {
	builder       *StreamBuilder
	agentID       string
	statusManager common.StatusManager
	includeCached bool
	cachedMetrics interface{}
}

// WithCachedData includes cached metrics and system info
func (m *MetricsStreamBuilder) WithCachedData(metrics interface{}) *MetricsStreamBuilder {
	m.includeCached = true
	m.cachedMetrics = metrics
	return m
}

// WithStatusManager adds status validation
func (m *MetricsStreamBuilder) WithStatusManager(statusManager common.StatusManager) *MetricsStreamBuilder {
	m.statusManager = statusManager
	return m
}

// Handle processes the SSE connection with metrics-specific conventions
func (m *MetricsStreamBuilder) Handle(c *gin.Context) error {
	// Extract agent ID from URL if not provided
	if m.agentID == "" {
		m.agentID = c.Param("agentID")
	}

	if m.agentID == "" {
		c.JSON(400, gin.H{"error": "Agent ID is required"})
		return fmt.Errorf("agent ID is required")
	}

	// Check if agent is online
	if m.statusManager != nil && !m.statusManager.IsAgentOnline(m.agentID) {
		c.JSON(404, gin.H{"error": "Agent not found or offline"})
		return fmt.Errorf("agent not found or offline")
	}

	// Setup headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Generate client ID
	clientID := fmt.Sprintf("metrics_%s_%d", m.agentID, time.Now().UnixNano())

	// Add client
	client := m.builder.sseManager.AddClient(clientID)
	if client == nil {
		c.JSON(500, gin.H{"error": "Failed to create SSE client"})
		return fmt.Errorf("failed to create SSE client")
	}

	// Join metrics room
	room := fmt.Sprintf("metrics_%s", m.agentID)
	if err := m.builder.sseManager.JoinRoom(clientID, room); err != nil {
		log.Printf("Error joining metrics room %s: %v", room, err)
	}

	defer m.builder.sseManager.RemoveClient(clientID)

	// Send initial messages
	m.sendInitialMessages(c)

	// Handle connection
	return m.handleConnection(c, client)
}

func (m *MetricsStreamBuilder) sendInitialMessages(c *gin.Context) {
	if m.includeCached {
		if m.cachedMetrics != nil {
			data, _ := json.Marshal(m.cachedMetrics)
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(data)); err == nil {
				c.Writer.Flush()
			}
		}
	}
}

func (m *MetricsStreamBuilder) handleConnection(c *gin.Context, client common.SSEClient) error {
	for {
		select {
		case msg := <-client.GetChannel():
			if data, err := json.Marshal(msg.Data); err == nil {
				if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(data)); err != nil {
					return fmt.Errorf("error writing SSE message: %v", err)
				}
				c.Writer.Flush()
			}

		case <-c.Request.Context().Done():
			return nil
		case <-client.GetContext().Done():
			return nil
		}
	}
}

// GlobalStreamBuilder handles global stream patterns
type GlobalStreamBuilder struct {
	builder *StreamBuilder
}

// Handle processes global SSE connections
func (g *GlobalStreamBuilder) Handle(c *gin.Context) error {
	// Setup headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Generate client ID
	clientID := fmt.Sprintf("global_%d", time.Now().UnixNano())

	// Add client
	client := g.builder.sseManager.AddClient(clientID)
	if client == nil {
		c.JSON(500, gin.H{"error": "Failed to create SSE client"})
		return fmt.Errorf("failed to create SSE client")
	}

	defer g.builder.sseManager.RemoveClient(clientID)

	// Handle connection
	for {
		select {
		case msg := <-client.GetChannel():
			data, _ := json.Marshal(map[string]interface{}{
				"event": msg.EventType,
				"data":  msg.Data,
			})
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(data)); err != nil {
				return fmt.Errorf("error writing SSE message: %v", err)
			}
			c.Writer.Flush()

		case <-c.Request.Context().Done():
			return nil
		case <-client.GetContext().Done():
			return nil
		}
	}
}
