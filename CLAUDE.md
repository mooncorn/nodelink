# Copilot Instructions for Nodelink Project

## Project Overview
Nodelink is a distributed task execution system with gRPC-based agent communication and real-time streaming capabilities. The system consists of:
- **Server**: Central orchestrator with HTTP APIs and SSE streaming
- **Agent**: Task executor that connects to server via gRPC
- **Proto**: Protocol buffer definitions (source of truth in `proto/agent.proto`)

## Architecture Components

### Communication Protocol
- **gRPC**: Bidirectional streaming between server and agents
- **Protocol Buffers**: [`agent.proto`](proto/agent.proto) defines all message types
- **Generated Code**: Each service has its own generated protobuf files in `internal/proto/`
- **Message Types**: Uses `oneof` for type-safe message routing (ping/pong, commands, responses)

### Internal Package Structure
Each internal package is self-contained and manages its own domain:

#### Status Management (`internal/status/`)
- **Purpose**: Centralized agent status tracking and lifecycle management
- **Components**:
  - `manager.go`: Core status tracking with event notifications
  - `http_handler.go`: HTTP API for querying agent status
  - `sse_handler.go`: Real-time status change streaming
- **Dependencies**: None (foundation layer)

#### Ping/Pong Monitoring (`internal/ping/`)
- **Purpose**: Connection health monitoring through periodic ping/pong exchanges
- **Components**:
  - `handler.go`: Ping/pong message processing and timeout management
- **Dependencies**: `status` (updates agent status based on ping responses)

#### Command Execution (`internal/command/`)
- **Purpose**: Remote command execution on agents
- **Components**:
  - `handler.go`: Command request/response management
  - `http_handler.go`: HTTP API for executing commands
- **Dependencies**: `status` (validates agent availability)

#### Terminal Management (`internal/terminal/`)
- **Purpose**: Interactive terminal session management and real-time terminal streaming
- **Components**:
  - `handler.go`: Terminal session lifecycle and command processing
  - `http_handler.go`: HTTP API for terminal operations
  - `sse_handler.go`: Real-time terminal output streaming
  - `session.go`: Terminal session state management
- **Dependencies**: `status`, `common`, `sse` (validates agent availability, uses shared interfaces, leverages SSE utilities)

#### Metrics Collection (`internal/metrics/`)
- **Purpose**: System metrics collection with separated streaming and system information access
- **Components**:
  - `handler.go`: Metrics request/response management
  - `http_handler.go`: HTTP API for metrics endpoints
  - `sse_handler.go`: Real-time metrics streaming (metrics only) + system info GET endpoint
  - `manager.go`: Centralized metrics polling and distribution
- **API Endpoints**:
  - `GET /metrics/:agentID/stream`: SSE stream for real-time metrics updates only
  - `GET /metrics/:agentID/system-info`: HTTP endpoint for system information
- **Dependencies**: `status`, `common`, `sse` (validates agent availability, uses shared interfaces, leverages SSE utilities)

#### Common Types and Interfaces (`internal/common/`)
- **Purpose**: Shared types, interfaces, and constants used across packages
- **Components**:
  - `types.go`: Common data structures (AgentInfo, AgentStatus, StatusChangeEvent, etc.)
  - `interfaces.go`: Shared interfaces (StreamSender, StatusManager, Authenticator, etc.)
  - `constants.go`: Common constants and error definitions
- **Dependencies**: None (foundation layer)

#### Server-Sent Events Infrastructure (`internal/sse/`)
- **Purpose**: Real-time streaming infrastructure with convention-based streaming patterns
- **Components**:
  - `manager.go`: SSE client management and message distribution
  - `stream.go`: Fluent API with domain-specific stream builders (AgentStreamBuilder, TerminalStreamBuilder, MetricsStreamBuilder)
  - `broadcaster.go`: Centralized message broadcasting with standard room conventions
- **Dependencies**: `common` (implements common interfaces)
- **Architecture**: Uses "convention over configuration" with fluent API for simple, consistent SSE handling

#### Authentication (`internal/auth/`)
- **Purpose**: Agent authentication and security
- **Components**:
  - `authenticator.go`: Agent authentication logic
- **Dependencies**: `common` (uses shared error definitions)

#### Communication (`internal/comm/`)
- **Purpose**: gRPC stream management and message routing
- **Components**:
  - `communication.go`: Bidirectional stream handling and message dispatch
- **Dependencies**: `status`, `ping`, `command`, `terminal`, `metrics`, `auth` (coordinates all communication)

## Development Guidelines

### Code Quality Guidelines

