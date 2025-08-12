package command

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/agent"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CommandServer struct {
	pb.UnimplementedAgentServiceServer
	agentManager *agent.Manager

	// Track active streaming commands
	mu                sync.RWMutex
	streamingCommands map[string]context.CancelFunc
}

func NewCommandServer(agentManager *agent.Manager) *CommandServer {
	return &CommandServer{
		agentManager:      agentManager,
		streamingCommands: make(map[string]context.CancelFunc),
	}
}

// ExecuteCommand handles simple command execution with immediate response
func (s *CommandServer) ExecuteCommand(ctx context.Context, req *pb.CommandRequest) (*pb.CommandResponse, error) {
	if req.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "metadata is required")
	}

	agentID := req.Metadata.AgentId
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	// Get agent connection directly
	conn, exists := s.agentManager.GetAgentConnection(agentID)
	if !exists {
		return nil, status.Errorf(codes.NotFound, "agent %s is not connected", agentID)
	}

	// Set timeout
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second // default timeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Generate request ID if not provided
	if req.Metadata.RequestId == "" {
		req.Metadata.RequestId = uuid.New().String()
	}
	req.Metadata.Timestamp = time.Now().UnixMilli()

	// Create task assignment for the command
	taskAssignment := &pb.TaskAssignment{
		TaskId:   req.Metadata.RequestId,
		TaskType: "command",
		Priority: 1,
		// TODO: Serialize the command request as payload
	}

	// Send command to agent via the connection stream
	message := &pb.AgentMessage{
		AgentId:   agentID,
		MessageId: req.Metadata.RequestId,
		Timestamp: time.Now().Unix(),
		Payload: &pb.AgentMessage_TaskAssignment{
			TaskAssignment: taskAssignment,
		},
	}

	if err := conn.Stream.Send(message); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send command to agent: %v", err)
	}

	// TODO: Wait for response from agent
	// This would require setting up a response channel and handling the response
	// in the ManageAgent stream handler

	return nil, status.Error(codes.Unimplemented, "command execution response handling not yet implemented")
}

// StreamCommand handles long-running command execution with real-time output
func (s *CommandServer) StreamCommand(req *pb.CommandRequest, stream pb.AgentService_StreamCommandServer) error {
	if req.Metadata == nil {
		return status.Error(codes.InvalidArgument, "metadata is required")
	}

	agentID := req.Metadata.AgentId
	if agentID == "" {
		return status.Error(codes.InvalidArgument, "agent_id is required")
	}

	// Get agent connection directly
	conn, exists := s.agentManager.GetAgentConnection(agentID)
	if !exists {
		return status.Errorf(codes.NotFound, "agent %s is not connected", agentID)
	}

	// Generate request ID if not provided
	if req.Metadata.RequestId == "" {
		req.Metadata.RequestId = uuid.New().String()
	}
	req.Metadata.Timestamp = time.Now().UnixMilli()

	// Set timeout
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute // default timeout for streaming
	}

	_, cancel := context.WithTimeout(stream.Context(), timeout)
	defer cancel()

	// Track this streaming command for cancellation
	s.mu.Lock()
	s.streamingCommands[req.Metadata.RequestId] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.streamingCommands, req.Metadata.RequestId)
		s.mu.Unlock()
	}()

	// Create task assignment for streaming command
	taskAssignment := &pb.TaskAssignment{
		TaskId:   req.Metadata.RequestId,
		TaskType: "command_stream",
		Priority: 1,
		// TODO: Serialize the command request as payload
	}

	// Send streaming command to agent
	message := &pb.AgentMessage{
		AgentId:   agentID,
		MessageId: req.Metadata.RequestId,
		Timestamp: time.Now().Unix(),
		Payload: &pb.AgentMessage_TaskAssignment{
			TaskAssignment: taskAssignment,
		},
	}

	if err := conn.Stream.Send(message); err != nil {
		return status.Errorf(codes.Internal, "failed to send streaming command to agent: %v", err)
	}

	// TODO: Handle streaming responses from agent
	// This would require setting up a response channel and handling streaming responses
	// from the ManageAgent stream handler

	return status.Error(codes.Unimplemented, "streaming command response handling not yet implemented")
}

// CancelTask cancels a running command by request ID
func (s *CommandServer) CancelTask(ctx context.Context, req *pb.TaskCancelRequest) (*pb.TaskCancelResponse, error) {
	if req.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "metadata is required")
	}

	requestID := req.TargetRequestId
	if requestID == "" {
		return nil, status.Error(codes.InvalidArgument, "target_request_id is required")
	}

	s.mu.RLock()
	cancelFunc, exists := s.streamingCommands[requestID]
	s.mu.RUnlock()

	if !exists {
		return &pb.TaskCancelResponse{
			RequestId: req.Metadata.RequestId,
			Success:   false,
			Message:   "command not found or already completed",
		}, nil
	}

	// Cancel the command
	cancelFunc()

	return &pb.TaskCancelResponse{
		RequestId: req.Metadata.RequestId,
		Success:   true,
		Message:   "command cancelled successfully",
	}, nil
}
