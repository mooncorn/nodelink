package metrics

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/mooncorn/nodelink/server/internal/proto"
	"github.com/mooncorn/nodelink/server/internal/common"
)

// Handler manages metrics requests to agents
type Handler struct {
	statusManager common.StatusManager
	streamSender  common.StreamSender
	mu            sync.RWMutex
	requests      map[string]*MetricsRequest
}

// MetricsRequest tracks a pending metrics request
type MetricsRequest struct {
	RequestID    string
	AgentID      string
	ResponseChan chan *MetricsResponse
	Timeout      time.Duration
	Created      time.Time
}

// MetricsResponse contains the response data
type MetricsResponse struct {
	SystemInfo    *pb.SystemInfo
	SystemMetrics *pb.SystemMetrics
	Error         string
}

// NewHandler creates a new metrics handler
func NewHandler(statusManager common.StatusManager) *Handler {
	return &Handler{
		statusManager: statusManager,
		requests:      make(map[string]*MetricsRequest),
	}
}

// SetStreamSender sets the stream sender for sending messages to agents
func (h *Handler) SetStreamSender(sender common.StreamSender) {
	h.streamSender = sender
}

// RequestSystemInfo requests system information from an agent
func (h *Handler) RequestSystemInfo(ctx context.Context, agentID string) (*pb.SystemInfo, error) {
	requestID := uuid.New().String()

	// Create request tracking
	responseChan := make(chan *MetricsResponse, 1)
	request := &MetricsRequest{
		RequestID:    requestID,
		AgentID:      agentID,
		ResponseChan: responseChan,
		Timeout:      30 * time.Second,
		Created:      time.Now(),
	}

	h.mu.Lock()
	h.requests[requestID] = request
	h.mu.Unlock()

	// Clean up when done
	defer func() {
		h.mu.Lock()
		delete(h.requests, requestID)
		h.mu.Unlock()
	}()

	// Send system info request to agent
	message := &pb.ServerMessage{
		Message: &pb.ServerMessage_SystemInfoRequest{
			SystemInfoRequest: &pb.SystemInfoRequest{
				RequestId: requestID,
			},
		},
	}

	err := h.streamSender.SendToAgent(agentID, message)
	if err != nil {
		return nil, fmt.Errorf("failed to send system info request to agent %s: %w", agentID, err)
	}

	// Wait for response
	select {
	case response := <-responseChan:
		if response.Error != "" {
			return nil, fmt.Errorf("agent error: %s", response.Error)
		}
		return response.SystemInfo, nil
	case <-time.After(request.Timeout):
		return nil, fmt.Errorf("timeout waiting for system info from agent %s", agentID)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// RequestMetrics requests current metrics from an agent
func (h *Handler) RequestMetrics(ctx context.Context, agentID string) (*pb.SystemMetrics, error) {
	requestID := uuid.New().String()

	// Create request tracking
	responseChan := make(chan *MetricsResponse, 1)
	request := &MetricsRequest{
		RequestID:    requestID,
		AgentID:      agentID,
		ResponseChan: responseChan,
		Timeout:      30 * time.Second,
		Created:      time.Now(),
	}

	h.mu.Lock()
	h.requests[requestID] = request
	h.mu.Unlock()

	// Clean up when done
	defer func() {
		h.mu.Lock()
		delete(h.requests, requestID)
		h.mu.Unlock()
	}()

	// Send metrics request to agent
	message := &pb.ServerMessage{
		Message: &pb.ServerMessage_MetricsRequest{
			MetricsRequest: &pb.MetricsRequest{
				RequestId: requestID,
			},
		},
	}

	err := h.streamSender.SendToAgent(agentID, message)
	if err != nil {
		return nil, fmt.Errorf("failed to send metrics request to agent %s: %w", agentID, err)
	}

	// Wait for response
	select {
	case response := <-responseChan:
		if response.Error != "" {
			return nil, fmt.Errorf("agent error: %s", response.Error)
		}
		return response.SystemMetrics, nil
	case <-time.After(request.Timeout):
		return nil, fmt.Errorf("timeout waiting for metrics from agent %s", agentID)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// HandleSystemInfoResponse handles system info responses from agents
func (h *Handler) HandleSystemInfoResponse(response *pb.SystemInfoResponse) {
	h.mu.RLock()
	request, exists := h.requests[response.RequestId]
	h.mu.RUnlock()

	if !exists {
		log.Printf("Received system info response for unknown request ID: %s", response.RequestId)
		return
	}

	metricsResponse := &MetricsResponse{
		SystemInfo: response.SystemInfo,
		Error:      response.Error,
	}

	select {
	case request.ResponseChan <- metricsResponse:
		// Response sent successfully
	default:
		// Channel full or closed, ignore
		log.Printf("Failed to send system info response for request %s", response.RequestId)
	}
}

// HandleMetricsResponse handles metrics responses from agents
func (h *Handler) HandleMetricsResponse(response *pb.MetricsResponse) {
	h.mu.RLock()
	request, exists := h.requests[response.RequestId]
	h.mu.RUnlock()

	if !exists {
		log.Printf("Received metrics response for unknown request ID: %s", response.RequestId)
		return
	}

	metricsResponse := &MetricsResponse{
		SystemMetrics: response.Metrics,
		Error:         response.Error,
	}

	select {
	case request.ResponseChan <- metricsResponse:
		// Response sent successfully
	default:
		// Channel full or closed, ignore
		log.Printf("Failed to send metrics response for request %s", response.RequestId)
	}
}

// CleanupExpiredRequests removes expired requests
func (h *Handler) CleanupExpiredRequests() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	for requestID, request := range h.requests {
		if now.Sub(request.Created) > request.Timeout {
			close(request.ResponseChan)
			delete(h.requests, requestID)
		}
	}
}

// Start starts the cleanup routine
func (h *Handler) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.CleanupExpiredRequests()
		case <-ctx.Done():
			return
		}
	}
}
