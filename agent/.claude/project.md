# Nodelink Agent - Claude Configuration

## Project Overview
The agent is a Go-based task executor that connects to the central server via gRPC bidirectional streaming.

## Key Technologies
- **Language**: Go 1.21+
- **Communication**: gRPC client with bidirectional streaming
- **Protocol**: Protocol Buffers (generated from `proto/agent.proto`)
- **Build Tool**: Go modules

## Project Structure
```
agent/
├── cmd/agent/main.go          # Main entry point
├── internal/proto/            # Generated protobuf files
├── pkg/
│   ├── grpc/client.go         # gRPC client implementation
│   ├── command/executor.go    # Command execution logic
│   ├── terminal/manager.go    # Terminal session management
│   └── metrics/collector.go   # System metrics collection
└── scripts/                   # Setup and utility scripts
```

## Development Guidelines
- Use generated protobuf types from `internal/proto/`
- Implement bidirectional streaming handlers
- Handle connection lifecycle and reconnection logic
- Collect and report system metrics
- Execute commands securely with proper resource limits

## Key Interfaces
- `grpc.ClientConnInterface`: gRPC connection management
- `proto.AgentClient`: Generated gRPC client interface
- Command execution with timeout and resource monitoring

## Common Tasks
- Update protobuf handlers when `proto/agent.proto` changes
- Implement new command types and execution logic
- Add new metrics collection capabilities
- Handle connection failures and reconnection
