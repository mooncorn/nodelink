package docker

import (
	"fmt"

	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/interfaces"
)

// DockerProcessor handles Docker-related events (placeholder for future implementation)
type DockerProcessor struct{}

// NewDockerProcessor creates a new Docker processor
func NewDockerProcessor() *DockerProcessor {
	return &DockerProcessor{}
}

// ProcessEvent processes Docker events
func (dp *DockerProcessor) ProcessEvent(event *pb.TaskResponse) (*interfaces.ProcessedEvent, error) {
	dockerResp := event.GetDockerOperation()
	if dockerResp == nil {
		return nil, fmt.Errorf("not a docker response")
	}

	// Process docker operation
	processedData := dp.processDockerOperation(dockerResp)

	// Determine target room based on operation type
	targetRoom := fmt.Sprintf("docker_operations_%s", event.AgentId)
	if dockerResp.GetOperation() == "logs" {
		targetRoom = fmt.Sprintf("container_logs_%s", dockerResp.GetContainerId())
	}

	return &interfaces.ProcessedEvent{
		OriginalEvent: event,
		ProcessedData: processedData,
		ShouldRelay:   true,
		TargetRoom:    targetRoom,
		EventType:     "docker_operation",
	}, nil
}

// GetEventType returns the event type this processor handles
func (dp *DockerProcessor) GetEventType() string {
	return "docker_operation"
}

// processDockerOperation processes docker operation data
func (dp *DockerProcessor) processDockerOperation(dockerResp *pb.DockerOperationResponse) *ProcessedDockerOperation {
	processed := &ProcessedDockerOperation{
		Operation:   dockerResp.GetOperation(),
		ContainerID: dockerResp.GetContainerId(),
		Status:      dockerResp.GetStatus(),
		Message:     dockerResp.GetMessage(),
	}

	// Handle specific operation data
	if runResult := dockerResp.GetRunResult(); runResult != nil {
		processed.RunResult = &DockerRunInfo{
			ContainerID: runResult.GetContainerId(),
			Image:       runResult.GetImage(),
			Ports:       runResult.GetPorts(),
			Status:      runResult.GetStatus(),
		}
	}

	if logsChunk := dockerResp.GetLogsChunk(); logsChunk != nil {
		processed.LogsChunk = &DockerLogsInfo{
			ContainerID: logsChunk.GetContainerId(),
			LogLine:     logsChunk.GetLogLine(),
			Stream:      logsChunk.GetStream(),
			Timestamp:   logsChunk.GetTimestamp(),
		}
	}

	return processed
}

// ProcessedDockerOperation represents processed docker operation data
type ProcessedDockerOperation struct {
	Operation   string          `json:"operation"`
	ContainerID string          `json:"container_id"`
	Status      string          `json:"status"`
	Message     string          `json:"message"`
	RunResult   *DockerRunInfo  `json:"run_result,omitempty"`
	LogsChunk   *DockerLogsInfo `json:"logs_chunk,omitempty"`
}

// DockerRunInfo represents Docker run operation result
type DockerRunInfo struct {
	ContainerID string   `json:"container_id"`
	Image       string   `json:"image"`
	Ports       []string `json:"ports"`
	Status      string   `json:"status"`
}

// DockerLogsInfo represents Docker logs chunk
type DockerLogsInfo struct {
	ContainerID string `json:"container_id"`
	LogLine     string `json:"log_line"`
	Stream      string `json:"stream"`
	Timestamp   int64  `json:"timestamp"`
}
