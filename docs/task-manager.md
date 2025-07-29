# Task Manager Component

This document describes the Task Manager component, which serves as the central orchestrator for task lifecycle management in the nodelink project.

## Overview

The Task Manager is responsible for:
- Creating and tracking tasks with unique IDs
- Sending tasks to appropriate agents via gRPC
- Processing task responses from agents
- Managing task lifecycle and status transitions
- Providing real-time notifications to clients
- Automatic cleanup of completed tasks

## Architecture

```go
type TaskManager struct {
    mu         sync.RWMutex          // Protects task registry
    tasks      map[string]*Task      // Active task registry  
    listeners  []TaskEventListener   // Event notification callbacks
    taskServer *grpc.TaskServer      // gRPC server for agent communication
    respCh     chan *pb.TaskResponse // Response processing channel
    stopCh     chan struct{}         // Shutdown signal channel
}
```

## Core Components

### 1. Task Registry

The task registry is an in-memory map that stores active tasks by their unique IDs:

```go
tasks map[string]*Task
```

**Features:**
- Thread-safe access with RWMutex
- UUID-based task identification
- Automatic cleanup of completed tasks
- Support for concurrent access

**Task Structure:**
```go
type Task struct {
    ID        string              // Unique task identifier
    Status    TaskStatus          // Current task status
    Request   *pb.TaskRequest     // Original request
    Response  *pb.TaskResponse    // Latest response from agent
    CreatedAt time.Time          // Task creation timestamp
    UpdatedAt time.Time          // Last status update
    Timeout   time.Duration      // Task timeout duration
    ctx       context.Context    // Cancellation context
    cancel    context.CancelFunc // Cancellation function
}
```

### 2. Response Processing Pipeline

The Task Manager uses a dedicated goroutine to process incoming responses:

```go
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
```

**Response Channel Features:**
- Bounded channel (100 entries) to prevent memory leaks
- Non-blocking enqueue with overflow protection
- Dedicated processing goroutine for serialized updates

### 3. Task Lifecycle Management

#### Task Creation and Execution
```go
func (tm *TaskManager) SendTask(taskRequest *pb.TaskRequest, timeout time.Duration) (*Task, error)
```

**Process:**
1. Generate unique UUID for task
2. Create task with timeout context
3. Store in registry with pending status
4. Send to agent via gRPC
5. Update status to sent
6. Start timeout monitoring

#### Status Transitions
```go
const (
    TaskStatusCreated     TaskStatus = "created"
    TaskStatusSent        TaskStatus = "sent"
    TaskStatusInProgress  TaskStatus = "in_progress"
    TaskStatusCompleted   TaskStatus = "completed"
    TaskStatusFailed      TaskStatus = "failed"
    TaskStatusTimeout     TaskStatus = "timeout"
    TaskStatusCancelled   TaskStatus = "cancelled"
)
```

**Transition Flow:**
```
created → sent → in_progress → completed/failed
    ↓       ↓         ↓
timeout ← timeout ← timeout
    ↓       ↓         ↓
cancelled ← cancelled ← cancelled
```

### 4. Event Notification System

#### Listener Interface
```go
type TaskEventListener func(task *Task)
```

#### Notification Process
```go
func (tm *TaskManager) notifyListeners(task *Task) {
    // Copy listeners under read lock
    tm.mu.RLock()
    listeners := make([]TaskEventListener, len(tm.listeners))
    copy(listeners, tm.listeners)
    tm.mu.RUnlock()
    
    // Notify in separate goroutines
    for _, listener := range listeners {
        go listener(task)
    }
}
```

**Features:**
- Asynchronous notification to prevent blocking
- Safe concurrent access to listener list
- Support for multiple listeners per task event

## Key Operations

### 1. Task Creation
```go
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
```

### 2. Response Processing
```go
func (tm *TaskManager) processTaskResponse(taskResponse *pb.TaskResponse) {
    // Update task state based on response
    // Handle cancellation flags
    // Notify listeners
    // Schedule cleanup for final responses
}
```

**Response Handling Logic:**
- Checks `taskResponse.Cancelled` for cancellation detection
- Updates status based on `pb.TaskResponse_Status`
- Handles both incremental and final responses
- Triggers cleanup with 5-second delay for final responses

### 3. Task Cancellation
```go
func (tm *TaskManager) CancelTask(taskID string) error {
    // Validate task exists and is cancellable
    // Update status to cancelled
    // Cancel context
    // Send cancel request to agent
}
```

### 4. Automatic Cleanup
```go
func (tm *TaskManager) CleanupCompletedTasks(olderThan time.Duration) int {
    // Remove tasks in final states older than cutoff
    // Cancel their contexts
    // Return count of removed tasks
}
```

