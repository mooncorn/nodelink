# Internal Packages Architecture

This document describes the reorganized internal package structure for the NodeLink server, following single-responsibility principles and dependency injection patterns.

## Package Overview

### 1. Agent Package (`internal/agent/`)
Manages agent connections and communication with single-responsibility design.

**Files:**
- `manager.go` - Core agent connection management and communication
- `service.go` - gRPC service implementation for agent operations

**Key Features:**
- Bidirectional streaming communication with agents
- Connection lifecycle management
- Agent capability tracking
- Health monitoring integration
- Thread-safe concurrent operations

**Dependencies:**
- `heartbeat` package for health monitoring
- `proto` package for gRPC communication

### 2. Command Package (`internal/command/`)
Handles all command-related functionality with separate concerns.

**Files:**
- `service.go` - Main service that orchestrates all command functionality
- `server.go` - gRPC command server implementation
- `processor.go` - Command output processing and cleanup
- `http_handler.go` - HTTP REST API for command execution
- `sse_handler.go` - Server-Sent Events for real-time command streaming

**Key Features:**
- HTTP REST API for simple command execution
- SSE streaming for real-time command output
- gRPC streaming for agent communication
- ANSI code cleanup and output processing
- Command cancellation support

**Dependencies:**
- `agent` package for agent communication
- `sse` package for streaming
- External: `gin` for HTTP handling

### 3. Heartbeat Package (`internal/heartbeat/`)
Standalone package for agent health monitoring and heartbeat management.

**Files:**
- `service.go` - Core heartbeat monitoring service
- `http_handler.go` - HTTP API for heartbeat status queries
- `package.go` - Package-level service wrapper

**Key Features:**
- Continuous agent health monitoring
- Configurable heartbeat intervals
- Missed heartbeat tracking
- Health status callbacks
- REST API for health queries

**Dependencies:**
- Minimal external dependencies
- Designed for easy testing and reuse

## Architecture Patterns

### Single Responsibility Principle
Each package and file has a clearly defined purpose:
- Agent package: Connection management only
- Command package: Command execution only  
- Heartbeat package: Health monitoring only

### Dependency Injection
Services are composed using dependency injection:
```go
// Example: Command service composition
agentManager := agent.NewManager(config, heartbeatService)
commandService := command.NewService(agentManager, sseManager, config)
```

### Interface Segregation
Each package exposes minimal, focused interfaces:
- Agent Manager provides connection management methods
- Command Service provides execution methods
- Heartbeat Service provides monitoring methods

### Separation of Concerns

#### HTTP Layer
- REST APIs for simple request/response operations
- Standard HTTP status codes and JSON responses
- Input validation and error handling

#### SSE Layer  
- Real-time streaming for long-running operations
- Event-based communication
- Automatic reconnection support

#### gRPC Layer
- Agent-to-server communication
- Bidirectional streaming
- Protocol buffer efficiency

#### Processing Layer
- Business logic and data transformation
- Output cleanup and formatting
- Event processing

## Integration Points

### Agent Registration Flow
1. Agent connects via gRPC `ManageAgent` stream
2. Agent sends capabilities message
3. Agent Manager registers the connection
4. Heartbeat Service begins monitoring
5. Agent sends periodic heartbeats

### Command Execution Flow
1. HTTP request received by Command HTTP Handler
2. Request validated and forwarded to Agent Manager
3. Agent Manager sends command via gRPC stream
4. Agent executes command and streams responses
5. Responses processed and returned to client

### Real-time Streaming Flow
1. SSE connection established via Command SSE Handler
2. Command execution initiated
3. Output streamed in real-time via SSE
4. Client receives formatted events

## Configuration

Each package accepts configuration through structured config objects:
```go
type ManagerConfig struct {
    HeartbeatInterval time.Duration
    ConnectionTimeout time.Duration
}

type ServiceConfig struct {
    MaxMissedBeats   int
    CheckInterval    time.Duration
    OnAgentUnhealthy func(agentID string)
    OnAgentHealthy   func(agentID string)
}
```

## Testing Strategy

- Unit tests for each service in isolation
- Integration tests for package interactions
- Mock interfaces for external dependencies
- Test helpers for common scenarios

## Future Extensions

The architecture supports easy extension:
- New command types can be added to the command package
- New monitoring metrics can be added to heartbeat
- New communication protocols can be added to agent
- Additional HTTP/SSE endpoints can be added easily

This modular design ensures maintainability, testability, and scalability while following Go best practices and clean architecture principles.
