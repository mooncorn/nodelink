# Nodelink Refactor v1.0 - Minimal Architecture Improvements

## Overview

This specification outlines a minimal refactor of the nodelink architecture to address core issues while preserving working functionality. The approach focuses on fixing architectural problems without over-engineering the solution for the current scale (10 agents, 10 containers each).

## Problem Statement

### Current Issues
1. **Circular Dependency**: TaskManager ↔ TaskServer creates initialization complexity
2. **Scattered SSE Logic**: Different streaming patterns across packages
3. **No Server-Side Processing**: Server acts as dumb proxy for agent events
4. **Inconsistent Patterns**: Task-based vs periodic streaming handled differently

### What Works Well (Keep As-Is)
- ✅ gRPC bidirectional streaming for tasks
- ✅ Task lifecycle management
- ✅ Agent authentication and connection handling
- ✅ Basic SSE infrastructure

## Solution: Minimal Refactor Approach

### Core Principles
1. **Interface-Based Decoupling**: Break circular dependencies with interfaces
2. **Event Router Pattern**: Centralize event processing without complex framework
3. **Composition Over Inheritance**: Simple, focused components
4. **Preserve Working Code**: Minimal changes to functioning systems

## Architecture Changes

### 1. Break Circular Dependencies

#### Current Problem
```go
// server/cmd/server/main.go - Circular initialization
taskManager := tasks.NewTaskManager()
agentServer := servergrpc.NewTaskServer(taskManager.GetResponseChannel(), metricsStore)
taskManager.SetTaskServer(agentServer) // Circular dependency
```

#### Solution: Interface-Based Decoupling
```go
// pkg/interfaces/interfaces.go - New shared interfaces
package interfaces

type TaskSender interface {
    SendTask(request *pb.TaskRequest) error
}

type ResponseReceiver interface {
    HandleResponse(response *pb.TaskResponse)
}

// server/pkg/tasks/manager.go - Use interface
type TaskManager struct {
    taskSender TaskSender // Interface, not concrete type
    // ... other fields
}

func (tm *TaskManager) SetTaskSender(sender TaskSender) {
    tm.taskSender = sender
}

// server/pkg/grpc/server.go - Implement interface
func (s *TaskServer) SendTask(request *pb.TaskRequest) error {
    // Implementation
}
```

### 2. Centralized Event Processing

#### Event Router Pattern
```go
// pkg/events/router.go - Central event processing
package events

type EventRouter struct {
    processors map[string]EventProcessor
    sseManager *sse.Manager[*pb.TaskResponse]
    metrics    *metrics.MetricsStore
}

type EventProcessor interface {
    ProcessEvent(event *pb.TaskResponse) (*ProcessedEvent, error)
    GetEventType() string
}

type ProcessedEvent struct {
    OriginalEvent *pb.TaskResponse
    ProcessedData interface{}
    ShouldRelay   bool
    TargetRoom    string
}

func (er *EventRouter) ProcessAndRelay(response *pb.TaskResponse) {
    processor, exists := er.processors[response.GetEventType()]
    if !exists {
        // Default: relay as-is
        er.relayRaw(response)
        return
    }
    
    processed, err := processor.ProcessEvent(response)
    if err != nil {
        log.Printf("Event processing failed: %v", err)
        er.relayRaw(response) // Fallback to raw relay
        return
    }
    
    if processed.ShouldRelay {
        er.relayProcessed(processed)
    }
}
```

### 3. Unified Stream Types

#### Stream Type Registry
```go
// pkg/streams/registry.go - Stream type definitions
package streams

type StreamType struct {
    Name         string
    Description  string
    Buffered     bool
    BufferSize   int
    AutoCleanup  bool
    ProcessorKey string
}

var StreamTypes = map[string]StreamType{
    "shell_output": {
        Name:         "shell_output",
        Description:  "Shell command output streaming",
        Buffered:     true,
        BufferSize:   50,
        AutoCleanup:  true,
        ProcessorKey: "shell",
    },
    "metrics": {
        Name:         "metrics", 
        Description:  "System metrics streaming",
        Buffered:     true,
        BufferSize:   100,
        AutoCleanup:  false,
        ProcessorKey: "metrics",
    },
    "container_logs": {
        Name:         "container_logs",
        Description:  "Container log streaming", 
        Buffered:     true,
        BufferSize:   200,
        AutoCleanup:  true,
        ProcessorKey: "docker",
    },
    "docker_operations": {
        Name:         "docker_operations",
        Description:  "Docker operation status",
        Buffered:     false,
        BufferSize:   0,
        AutoCleanup:  true,
        ProcessorKey: "docker",
    },
}
```

### 4. Specialized Event Processors