#### Interface Usage
- **Always use interfaces when injecting dependencies**: Prefer `common.StatusManager` over `*status.Manager`
- **Use interfaces, types, and constants from common package when possible**: Leverage shared definitions to maintain consistency
- **Move interfaces, types, and constants to common unless domain-specific**: Only keep in local packages when they are truly package-specific

#### SSE Development
- **Use the StreamBuilder fluent API**: Leverage domain-specific stream builders for consistent SSE handling
- **Follow convention-based patterns**: Use standard room naming and broadcasting patterns
- **Prefer StreamBuilder over manual configuration**: Use `streamBuilder.ForAgent(id).WithCurrentStatus(manager).Handle(c)` instead of manual setup

### Package Design Principles
1. **Single Responsibility**: Each package manages one core concern
2. **Self-Contained**: Packages include their own HTTP/SSE handlers
3. **Dependency Direction**: Clear one-way dependency flow (no circular dependencies)
4. **Event-Driven**: Status changes propagate through listener pattern

### Dependency Flow
```
comm → status, ping, command, terminal, metrics, auth, common, sse (orchestrates all)
terminal → status, common, sse (manages sessions, uses shared interfaces, leverages SSE utilities)
metrics → status, common, sse (collects metrics, uses shared interfaces, leverages SSE utilities)
ping → status, common (updates agent health, uses shared constants)
command → status, common (checks agent availability, uses shared interfaces)
auth → common (uses shared error definitions)
status → common (uses shared types and interfaces)
sse → common (implements common interfaces)
common → (no dependencies - foundation layer)
```

### SSE Architecture (Refactored)
The SSE system uses a "convention over configuration" approach with fluent APIs:

#### StreamBuilder Pattern
```go
// Agent status streaming
h.streamBuilder.ForAgent(agentID).WithCurrentStatus(manager).Handle(c)

// Terminal streaming with authentication
h.streamBuilder.ForTerminal(sessionID).RequireAuth(sessionManager).Handle(c)

// Metrics streaming with cached data
h.streamBuilder.ForMetrics(agentID).WithStatusManager(statusManager).WithCachedData(metrics, nil).Handle(c)
```

#### Broadcaster Pattern
```go
// Centralized broadcasting with standard room conventions
broadcaster.AgentStatus(event)        // → "agents" and "agent_{id}" rooms
broadcaster.TerminalOutput(sessionID, output) // → "terminal_{sessionID}" room
broadcaster.Metrics(agentID, metrics) // → "metrics_{agentID}" room
```

#### Benefits
- **90% less boilerplate**: Handler methods reduced from 15+ lines to 1-3 lines
- **Convention-based room naming**: Automatic room management
- **Type-safe configuration**: Fluent API prevents configuration mistakes
- **Centralized broadcasting**: Single point for message distribution

### Adding New Features
1. Update protocol buffers in `proto/agent.proto` if new message types are needed
2. Run `./generate.sh` to regenerate protobuf files in both services
3. Create new package in `internal/` for the feature domain 
4. Implement core logic and HTTP handlers within the package
5. For SSE streaming, use StreamBuilder fluent API: `streamBuilder.ForDomain(id).WithOptions().Handle(c)`
6. For broadcasting, use Broadcaster: `broadcaster.DomainEvent(data)`
7. Add dependencies only to lower-level packages
8. Register routes in main.go
9. Implement core logic in the agent

## File Organization

### Server Structure
- `cmd/server/main.go`: Main server entry point
- `internal/proto/`: Generated protobuf files for server
- `internal/sse/`: Real-time streaming infrastructure with StreamBuilder and Broadcaster
- `internal/status/`: Centralized agent status tracking
- `internal/ping/`: Heartbeat and connection monitoring
- `internal/command/`: Command execution feature
- `internal/terminal/`: Interactive terminal session management
- `internal/metrics/`: System metrics collection with separated streaming (metrics) and HTTP (system info)
- `internal/comm/`: gRPC communication and message routing
- `internal/auth/`: Agent authentication and security
- `internal/common/`: Shared types, interfaces, and constants

### Agent Structure  
- `cmd/agent/main.go`: Main agent entry point
- `internal/proto/`: Generated protobuf files for agent
- `pkg/grpc/client.go`: gRPC client implementation
- `pkg/command/`: Command execution handling on agent side
- `pkg/terminal/`: Terminal session management on agent side
- `pkg/metrics/`: Metrics collection on agent side

### Protocol Definitions
- `proto/agent.proto`: Protocol buffer definitions (source of truth)
- `generate.sh`: Script to generate protobuf files in both services
- Generated files are located in `server/internal/proto/` and `agent/internal/proto/`

## Security Considerations
- Agent authentication via tokens
- Resource limits and timeouts for long-running tasks