## Concurrency and Thread Safety

### Locking Strategy
- **RWMutex**: Allows concurrent reads, exclusive writes
- **Read Operations**: `GetTask`, `ListTasks` use read locks
- **Write Operations**: `SendTask`, `UpdateTaskStatus` use write locks
- **Response Processing**: Single goroutine eliminates race conditions

### Race Condition Prevention
1. **Task Registration**: Tasks added to registry before sending to agent
2. **Status Updates**: Atomic updates with proper locking
3. **Cleanup Timing**: Delayed cleanup prevents premature removal
4. **Response Correlation**: Task ID ensures proper response routing

## Memory Management

### Bounded Resources
- **Response Channel**: Limited to 100 entries
- **Task Registry**: Automatic cleanup of old tasks
- **Listener Notifications**: Asynchronous to prevent accumulation

### Cleanup Strategies
1. **Immediate Cleanup**: Remove from registry after final response
2. **Delayed Cleanup**: 5-second delay for cancellation requests
3. **Periodic Cleanup**: Background cleanup of old completed tasks
4. **Context Cancellation**: Proper cleanup of timeout contexts

## Error Handling

### Task Creation Errors
- **Agent Unavailable**: Mark task as failed, return error
- **Invalid Request**: Validate before creating task
- **Registry Full**: Currently unlimited, future enhancement

### Response Processing Errors
- **Unknown Task ID**: Log warning, ignore response
- **Malformed Response**: Log error, continue processing
- **Channel Overflow**: Drop response with warning

### Agent Communication Errors
- **Send Failure**: Mark task as failed
- **Connection Loss**: Detect via gRPC stream errors
- **Timeout**: Automatic status update via context cancellation

## Performance Characteristics

### Scalability
- **O(1) Task Lookup**: Hash map provides constant time access
- **Concurrent Processing**: Multiple goroutines for I/O operations
- **Memory Efficient**: Automatic cleanup prevents unbounded growth

### Latency
- **Task Creation**: Sub-millisecond for in-memory operations
- **Status Updates**: Immediate via dedicated response loop
- **Notifications**: Asynchronous to prevent blocking

### Throughput
- **Bounded Channels**: Prevent memory exhaustion
- **Lock-Free Reads**: Multiple concurrent status checks
- **Batch Operations**: Support for listing multiple tasks

## Configuration

### Default Settings
```go
const (
    DefaultResponseChannelSize = 100
    DefaultCleanupDelay       = 5 * time.Second
    DefaultCleanupAge         = 30 * time.Minute
    DefaultCleanupInterval    = 5 * time.Minute
)
```

### Tuning Parameters
- **Response Channel Size**: Adjust based on expected load
- **Cleanup Delay**: Balance between resource usage and cancellation window
- **Cleanup Age**: Retain completed tasks for debugging/auditing
- **Cleanup Interval**: Frequency of background cleanup

## Integration Points

### gRPC Task Server
```go
func (tm *TaskManager) SetTaskServer(taskServer *grpc.TaskServer) {
    tm.taskServer = taskServer
}
```

### Event Stream Integration
- **SSE Handler**: Registers as task event listener
- **Real-time Updates**: Pushes task status changes to connected clients
- **Room Management**: Groups clients by task ID for targeted updates

### HTTP API Integration
- **Task Creation**: Called by `/tasks/shell` and `/tasks/docker` endpoints
- **Status Queries**: Powers `/tasks/{id}` endpoint
- **Task Listing**: Supports `/tasks` and `/agents/{id}/tasks` endpoints

## Monitoring and Observability

### Logging
```go
log.Printf("Created task %s for agent %s", task.ID, taskRequest.AgentId)
log.Printf("Task %s not found for response", resp.TaskId)
log.Printf("TaskManager response channel full, dropping response for task %s", resp.TaskId)
```

### Metrics (Future Enhancement)
- Task creation rate
- Task completion rate
- Average task duration
- Response channel utilization
- Cleanup frequency

## Testing Strategy

### Unit Tests
- Task creation and lifecycle
- Response processing logic
- Concurrency and race conditions
- Error handling scenarios

### Integration Tests
- Full request/response cycle
- Agent communication
- SSE notification delivery
- Cleanup and memory management

### Load Tests
- Concurrent task creation
- High-frequency status updates
- Memory usage under load
- Performance degradation analysis

## Future Enhancements

### Persistence
- Optional database storage for task history
- Recovery from server restarts
- Audit trail for completed tasks

### Advanced Features
- Task dependencies and workflows
- Scheduled task execution
- Task priority and queuing
- Resource-based task allocation

### Monitoring
- Prometheus metrics integration
- Health check endpoints
- Performance profiling
- Distributed tracing support
