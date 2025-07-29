# gRPC Communication Layer

This document describes the gRPC-based communication system between the server and agents in the nodelink project.

## Overview

The gRPC communication layer provides:
- **Bidirectional Streaming**: Real-time task distribution and response collection
- **Type Safety**: Protocol Buffer message definitions
- **Connection Management**: Automatic reconnection and health monitoring
- **Scalability**: Support for multiple concurrent agents
- **Reliability**: Message ordering and delivery guarantees

## Architecture

```
┌─────────────────┐                    ┌──────────────────┐
│     Server      │                    │      Agent       │
│                 │                    │                  │
│  TaskServer     │ ◄──── Stream ────► │   TaskClient     │
│  - Send Tasks   │                    │   - Receive      │
│  - Collect      │                    │   - Process      │
│    Responses    │                    │   - Respond      │
└─────────────────┘                    └──────────────────┘
         │                                       │
         ▼                                       ▼
┌─────────────────┐                    ┌──────────────────┐
│  Task Manager   │                    │  Task Handlers   │
│  - Route Tasks  │                    │  - Shell Exec    │
│  - Update State │                    │  - Docker Ops    │
│  - Notify SSE   │                    │  - Cancellation  │
└─────────────────┘                    └──────────────────┘
```

## Protocol Buffer Definitions

### 1. Task Request Structure

```protobuf
message TaskRequest {
  string agent_id = 1;    // Target agent identifier
  string task_id = 2;     // Unique task identifier  
  oneof task {
    ShellExecute shell_execute = 3;        // Shell command execution
    DockerOperation docker_operation = 4;   // Docker container operations
    TaskCancel task_cancel = 5;            // Task cancellation request
  }
}
```

### 2. Task Response Structure

```protobuf
message TaskResponse {
  string agent_id = 1;              // Responding agent identifier
  string task_id = 2;               // Task identifier for correlation
  TaskResponse.Status status = 3;   // Execution status
  bool is_final = 4;               // Whether this is the final response
  bool cancelled = 5;              // Whether task was cancelled
  oneof response {
    ShellExecuteResponse shell_execute = 6;        // Shell execution output
    DockerOperationResponse docker_operation = 7;  // Docker operation result
    TaskCancelResponse task_cancel = 8;            // Cancellation acknowledgment
  }
}
```

### 3. Shell Execution Messages

```protobuf
message ShellExecute {
  string cmd = 1;           // Command to execute
  repeated string env = 2;  // Environment variables (future)
  string working_dir = 3;   // Working directory (future)
}

message ShellExecuteResponse {
  string stdout = 1;    // Standard output (incremental)
  string stderr = 2;    // Standard error (incremental)  
  int32 exit_code = 3;  // Process exit code (final response only)
}
```

### 4. Task Cancellation Messages

```protobuf
message TaskCancel {
  string reason = 1;    // Cancellation reason
}

message TaskCancelResponse {
  string message = 1;   // Cancellation acknowledgment
}
```

## Server-Side Implementation

### 1. TaskServer Structure

```go
type TaskServer struct {
    mu          sync.RWMutex
    agents      map[string]*AgentConnection  // Connected agents
    taskManager *TaskManager                 // Task lifecycle manager
    respCh      chan<- *pb.TaskResponse     // Response processing channel
}

type AgentConnection struct {
    agentID    string
    stream     pb.TaskService_StreamTasksServer
    lastSeen   time.Time
    mu         sync.Mutex
}
```

### 2. Agent Connection Handling

```go
func (ts *TaskServer) StreamTasks(stream pb.TaskService_StreamTasksServer) error {
    // Authenticate agent from stream context
    agentID, err := ts.authenticateAgent(stream.Context())
    if err != nil {
        return status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
    }
    
    // Register agent connection
    ts.registerAgent(agentID, stream)
    defer ts.unregisterAgent(agentID)
    
    // Handle incoming responses
    for {
        resp, err := stream.Recv()
        if err != nil {
            if err == io.EOF {
                break // Client closed connection
            }
            return err
        }
        
        // Forward response to task manager
        ts.respCh <- resp
    }
    
    return nil
}
```

### 3. Task Distribution

```go
func (ts *TaskServer) Send(taskRequest *pb.TaskRequest) error {
    ts.mu.RLock()
    agent, exists := ts.agents[taskRequest.AgentId]
    ts.mu.RUnlock()
    
    if !exists {
        return fmt.Errorf("agent %s not connected", taskRequest.AgentId)
    }
    
    agent.mu.Lock()
    err := agent.stream.Send(taskRequest)
    agent.lastSeen = time.Now()
    agent.mu.Unlock()
    
    return err
}
```

