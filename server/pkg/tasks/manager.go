package tasks

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/pkg/grpc"
)

// TaskManager manages the lifecycle of tasks
type TaskManager struct {
	mu         sync.RWMutex
	tasks      map[string]*Task
	listeners  []TaskEventListener
	taskServer *grpc.TaskServer
	respCh     chan *pb.TaskResponse
	stopCh     chan struct{}
}

// NewTaskManager creates a new task manager
func NewTaskManager() *TaskManager {
	tm := &TaskManager{
		tasks:     make(map[string]*Task),
		listeners: make([]TaskEventListener, 0),
		respCh:    make(chan *pb.TaskResponse, 100),
		stopCh:    make(chan struct{}),
	}
	go tm.responseLoop()
	return tm
}

// SetTaskServer sets the task server for sending tasks to agents
func (tm *TaskManager) SetTaskServer(taskServer *grpc.TaskServer) {
	tm.taskServer = taskServer
}

// GetResponseChannel returns the channel for receiving task responses
func (tm *TaskManager) GetResponseChannel() chan<- *pb.TaskResponse {
	return tm.respCh
}

// AddListener adds a task event listener
func (tm *TaskManager) AddListener(listener TaskEventListener) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.listeners = append(tm.listeners, listener)
}

// SendTask sends the task to the agent
func (tm *TaskManager) SendTask(taskRequest *pb.TaskRequest, timeout time.Duration) (*Task, error) {
	if taskRequest == nil {
		return nil, fmt.Errorf("task cannot be nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	task := &Task{
		ID:        uuid.NewString(),
		Status:    TaskStatusCreated,
		Request:   taskRequest,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Timeout:   timeout,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Set the generated task ID in the request
	taskRequest.TaskId = task.ID

	tm.mu.Lock()
	tm.tasks[task.ID] = task
	tm.mu.Unlock()

	// Send to agent
	err := tm.taskServer.Send(taskRequest)
	if err != nil {
		tm.UpdateTaskStatus(task.ID, TaskStatusFailed)
		return nil, fmt.Errorf("failed to send task to agent: %s", err.Error())
	}

	tm.UpdateTaskStatus(task.ID, TaskStatusSent)
	go tm.monitorTask(task)

	log.Printf("Created task %s for agent %s", task.ID, taskRequest.AgentId)
	return task, nil
}

// GetTask retrieves a task by ID
func (tm *TaskManager) GetTask(taskID string) (*Task, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	task, exists := tm.tasks[taskID]
	return task, exists
}

// UpdateTaskStatus updates the status of a task
func (tm *TaskManager) UpdateTaskStatus(taskID string, status TaskStatus) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}
	task.Status = status
	task.UpdatedAt = time.Now()
	return nil
}

// EnqueueTaskResponse is called by the gRPC server to enqueue a response for processing
func (tm *TaskManager) EnqueueTaskResponse(resp *pb.TaskResponse) {
	select {
	case tm.respCh <- resp:
	default:
		log.Printf("TaskManager response channel full, dropping response for task %s", resp.TaskId)
	}
}

// responseLoop processes incoming TaskResponses from the channel
func (tm *TaskManager) responseLoop() {
	for {
		select {
		case resp := <-tm.respCh:
			tm.processTaskResponse(resp)
		case <-tm.stopCh:
			return
		}
	}
}

// processTaskResponse updates the task and notifies listeners
func (tm *TaskManager) processTaskResponse(taskResponse *pb.TaskResponse) {
	tm.mu.Lock()
	task, exists := tm.tasks[taskResponse.TaskId]

	if !exists {
		tm.mu.Unlock()
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
		task.Status = TaskStatusInProgress

	case pb.TaskResponse_COMPLETED:
		if isCancellation && isFinal {
			task.Status = TaskStatusCancelled
		} else if isFinal {
			task.Status = TaskStatusCompleted
		} else {
			task.Status = TaskStatusInProgress
		}

	case pb.TaskResponse_FAILURE:
		if isCancellation && isFinal {
			task.Status = TaskStatusCancelled
		} else {
			task.Status = TaskStatusFailed
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

	tm.mu.Unlock()

	tm.notifyListeners(task)

	// Cleanup if final - add appropriate delay based on task type
	if isFinal {
		go func() {
			time.Sleep(cleanupDelay)
			tm.completeTask(taskResponse.TaskId)
		}()
	}
}

// CancelTask cancels a task and notifies the agent
func (tm *TaskManager) CancelTask(taskID string) error {
	tm.mu.Lock()
	task, exists := tm.tasks[taskID]
	if !exists {
		tm.mu.Unlock()
		return fmt.Errorf("task %s not found", taskID)
	}
	if task.Status == TaskStatusCompleted ||
		task.Status == TaskStatusFailed ||
		task.Status == TaskStatusCancelled {
		tm.mu.Unlock()
		return fmt.Errorf("task %s is already in final state: %s", taskID, task.Status.String())
	}
	task.cancel()
	task.Status = TaskStatusCancelled
	task.UpdatedAt = time.Now()
	agentID := task.Request.AgentId
	tm.mu.Unlock()
	// Send cancel event to agent
	if tm.taskServer != nil {
		cancelEvent := &pb.TaskRequest{
			AgentId: agentID,
			TaskId:  taskID,
			Task: &pb.TaskRequest_TaskCancel{
				TaskCancel: &pb.TaskCancel{
					Reason: "cancelled by user",
				},
			},
		}
		_ = tm.taskServer.Send(cancelEvent)
	}
	return nil
}

// ListTasks returns all tasks for an agent (or all if agentID is empty)
func (tm *TaskManager) ListTasks(agentID string) []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	var tasks []*Task
	for _, task := range tm.tasks {
		if agentID == "" || task.Request.AgentId == agentID {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// CleanupCompletedTasks removes completed tasks older than the specified duration
func (tm *TaskManager) CleanupCompletedTasks(olderThan time.Duration) int {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	cutoff := time.Now().Add(-olderThan)
	var removedCount int
	for taskID, task := range tm.tasks {
		if (task.Status == TaskStatusCompleted ||
			task.Status == TaskStatusFailed ||
			task.Status == TaskStatusTimeout ||
			task.Status == TaskStatusCancelled) &&
			task.UpdatedAt.Before(cutoff) {
			task.cancel()
			delete(tm.tasks, taskID)
			removedCount++
		}
	}
	return removedCount
}

// monitorTask monitors a task for timeout
func (tm *TaskManager) monitorTask(task *Task) {
	<-task.ctx.Done()
	if task.ctx.Err() == context.DeadlineExceeded {
		tm.UpdateTaskStatus(task.ID, TaskStatusTimeout)
		tm.completeTask(task.ID)
	}
}

// completeTask removes a task from the manager
func (tm *TaskManager) completeTask(taskID string) {
	tm.mu.Lock()
	delete(tm.tasks, taskID)
	tm.mu.Unlock()
}

// notifyListeners notifies all task event listeners
func (tm *TaskManager) notifyListeners(task *Task) {
	tm.mu.RLock()
	listeners := make([]TaskEventListener, len(tm.listeners))
	copy(listeners, tm.listeners)
	tm.mu.RUnlock()
	for _, listener := range listeners {
		go listener(task)
	}
}
