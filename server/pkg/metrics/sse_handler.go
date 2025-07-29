package metrics

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/pkg/sse"
	"github.com/mooncorn/nodelink/server/pkg/tasks"
)

// SSEHandler handles SSE streaming for metrics
type SSEHandler struct {
	store         *MetricsStore
	sseManager    *sse.Manager[*pb.MetricsDataResponse]
	taskManager   *tasks.TaskManager
	clientCounter map[string]int // tracks number of clients per agent
	counterMutex  sync.RWMutex
}

// NewSSEHandler creates a new SSE handler for metrics
func NewSSEHandler(store *MetricsStore, taskManager *tasks.TaskManager) *SSEHandler {
	// Create SSE manager for metrics
	config := sse.ManagerConfig{
		BufferSize:     100,
		EnableRooms:    true,
		EnableMetadata: true,
	}

	handler := &SSEHandler{
		store:         store,
		taskManager:   taskManager,
		clientCounter: make(map[string]int),
	}

	// Create custom event handler that handles client disconnections
	eventHandler := sse.EventHandler[*pb.MetricsDataResponse]{
		OnConnect: func(client *sse.Client[*pb.MetricsDataResponse]) {
			log.Printf("SSE Client connected: %s", client.ID)
		},
		OnDisconnect: func(client *sse.Client[*pb.MetricsDataResponse]) {
			log.Printf("SSE Client disconnected: %s", client.ID)
			handler.handleClientDisconnect(client)
		},
		OnMessage: func(message sse.Message[*pb.MetricsDataResponse]) {
			// Default message handling
		},
		OnError: func(clientID sse.ClientID, err error) {
			log.Printf("SSE Error for client %s: %v", clientID, err)
		},
	}

	sseManager := sse.NewManager(config, eventHandler)

	handler.sseManager = sseManager
	return handler
}

// Start starts the SSE manager
func (h *SSEHandler) Start() {
	h.sseManager.Start()
}

// Stop stops the SSE manager
func (h *SSEHandler) Stop() {
	h.sseManager.Stop()
}

// RegisterRoutes registers SSE routes with the given router
func (h *SSEHandler) RegisterRoutes(router gin.IRouter) {
	// Real-time metrics streaming
	router.GET("/agents/:agentId/metrics/stream",
		sse.SSEHeaders(),
		sse.SSEConnection(h.sseManager),
		h.StreamAgentMetrics)
}

// StreamAgentMetrics handles SSE streaming for a specific agent's metrics
func (h *SSEHandler) StreamAgentMetrics(c *gin.Context) {
	agentID := c.Param("agentId")
	intervalStr := c.DefaultQuery("interval", "5")

	client, ok := sse.GetClientFromContext[*pb.MetricsDataResponse](c)
	if !ok {
		c.Status(500)
		return
	}

	// Parse interval
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval < 1 {
		interval = 5 // default 5 seconds
	}

	// Join room for this agent's metrics
	roomName := fmt.Sprintf("metrics:%s", agentID)
	h.sseManager.JoinRoom(client.ID, roomName)

	// Track client connection for this agent
	h.trackClientConnection(agentID, client.ID)

	// Check if we need to start streaming (first client for this agent)
	h.ensureStreamingStarted(agentID, interval)

	log.Printf("Client %s started streaming metrics for agent %s with interval %ds",
		client.ID, agentID, interval)

	// Send current metrics if available
	if metrics, err := h.store.GetCurrentMetrics(agentID); err == nil {
		h.sseManager.SendToClient(client.ID, metrics, "metrics")
	}

	// Handle the stream
	sse.HandleSSEStream[*pb.MetricsDataResponse](c)
}

// trackClientConnection increments the client counter for an agent
func (h *SSEHandler) trackClientConnection(agentID string, clientID sse.ClientID) {
	h.counterMutex.Lock()
	defer h.counterMutex.Unlock()
	h.clientCounter[agentID]++
	log.Printf("Agent %s now has %d SSE clients", agentID, h.clientCounter[agentID])
}

// handleClientDisconnect handles when an SSE client disconnects
func (h *SSEHandler) handleClientDisconnect(client *sse.Client[*pb.MetricsDataResponse]) {
	// Check which rooms this client was in (specifically metrics rooms)
	for roomName := range client.Rooms {
		if len(roomName) > 8 && roomName[:8] == "metrics:" {
			agentID := roomName[8:] // extract agent ID from "metrics:agentID"
			h.handleClientDisconnectFromAgent(agentID, client.ID)
		}
	}
}