**Features:**
- **Agent Lookup**: O(1) agent connection lookup
- **Connection Validation**: Checks agent availability before sending
- **Thread Safety**: Per-agent locks prevent concurrent stream access
- **Heartbeat Tracking**: Updates last seen timestamp

### 4. Connection Management

```go
func (ts *TaskServer) registerAgent(agentID string, stream pb.TaskService_StreamTasksServer) {
    ts.mu.Lock()
    defer ts.mu.Unlock()
    
    // Close existing connection if present
    if existingAgent, exists := ts.agents[agentID]; exists {
        existingAgent.stream.Context().Done()
    }
    
    // Register new connection
    ts.agents[agentID] = &AgentConnection{
        agentID:  agentID,
        stream:   stream,
        lastSeen: time.Now(),
    }
    
    log.Printf("Agent %s connected", agentID)
}
```

## Client-Side Implementation

### 1. TaskClient Structure

```go
type TaskClient struct {
    conn        *grpc.ClientConn
    client      pb.TaskServiceClient
    stream      pb.TaskService_StreamTasksClient
    listeners   []TaskRequestListener
    mu          sync.RWMutex
    connected   bool
    agentID     string
    agentToken  string
}

type TaskRequestListener func(taskRequest *pb.TaskRequest)
```

### 2. Connection Establishment

```go
func (tc *TaskClient) Connect(agentID, agentToken string) error {
    tc.agentID = agentID
    tc.agentToken = agentToken
    
    // Create stream with authentication metadata
    ctx := metadata.AppendToOutgoingContext(context.Background(),
        "agent_id", agentID,
        "agent_token", agentToken)
    
    stream, err := tc.client.StreamTasks(ctx)
    if err != nil {
        return fmt.Errorf("failed to create stream: %w", err)
    }
    
    tc.stream = stream
    tc.connected = true
    
    // Start receiving messages
    go tc.receiveLoop()
    
    return nil
}
```

### 3. Message Reception

```go
func (tc *TaskClient) receiveLoop() {
    for tc.connected {
        taskRequest, err := tc.stream.Recv()
        if err != nil {
            if err == io.EOF {
                log.Println("Server closed connection")
            } else {
                log.Printf("Stream receive error: %v", err)
            }
            tc.connected = false
            break
        }
        
        // Notify all listeners
        tc.notifyListeners(taskRequest)
    }
}

func (tc *TaskClient) notifyListeners(taskRequest *pb.TaskRequest) {
    tc.mu.RLock()
    listeners := make([]TaskRequestListener, len(tc.listeners))
    copy(listeners, tc.listeners)
    tc.mu.RUnlock()
    
    for _, listener := range listeners {
        go listener(taskRequest) // Async notification
    }
}
```

### 4. Response Sending

```go
func (tc *TaskClient) Send(response *pb.TaskResponse) error {
    if !tc.connected {
        return fmt.Errorf("client not connected")
    }
    
    // Set agent ID if not already set
    if response.AgentId == "" {
        response.AgentId = tc.agentID
    }
    
    return tc.stream.Send(response)
}
```

## Authentication and Authorization

### 1. Agent Authentication

```go
func (ts *TaskServer) authenticateAgent(ctx context.Context) (string, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return "", fmt.Errorf("no metadata provided")
    }
    
    agentIDs := md.Get("agent_id")
    agentTokens := md.Get("agent_token")
    
    if len(agentIDs) == 0 || len(agentTokens) == 0 {
        return "", fmt.Errorf("missing agent_id or agent_token")
    }
    
    agentID := agentIDs[0]
    agentToken := agentTokens[0]
    
    // Validate token (implementation specific)
    if !ts.isValidToken(agentID, agentToken) {
        return "", fmt.Errorf("invalid credentials for agent %s", agentID)
    }
    
    return agentID, nil
}
```

### 2. Token Validation

```go
func (ts *TaskServer) isValidToken(agentID, agentToken string) bool {
    // Simple token validation (replace with proper implementation)
    expectedToken := ts.getExpectedToken(agentID)
    return agentToken == expectedToken
}
```

**Security Features:**
- **Metadata Authentication**: Uses gRPC metadata for credentials
- **Per-Agent Tokens**: Each agent has unique authentication token
- **Connection Validation**: Checks credentials on every connection
- **Token Rotation**: Supports token updates (future enhancement)

## Error Handling and Recovery

### 1. Connection Errors