#### Shell Output Processor
```go
// pkg/events/processors/shell.go
package processors

type ShellProcessor struct {
    taskManager *tasks.TaskManager
}

func (sp *ShellProcessor) ProcessEvent(event *pb.TaskResponse) (*events.ProcessedEvent, error) {
    shellResp := event.GetShellExecute()
    if shellResp == nil {
        return nil, fmt.Errorf("not a shell response")
    }
    
    // Process shell output (e.g., escape ANSI codes, detect errors)
    processedOutput := sp.processShellOutput(shellResp)
    
    // Update task status
    if event.IsFinal {
        sp.taskManager.UpdateTaskStatus(event.TaskId, tasks.TaskStatusCompleted)
    }
    
    return &events.ProcessedEvent{
        OriginalEvent: event,
        ProcessedData: processedOutput,
        ShouldRelay:   true,
        TargetRoom:    event.TaskId, // Stream to task-specific room
    }, nil
}

type ProcessedShellOutput struct {
    Stdout       string    `json:"stdout"`
    Stderr       string    `json:"stderr"`
    ExitCode     int       `json:"exit_code,omitempty"`
    ErrorType    string    `json:"error_type,omitempty"`
    CleanOutput  string    `json:"clean_output"` // ANSI codes removed
    HasErrors    bool      `json:"has_errors"`
    Timestamp    time.Time `json:"timestamp"`
}
```

#### Metrics Processor
```go
// pkg/events/processors/metrics.go
package processors

type MetricsProcessor struct {
    store *metrics.MetricsStore
}

func (mp *MetricsProcessor) ProcessEvent(event *pb.TaskResponse) (*events.ProcessedEvent, error) {
    metricsResp := event.GetMetricsResponse()
    if metricsResp == nil {
        return nil, fmt.Errorf("not a metrics response")
    }
    
    // Store metrics in database
    mp.store.StoreMetrics(event.AgentId, metricsResp)
    
    // Calculate aggregated metrics
    aggregated := mp.calculateAggregatedMetrics(event.AgentId, metricsResp)
    
    return &events.ProcessedEvent{
        OriginalEvent: event,
        ProcessedData: aggregated,
        ShouldRelay:   true,
        TargetRoom:    fmt.Sprintf("metrics_%s", event.AgentId),
    }, nil
}

type AggregatedMetrics struct {
    AgentID           string              `json:"agent_id"`
    Current           *pb.MetricsResponse `json:"current"`
    Trend             string              `json:"trend"` // "increasing", "decreasing", "stable"
    Alerts            []MetricAlert       `json:"alerts,omitempty"`
    HealthScore       int                 `json:"health_score"` // 0-100
    LastUpdated       time.Time           `json:"last_updated"`
}

type MetricAlert struct {
    Type        string  `json:"type"`
    Severity    string  `json:"severity"`
    Message     string  `json:"message"`
    Threshold   float64 `json:"threshold"`
    CurrentValue float64 `json:"current_value"`
}
```

## Updated Protobuf Definitions

### Enhanced Event Types
```protobuf
// proto/agent.proto - Enhanced event types
message TaskResponse {
  string agent_id = 1;
  string task_id = 2;
  
  enum Status {
    UNKNOWN = 0;
    COMPLETED = 1;
    FAILURE = 2;
    IN_PROGRESS = 3;
  }
  Status status = 3;
  
  bool is_final = 4;
  bool cancelled = 5;
  
  // Enhanced with event metadata
  string event_type = 6;      // "shell_output", "metrics", "docker_operation"
  int64 timestamp = 7;        // Unix timestamp
  map<string, string> metadata = 8; // Additional context
  
  oneof response {
    ShellExecuteResponse shell_execute = 10;
    TaskCancelResponse task_cancel = 11;
    MetricsResponse metrics_response = 12;
    DockerOperationResponse docker_operation = 13;
  }
}

// New Docker operation response
message DockerOperationResponse {
  string operation = 1;       // "run", "start", "stop", "logs"
  string container_id = 2;
  string status = 3;
  string message = 4;
  oneof operation_data {
    DockerRunResult run_result = 5;
    DockerLogsChunk logs_chunk = 6;
  }
}

message DockerRunResult {
  string container_id = 1;
  string image = 2;
  repeated string ports = 3;
  string status = 4;
}

message DockerLogsChunk {
  string container_id = 1;
  string log_line = 2;
  string stream = 3;         // "stdout", "stderr"
  int64 timestamp = 4;
}
```

## TODO List

### Core Infrastructure
- [ ] Create `pkg/interfaces/interfaces.go` with TaskSender and ResponseReceiver interfaces
- [ ] Create `pkg/events/router.go` with EventRouter implementation
- [ ] Create `pkg/events/processor.go` with EventProcessor interface
- [ ] Create `pkg/streams/registry.go` with StreamType definitions
- [ ] Create `pkg/streams/manager.go` for unified stream lifecycle management

### Event Processors
- [ ] Implement `pkg/events/processors/shell.go` for shell output processing
- [ ] Implement `pkg/events/processors/metrics.go` for metrics aggregation
- [ ] Implement `pkg/events/processors/docker.go` for Docker operations
- [ ] Add ANSI code cleaning in shell processor
- [ ] Add health scoring in metrics processor
- [ ] Add alerting logic for metrics thresholds