// handleClientDisconnectFromAgent handles when a client disconnects from a specific agent's metrics
func (h *SSEHandler) handleClientDisconnectFromAgent(agentID string, clientID sse.ClientID) {
	h.counterMutex.Lock()
	defer h.counterMutex.Unlock()

	if count, exists := h.clientCounter[agentID]; exists && count > 0 {
		h.clientCounter[agentID]--
		log.Printf("Agent %s now has %d SSE clients", agentID, h.clientCounter[agentID])

		// If this was the last client, stop streaming
		if h.clientCounter[agentID] == 0 {
			delete(h.clientCounter, agentID)
			log.Printf("Last SSE client disconnected from agent %s, stopping metrics streaming", agentID)
			go h.stopStreamingForAgent(agentID)
		}
	}
}

// ensureStreamingStarted starts streaming if it's not already active
func (h *SSEHandler) ensureStreamingStarted(agentID string, interval int) {
	h.counterMutex.RLock()
	clientCount := h.clientCounter[agentID]
	h.counterMutex.RUnlock()

	// Only start streaming if this is the first client
	if clientCount == 1 {
		log.Printf("First SSE client connected to agent %s, starting metrics streaming", agentID)
		go h.startStreamingForAgent(agentID, interval)
	}
}

// startStreamingForAgent starts metrics streaming for an agent
func (h *SSEHandler) startStreamingForAgent(agentID string, interval int) {
	task, err := h.taskManager.SendTask(&pb.TaskRequest{
		AgentId: agentID,
		Task: &pb.TaskRequest_MetricsRequest{
			MetricsRequest: &pb.MetricsRequest{
				RequestType: &pb.MetricsRequest_StreamRequest{
					StreamRequest: &pb.MetricsStreamRequest{
						Action:          pb.MetricsStreamRequest_START,
						IntervalSeconds: uint32(interval),
						Metrics:         []string{}, // empty means all metrics
					},
				},
			},
		},
	}, 5*time.Minute)

	if err != nil {
		log.Printf("Failed to start metrics streaming for agent %s: %v", agentID, err)
		return
	}

	// Update store status
	h.store.SetStreamingStatus(agentID, true)
	log.Printf("Started metrics streaming for agent %s (task: %s)", agentID, task.ID)
}

// stopStreamingForAgent stops metrics streaming for an agent
func (h *SSEHandler) stopStreamingForAgent(agentID string) {
	task, err := h.taskManager.SendTask(&pb.TaskRequest{
		AgentId: agentID,
		Task: &pb.TaskRequest_MetricsRequest{
			MetricsRequest: &pb.MetricsRequest{
				RequestType: &pb.MetricsRequest_StreamRequest{
					StreamRequest: &pb.MetricsStreamRequest{
						Action: pb.MetricsStreamRequest_STOP,
					},
				},
			},
		},
	}, 5*time.Minute)

	if err != nil {
		log.Printf("Failed to stop metrics streaming for agent %s: %v", agentID, err)
		return
	}

	// Update store status
	h.store.SetStreamingStatus(agentID, false)
	log.Printf("Stopped metrics streaming for agent %s (task: %s)", agentID, task.ID)
}

// BroadcastMetrics broadcasts metrics to all clients listening to the agent
func (h *SSEHandler) BroadcastMetrics(agentID string, metrics *pb.MetricsDataResponse) {
	roomName := fmt.Sprintf("metrics:%s", agentID)
	h.sseManager.SendToRoom(roomName, metrics, "metrics")
}

// ProcessMetricsResponse processes incoming metrics responses from agents
func (h *SSEHandler) ProcessMetricsResponse(agentID string, response *pb.MetricsResponse) {
	switch data := response.ResponseType.(type) {
	case *pb.MetricsResponse_SystemInfo:
		// Update system info in store
		h.store.UpdateSystemInfo(agentID, data.SystemInfo)
		log.Printf("Updated system info for agent %s", agentID)

	case *pb.MetricsResponse_MetricsData:
		// Update metrics in store
		h.store.UpdateMetrics(agentID, data.MetricsData)

		// Broadcast to SSE clients
		h.BroadcastMetrics(agentID, data.MetricsData)

	case *pb.MetricsResponse_QueryResponse:
		// Historical query responses are handled by direct API calls
		log.Printf("Received historical query response for agent %s with %d points",
			agentID, len(data.QueryResponse.DataPoints))
	}
}
