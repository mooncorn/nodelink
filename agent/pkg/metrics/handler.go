package metrics

import (
	"log"
	"time"

	"github.com/mooncorn/nodelink/agent/pkg/grpc"
	pb "github.com/mooncorn/nodelink/proto"
)

// Handler handles metrics-related task requests
type Handler struct {
	collector *MetricsCollector
	client    *grpc.TaskClient
}

// NewHandler creates a new metrics handler
func NewHandler(client *grpc.TaskClient) *Handler {
	collector := NewMetricsCollector(func(response *pb.TaskResponse) {
		if err := client.Send(response); err != nil {
			// Only log non-EOF errors as errors, EOF is normal connection closure
			if err.Error() == "EOF" {
				log.Printf("Connection closed while sending metrics response")
			} else {
				log.Printf("Failed to send metrics response: %v", err)
			}
		}
	})

	return &Handler{
		collector: collector,
		client:    client,
	}
}

// HandleMetricsRequest handles incoming metrics requests
func (h *Handler) HandleMetricsRequest(taskRequest *pb.TaskRequest, metricsRequest *pb.MetricsRequest) {
	switch req := metricsRequest.RequestType.(type) {
	case *pb.MetricsRequest_SystemInfo:
		h.handleSystemInfoRequest(taskRequest, req.SystemInfo)
	case *pb.MetricsRequest_StreamRequest:
		h.handleStreamRequest(taskRequest, req.StreamRequest)
	case *pb.MetricsRequest_QueryRequest:
		h.handleQueryRequest(taskRequest, req.QueryRequest)
	default:
		h.sendErrorResponse(taskRequest, "Unknown metrics request type")
	}
}

// handleSystemInfoRequest handles system information requests
func (h *Handler) handleSystemInfoRequest(taskRequest *pb.TaskRequest, sysInfoReq *pb.SystemInfoRequest) {
	log.Printf("Collecting system information for task %s", taskRequest.TaskId)

	systemInfo := h.collector.GetSystemInfo()

	response := &pb.TaskResponse{
		AgentId:   taskRequest.AgentId,
		TaskId:    taskRequest.TaskId,
		Status:    pb.TaskResponse_COMPLETED,
		IsFinal:   true,
		Cancelled: false,
		Response: &pb.TaskResponse_MetricsResponse{
			MetricsResponse: &pb.MetricsResponse{
				ResponseType: &pb.MetricsResponse_SystemInfo{
					SystemInfo: systemInfo,
				},
			},
		},
	}

	if err := h.client.Send(response); err != nil {
		log.Printf("Failed to send system info response: %v", err)
	}
}

// handleStreamRequest handles metrics streaming requests
func (h *Handler) handleStreamRequest(taskRequest *pb.TaskRequest, streamReq *pb.MetricsStreamRequest) {
	switch streamReq.Action {
	case pb.MetricsStreamRequest_START:
		log.Printf("Starting metrics streaming for task %s with interval %ds",
			taskRequest.TaskId, streamReq.IntervalSeconds)

		interval := 5 // default 5 seconds
		if streamReq.IntervalSeconds > 0 {
			interval = int(streamReq.IntervalSeconds)
		}

		h.collector.StartStreaming(taskRequest.AgentId, taskRequest.TaskId, time.Duration(interval)*time.Second)

		// Send acknowledgment
		h.sendStreamResponse(taskRequest, "Metrics streaming started")

	case pb.MetricsStreamRequest_STOP:
		log.Printf("Stopping metrics streaming for task %s", taskRequest.TaskId)

		h.collector.StopStreaming()

		// Send acknowledgment
		h.sendStreamResponse(taskRequest, "Metrics streaming stopped")

	case pb.MetricsStreamRequest_UPDATE_INTERVAL:
		log.Printf("Updating metrics interval for task %s to %ds",
			taskRequest.TaskId, streamReq.IntervalSeconds)

		if streamReq.IntervalSeconds > 0 {
			interval := time.Duration(streamReq.IntervalSeconds) * time.Second
			h.collector.UpdateInterval(interval)
		}

		// Send acknowledgment
		h.sendStreamResponse(taskRequest, "Metrics interval updated")

	default:
		h.sendErrorResponse(taskRequest, "Unknown stream action")
	}
}

// handleQueryRequest handles historical metrics query requests
func (h *Handler) handleQueryRequest(taskRequest *pb.TaskRequest, queryReq *pb.MetricsQueryRequest) {
	log.Printf("Historical metrics query not yet implemented for task %s", taskRequest.TaskId)

	// For now, return empty query response
	// In a full implementation, this would query stored historical data
	response := &pb.TaskResponse{
		AgentId:   taskRequest.AgentId,
		TaskId:    taskRequest.TaskId,
		Status:    pb.TaskResponse_COMPLETED,
		IsFinal:   true,
		Cancelled: false,
		Response: &pb.TaskResponse_MetricsResponse{
			MetricsResponse: &pb.MetricsResponse{
				ResponseType: &pb.MetricsResponse_QueryResponse{
					QueryResponse: &pb.MetricsQueryResponse{
						DataPoints:          []*pb.MetricsTimePoint{},
						QueryStartTimestamp: queryReq.StartTimestamp,
						QueryEndTimestamp:   queryReq.EndTimestamp,
						TotalPoints:         0,
						Truncated:           false,
					},
				},
			},
		},
	}

	if err := h.client.Send(response); err != nil {
		log.Printf("Failed to send query response: %v", err)
	}
}

// sendStreamResponse sends a stream control response
func (h *Handler) sendStreamResponse(taskRequest *pb.TaskRequest, message string) {
	// For "started" and "updated" messages, keep task alive; for "stopped", mark as complete
	isStopMessage := message == "Metrics streaming stopped"

	status := pb.TaskResponse_IN_PROGRESS
	if isStopMessage {
		status = pb.TaskResponse_COMPLETED
	}

	// Send a simple task response without any metrics payload to avoid interfering with system info
	response := &pb.TaskResponse{
		AgentId:   taskRequest.AgentId,
		TaskId:    taskRequest.TaskId,
		Status:    status,
		IsFinal:   isStopMessage,
		Cancelled: false,
		// No response payload - this is just a control message
	}

	if err := h.client.Send(response); err != nil {
		log.Printf("Failed to send stream response: %v", err)
	}
} // sendErrorResponse sends an error response
func (h *Handler) sendErrorResponse(taskRequest *pb.TaskRequest, errorMsg string) {
	response := &pb.TaskResponse{
		AgentId:   taskRequest.AgentId,
		TaskId:    taskRequest.TaskId,
		Status:    pb.TaskResponse_FAILURE,
		IsFinal:   true,
		Cancelled: false,
		Response: &pb.TaskResponse_MetricsResponse{
			MetricsResponse: &pb.MetricsResponse{
				ResponseType: &pb.MetricsResponse_SystemInfo{
					SystemInfo: &pb.SystemInfoResponse{
						Timestamp: uint64(time.Now().Unix()),
						Software: &pb.SystemSoftware{
							Hostname: errorMsg,
						},
					},
				},
			},
		},
	}

	if err := h.client.Send(response); err != nil {
		log.Printf("Failed to send error response: %v", err)
	}
}

// GetCollector returns the metrics collector (for tracking processes)
func (h *Handler) GetCollector() *MetricsCollector {
	return h.collector
}

// Close stops any ongoing metrics collection
func (h *Handler) Close() {
	h.collector.StopStreaming()
}
