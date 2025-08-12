package tasks

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var AllowedAgents map[string]string = map[string]string{
	"agent1": "secret_token1",
	"agent2": "secret_token2",
}

type TaskServer struct {
	pb.UnimplementedAgentServiceServer

	// Agent connection management
	mu     sync.RWMutex
	agents map[string]pb.AgentService_StreamTasksServer

	// Task management
	tasksMu      sync.RWMutex
	tasks        map[string]*types.Task
	taskContexts map[string]context.CancelFunc // Track cancel functions separately

	// Dependencies
	metricsStore MetricsStore
	eventRouter  *EventRouter
}

// MetricsStore interface for registering agents
type MetricsStore interface {
	RegisterAgent(agentID string)
	UnregisterAgent(agentID string)
}

func NewTaskServer(metricsStore MetricsStore, eventRouter *EventRouter) *TaskServer {
	server := &TaskServer{
		agents:       make(map[string]pb.AgentService_StreamTasksServer),
		tasks:        make(map[string]*types.Task),
		taskContexts: make(map[string]context.CancelFunc),
		metricsStore: metricsStore,
		eventRouter:  eventRouter,
	}

	return server
}

func (s *TaskServer) StreamTasks(stream pb.AgentService_StreamTasksServer) error {
	// Authenticate agent
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return fmt.Errorf("missing metadata")
	}

	agentIDs := md["agent_id"]
	tokens := md["agent_token"]
	if len(agentIDs) == 0 || len(tokens) == 0 {
		return fmt.Errorf("missing agent credentials")
	}

	agentID := agentIDs[0]
	token := tokens[0]

	expectedToken, allowed := AllowedAgents[agentID]
	if !allowed || token != expectedToken {
		return fmt.Errorf("unauthorized agent")
	}

	s.mu.Lock()
	s.agents[agentID] = stream
	s.mu.Unlock()

	log.Printf("Agent %s connected", agentID)

	// Register agent in metrics store immediately
	if s.metricsStore != nil {
		s.metricsStore.RegisterAgent(agentID)
	}

	// Remove agent when done
	defer func() {
		s.mu.Lock()
		delete(s.agents, agentID)
		s.mu.Unlock()

		// Unregister agent from metrics store
		if s.metricsStore != nil {
			s.metricsStore.UnregisterAgent(agentID)
		}

		log.Printf("agent %s disconnected", agentID)
	}()

	// Listen for incoming task responses from agent
	for {
		task, err := stream.Recv()
		if err == io.EOF {
			log.Printf("Agent %s closed connection gracefully", agentID)
			break
		}
		if err != nil {
			// Check if it's a network error that might be recoverable
			if status.Code(err) == codes.Unavailable ||
				status.Code(err) == codes.DeadlineExceeded ||
				status.Code(err) == codes.Canceled {
				log.Printf("Agent %s connection lost (recoverable): %v", agentID, err)
			} else {
				log.Printf("Agent %s connection error (non-recoverable): %v", agentID, err)
			}
			return err
		}

		log.Printf("Received task from %s: %+v", agentID, task)

		// Process response directly instead of sending to channel
		s.handleTaskResponse(task)
	}

	return nil
}

func (s *TaskServer) Send(task *pb.TaskRequest) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// check if agent is connected
	stream, ok := s.agents[task.AgentId]
	if !ok {
		return fmt.Errorf("agent with this id is not connected: %s", task.AgentId)
	}

	// send task
	err := stream.Send(task)
	if err != nil {
		return fmt.Errorf("failed to send task to agent: %v", err)
	}

	return nil
}

// CreateTask creates and manages a new task
func (s *TaskServer) CreateTask(taskRequest *pb.TaskRequest, timeout time.Duration) (*types.Task, error) {
	task := &types.Task{
		ID:        uuid.New().String(),
		Request:   taskRequest,
		Status:    types.TaskStatusCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Timeout:   timeout,
	}

	// Set timeout context for the task
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	s.tasksMu.Lock()
	s.tasks[task.ID] = task
	s.taskContexts[task.ID] = cancel
	s.tasksMu.Unlock()

	// Try to find the agent and send the task
	s.mu.RLock()
	stream, exists := s.agents[taskRequest.AgentId]
	s.mu.RUnlock()

	if !exists {
		task.Status = types.TaskStatusFailed
		return task, fmt.Errorf("agent %s not found", taskRequest.AgentId)
	}

	// Send task to agent
	if err := stream.Send(taskRequest); err != nil {
		task.Status = types.TaskStatusFailed
		return task, fmt.Errorf("failed to send task to agent: %w", err)
	}

	task.Status = types.TaskStatusSent
	task.UpdatedAt = time.Now()

	// Start timeout monitoring
	go s.monitorTask(task.ID, ctx)

	log.Printf("Created and sent task %s to agent %s", task.ID, taskRequest.AgentId)
	return task, nil
}

// SendTask provides compatibility with TaskManager interface (alias for CreateTask)
func (s *TaskServer) SendTask(request *pb.TaskRequest, timeout time.Duration) (*types.Task, error) {
	return s.CreateTask(request, timeout)
}

// GetTask retrieves a task by ID
func (s *TaskServer) GetTask(taskID string) (*types.Task, bool) {
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()
	task, exists := s.tasks[taskID]
	return task, exists
}