```go
func (tc *TaskClient) handleConnectionError(err error) {
    log.Printf("Connection error: %v", err)
    tc.connected = false
    
    // Attempt reconnection with exponential backoff
    go tc.reconnectWithBackoff()
}

func (tc *TaskClient) reconnectWithBackoff() {
    backoff := time.Second
    maxBackoff := time.Minute
    
    for !tc.connected {
        time.Sleep(backoff)
        
        err := tc.Connect(tc.agentID, tc.agentToken)
        if err == nil {
            log.Println("Reconnected successfully")
            return
        }
        
        log.Printf("Reconnection failed: %v", err)
        backoff = time.Duration(float64(backoff) * 1.5)
        if backoff > maxBackoff {
            backoff = maxBackoff
        }
    }
}
```

### 2. Stream Errors

```go
func (ts *TaskServer) handleStreamError(agentID string, err error) {
    log.Printf("Stream error for agent %s: %v", agentID, err)
    
    // Remove agent from active connections
    ts.unregisterAgent(agentID)
    
    // Mark agent tasks as failed
    ts.taskManager.FailAgentTasks(agentID, "agent disconnected")
}
```

### 3. Message Delivery Guarantees

**Ordering**: gRPC guarantees message ordering within a single stream
**Delivery**: Messages are delivered at-least-once (duplicates possible on reconnection)
**Reliability**: Connection errors trigger task failure and cleanup

## Performance Optimization

### 1. Connection Pooling

```go
// Future enhancement: connection pooling for high-throughput scenarios
type ConnectionPool struct {
    mu          sync.RWMutex
    connections map[string]*grpc.ClientConn
    maxConns    int
}
```

### 2. Message Batching

```go
// Future enhancement: batch responses for high-frequency updates
type ResponseBatcher struct {
    responses []pb.TaskResponse
    timer     *time.Timer
    batchSize int
    interval  time.Duration
}
```

### 3. Stream Management

**Keep-Alive Settings**:
```go
var kacp = keepalive.ClientParameters{
    Time:                10 * time.Second,
    Timeout:             3 * time.Second,
    PermitWithoutStream: true,
}

conn, err := grpc.Dial(address, 
    grpc.WithKeepaliveParams(kacp),
    grpc.WithInsecure())
```

## Monitoring and Observability

### 1. Connection Metrics

```go
type Metrics struct {
    ConnectedAgents    prometheus.Gauge
    MessagesReceived   prometheus.Counter
    MessagesSent       prometheus.Counter
    ConnectionFailures prometheus.Counter
    StreamErrors       prometheus.Counter
}
```

### 2. Health Checks

```go
func (ts *TaskServer) HealthCheck() map[string]interface{} {
    ts.mu.RLock()
    defer ts.mu.RUnlock()
    
    return map[string]interface{}{
        "connected_agents": len(ts.agents),
        "total_connections": ts.totalConnections,
        "uptime": time.Since(ts.startTime),
    }
}
```

### 3. Distributed Tracing

```go
// Future enhancement: OpenTelemetry integration
func (ts *TaskServer) Send(ctx context.Context, taskRequest *pb.TaskRequest) error {
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(
        attribute.String("agent_id", taskRequest.AgentId),
        attribute.String("task_id", taskRequest.TaskId),
    )
    defer span.End()
    
    // Send task...
}
```

## Security Considerations

### 1. Transport Security

```go
// TLS configuration for production
creds, err := credentials.LoadTLSServerCerts("server.crt", "server.key")
s := grpc.NewServer(grpc.Creds(creds))
```

### 2. Authentication Middleware

```go
func (ts *TaskServer) authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    // Validate authentication for unary calls
    if err := ts.validateAuth(ctx); err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "authentication failed")
    }
    return handler(ctx, req)
}
```

### 3. Rate Limiting

```go
// Future enhancement: rate limiting per agent
type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
}
```

## Future Enhancements

### 1. Advanced Features
- **Message Compression**: gzip compression for large payloads
- **Load Balancing**: Multiple server instances with agent affinity
- **Failover**: Automatic failover to backup servers

### 2. Protocol Extensions
- **File Transfer**: Stream file uploads/downloads
- **Interactive Sessions**: Bidirectional interactive shell
- **Heartbeat Protocol**: Explicit ping/pong for connection validation

### 3. Observability
- **Structured Logging**: JSON-formatted logs with correlation IDs
- **Metrics Dashboard**: Grafana dashboard for gRPC metrics
- **Alerting**: Prometheus alerts for connection failures

### 4. Security Enhancements
- **mTLS**: Mutual TLS authentication
- **JWT Tokens**: Structured authentication tokens
- **RBAC**: Role-based access control for agents
