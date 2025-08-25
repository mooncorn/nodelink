package metrics

import (
	"log"

	pb "github.com/mooncorn/nodelink/agent/internal/proto"
)

// MessageSender interface for sending messages to the server
type MessageSender interface {
	Send(msg *pb.AgentMessage) error
}

// Handler handles metrics requests from the server
type Handler struct {
	collector     *Collector
	messageSender MessageSender
}

// NewHandler creates a new metrics handler
func NewHandler() *Handler {
	return &Handler{
		collector: NewCollector(),
	}
}

// SetMessageSender sets the message sender for the handler
func (h *Handler) SetMessageSender(sender MessageSender) {
	h.messageSender = sender
}

// HandleSystemInfoRequest handles system information requests
func (h *Handler) HandleSystemInfoRequest(request *pb.SystemInfoRequest) {
	log.Printf("Handling system info request: %s", request.RequestId)

	systemInfo, err := h.collector.GetSystemInfo()

	var response *pb.SystemInfoResponse
	if err != nil {
		log.Printf("Error collecting system info: %v", err)
		response = &pb.SystemInfoResponse{
			RequestId: request.RequestId,
			Error:     err.Error(),
		}
	} else {
		response = &pb.SystemInfoResponse{
			RequestId:  request.RequestId,
			SystemInfo: systemInfo,
		}
	}

	// Send response back to server
	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_SystemInfoResponse{
			SystemInfoResponse: response,
		},
	}

	if h.messageSender != nil {
		if err := h.messageSender.Send(agentMsg); err != nil {
			log.Printf("Error sending system info response: %v", err)
		}
	} else {
		log.Printf("Warning: no message sender set for metrics handler")
	}
}

// HandleMetricsRequest handles metrics requests
func (h *Handler) HandleMetricsRequest(request *pb.MetricsRequest) {
	log.Printf("Handling metrics request: %s", request.RequestId)

	metrics, err := h.collector.GetSystemMetrics()

	var response *pb.MetricsResponse
	if err != nil {
		log.Printf("Error collecting system metrics: %v", err)
		response = &pb.MetricsResponse{
			RequestId: request.RequestId,
			Error:     err.Error(),
		}
	} else {
		response = &pb.MetricsResponse{
			RequestId: request.RequestId,
			Metrics:   metrics,
		}
	}

	// Send response back to server
	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_MetricsResponse{
			MetricsResponse: response,
		},
	}

	if h.messageSender != nil {
		if err := h.messageSender.Send(agentMsg); err != nil {
			log.Printf("Error sending metrics response: %v", err)
		}
	} else {
		log.Printf("Warning: no message sender set for metrics handler")
	}
}