### Protocol Updates
- [ ] Update `proto/agent.proto` with enhanced TaskResponse fields (event_type, timestamp, metadata)
- [ ] Add DockerOperationResponse message
- [ ] Add DockerRunResult and DockerLogsChunk messages
- [ ] Regenerate protobuf Go files

### Server Refactoring
- [ ] Update `server/pkg/tasks/manager.go` to use TaskSender interface
- [ ] Update `server/pkg/grpc/server.go` to implement TaskSender interface
- [ ] Remove circular dependency in `server/cmd/server/main.go`
- [ ] Integrate EventRouter into main server initialization
- [ ] Update SSE manager to use stream type registry

### API Endpoints
- [ ] Create `GET /api/streams/types` endpoint
- [ ] Create `GET /api/streams/processed/:stream_type/:resource_id` endpoint
- [ ] Remove old SSE endpoints (`/agents/:agent_id/metrics/stream`)
- [ ] Update task endpoints to use new processing system

### Agent Updates
- [ ] Update agents to send event_type in TaskResponse
- [ ] Add timestamp to all agent responses
- [ ] Update Docker operation handlers to use new response format
- [ ] Test agent compatibility with new protocol

### Client Libraries
- [ ] Update TypeScript client to use new unified endpoints
- [ ] Remove old metrics streaming client methods
- [ ] Add support for processed event consumption
- [ ] Update examples and documentation

### Testing
- [ ] Unit tests for all EventProcessor implementations
- [ ] Integration tests for EventRouter
- [ ] End-to-end tests for stream processing
- [ ] Load testing with 10 agents × 10 containers
- [ ] Protocol compatibility tests

### Cleanup
- [ ] Delete `server/pkg/metrics/sse_handler.go`
- [ ] Remove old SSE middleware and handlers
- [ ] Clean up unused imports and dependencies
- [ ] Update documentation and README files

## File Structure Changes

```
server/
├── pkg/
│   ├── interfaces/           # NEW: Shared interfaces
│   │   └── interfaces.go
│   ├── events/              # NEW: Event processing
│   │   ├── router.go
│   │   ├── processor.go
│   │   └── processors/
│   │       ├── shell.go
│   │       ├── metrics.go
│   │       └── docker.go
│   ├── streams/             # NEW: Stream management
│   │   ├── registry.go
│   │   ├── manager.go
│   │   └── types.go
│   ├── grpc/               # UPDATED: Interface implementation
│   │   └── server.go
│   ├── tasks/              # UPDATED: Use interfaces
│   │   ├── manager.go
│   │   └── types.go
│   ├── metrics/            # UPDATED: Integrate with events
│   │   ├── store.go
│   │   └── http_handler.go
│   └── sse/               # UPDATED: Stream type awareness
│       ├── manager.go
│       └── types.go
```

## API Changes

### New Unified Stream Endpoints
```http
# Get available stream types
GET /api/streams/types

# Subscribe to processed events
GET /api/streams/processed/:stream_type/:resource_id

# Task-specific streaming (updated)
GET /api/stream?ref=:task_id
```

### Example Responses
```json
// GET /api/streams/types
{
  "stream_types": [
    {
      "name": "shell_output",
      "description": "Shell command output streaming",
      "supports_processing": true,
      "buffer_size": 50
    },
    {
      "name": "metrics", 
      "description": "System metrics streaming",
      "supports_processing": true,
      "buffer_size": 100
    }
  ]
}

// GET /api/streams/processed/shell_output/task_123
// SSE Event
event: shell_output
data: {
  "original": {...},
  "processed": {
    "stdout": "Hello world\n",
    "clean_output": "Hello world",
    "has_errors": false,
    "timestamp": "2025-08-09T10:30:00Z"
  }
}
```

## Testing Strategy
- Unit tests for all new interfaces and processors
- Integration tests for event routing
- End-to-end tests for streaming functionality
- Load testing with 10 agents × 10 containers
- Protocol compatibility testing between agents and server

## Benefits

### Immediate Improvements
- ✅ Eliminates circular dependencies
- ✅ Centralizes event processing logic
- ✅ Provides foundation for server-side processing
- ✅ Maintains all existing functionality

### Future Capabilities
- Easy addition of new stream types
- Server-side alerting and monitoring
- Event aggregation and analytics
- Improved debugging and observability
- Horizontal scaling preparation

### Clean Architecture Benefits
- Interface-based design for better testability
- Centralized event processing logic
- Consistent patterns across all stream types
- Simplified debugging and monitoring

## Success Metrics

- 50% reduction in SSE-related code complexity
- New stream type can be added in < 1 day
- CPU usage remains within 10% of current levels
- Memory usage scales linearly with active streams
- All streaming features work consistently

## Conclusion

This minimal refactor addresses the core architectural issues by eliminating circular dependencies, centralizing event processing, and providing a clean foundation for server-side processing. The interface-based approach and event router pattern create a maintainable architecture that can easily accommodate new streaming features while keeping the complexity appropriate for the project's scale.