// UpdateTaskStatus updates the status of a task
func (s *TaskServer) UpdateTaskStatus(taskID string, status types.TaskStatus) error {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()
	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}
	task.Status = status
	task.UpdatedAt = time.Now()
	return nil
}

// handleTaskResponse processes task responses from agents
func (s *TaskServer) handleTaskResponse(taskResponse *pb.TaskResponse) {
	s.tasksMu.Lock()
	task, exists := s.tasks[taskResponse.TaskId]

	if !exists {
		s.tasksMu.Unlock()
		// Only log as warning for non-final responses, as the task might have been cleaned up
		if taskResponse.IsFinal {
			log.Printf("Task %s not found for final response - may have been cleaned up", taskResponse.TaskId)
		} else {
			log.Printf("Task %s not found for progress response - task may have been completed", taskResponse.TaskId)
		}
		return
	}

	// Update task state
	task.Response = taskResponse
	isFinal := taskResponse.IsFinal
	isCancellation := taskResponse.Cancelled

	switch taskResponse.Status {
	case pb.TaskResponse_IN_PROGRESS:
		task.Status = types.TaskStatusInProgress

	case pb.TaskResponse_COMPLETED:
		if isCancellation && isFinal {
			task.Status = types.TaskStatusCancelled
		} else if isFinal {
			task.Status = types.TaskStatusCompleted
		} else {
			task.Status = types.TaskStatusInProgress
		}

	case pb.TaskResponse_FAILURE:
		if isCancellation && isFinal {
			task.Status = types.TaskStatusCancelled
		} else {
			task.Status = types.TaskStatusFailed
		}
	}

	task.UpdatedAt = time.Now()

	// For streaming tasks, identify them by checking if they have metrics_request with stream_request
	isStreamingTask := task.Request != nil &&
		task.Request.GetMetricsRequest() != nil &&
		task.Request.GetMetricsRequest().GetStreamRequest() != nil

	cleanupDelay := 5 * time.Second
	if isStreamingTask && !isFinal {
		cleanupDelay = 30 * time.Second // Longer delay for active streaming tasks
	}

	s.tasksMu.Unlock()

	// Process the event through the event router
	if task.Response != nil {
		s.eventRouter.ProcessAndRelay(task.Response)
	}

	// Cleanup if final - add appropriate delay based on task type
	if isFinal {
		go func() {
			time.Sleep(cleanupDelay)
			s.completeTask(taskResponse.TaskId)
		}()
	}
}

// CancelTask cancels a task and notifies the agent
func (s *TaskServer) CancelTask(taskID string) error {
	s.tasksMu.Lock()
	task, exists := s.tasks[taskID]
	cancelFunc, hasCancelFunc := s.taskContexts[taskID]
	if !exists {
		s.tasksMu.Unlock()
		return fmt.Errorf("task %s not found", taskID)
	}
	if task.Status == types.TaskStatusCompleted ||
		task.Status == types.TaskStatusFailed ||
		task.Status == types.TaskStatusCancelled {
		s.tasksMu.Unlock()
		return fmt.Errorf("task %s is already in final state: %s", taskID, task.Status.String())
	}
	if hasCancelFunc {
		cancelFunc()
	}
	task.Status = types.TaskStatusCancelled
	task.UpdatedAt = time.Now()
	agentID := task.Request.AgentId
	s.tasksMu.Unlock()

	// Send cancel request directly to agent
	cancelEvent := &pb.TaskRequest{
		AgentId: agentID,
		TaskId:  taskID,
		Task: &pb.TaskRequest_TaskCancel{
			TaskCancel: &pb.TaskCancel{
				Reason: "cancelled by user",
			},
		},
	}

	return s.Send(cancelEvent)
}

// ListTasks returns all tasks for an agent (or all if agentID is empty)
func (s *TaskServer) ListTasks(agentID string) []*types.Task {
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()
	var taskList []*types.Task
	for _, task := range s.tasks {
		if agentID == "" || task.Request.AgentId == agentID {
			taskList = append(taskList, task)
		}
	}
	return taskList
}

// CleanupCompletedTasks removes completed tasks older than the specified duration
func (s *TaskServer) CleanupCompletedTasks(olderThan time.Duration) int {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()
	cutoff := time.Now().Add(-olderThan)
	var removedCount int
	for taskID, task := range s.tasks {
		if (task.Status == types.TaskStatusCompleted ||
			task.Status == types.TaskStatusFailed ||
			task.Status == types.TaskStatusTimeout ||
			task.Status == types.TaskStatusCancelled) &&
			task.UpdatedAt.Before(cutoff) {
			if cancelFunc, exists := s.taskContexts[taskID]; exists {
				cancelFunc()
				delete(s.taskContexts, taskID)
			}
			delete(s.tasks, taskID)
			removedCount++
		}
	}
	return removedCount
}

// monitorTask monitors a task for timeout
func (s *TaskServer) monitorTask(taskID string, ctx context.Context) {
	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		s.UpdateTaskStatus(taskID, types.TaskStatusTimeout)
		s.completeTask(taskID)
	}
}

// completeTask removes a task from the manager
func (s *TaskServer) completeTask(taskID string) {
	s.tasksMu.Lock()
	delete(s.tasks, taskID)
	delete(s.taskContexts, taskID)
	s.tasksMu.Unlock()
}
