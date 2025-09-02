# Nodelink Server - Claude Configuration

## Project Overview
The server is the central orchestrator that manages agent connections, handles HTTP APIs, and provides real-time streaming via Server-Sent Events.

## Key Technologies
- **Language**: Go 1.21+
- **Web Framework**: Standard library HTTP with custom routing
- **Communication**: gRPC server with bidirectional streaming
- **Real-time**: Server-Sent Events (SSE) for live updates
- **Protocol**: Protocol Buffers (generated from `proto/agent.proto`)

## Project Structure
```
server/
├── cmd/server/main.go         # Main entry point
├── internal/
│   ├── proto/                 # Generated protobuf files
│   ├── comm/                  # gRPC communication layer
│   ├── status/                # Agent status management
│   ├── command/               # Command execution coordination
│   ├── terminal/              # Terminal session management
│   ├── metrics/               # Metrics collection and streaming
│   ├── sse/                   # Server-Sent Events infrastructure
│   ├── auth/                  # Agent authentication
│   └── common/                # Shared types and interfaces
└── tmp/                       # Build artifacts
```

## Architecture Principles

### Layered Architecture
The server follows a strict layered architecture with linear dependency flow:

**Dependency Rules:**
- Each layer can ONLY depend on layers below it
- NO circular dependencies allowed
- NO cross-layer dependencies (e.g., Layer 7 cannot depend on Layer 3 directly)
- Use dependency injection and interfaces for loose coupling

1. **Layered Dependencies**: Each layer can ONLY import from layers below it
2. **Interface Segregation**: Use small, focused interfaces
3. **Dependency Injection**: Inject dependencies through constructors
4. **Event-Driven**: Use event bus for cross-domain communication
5. **Repository Pattern**: All data access through repository interfaces
6. **Service Layer**: Business logic belongs in application services

#### Interface Usage
- **Always use interfaces for dependencies**: Inject interfaces, not concrete types
- **Repository Interfaces**: All data access through repository interfaces
- **Service Interfaces**: Business logic exposed through service interfaces
- **Infrastructure Interfaces**: Abstract external dependencies

#### Event-Driven Design
- **Domain Events**: Emit events for business state changes
- **Async Processing**: Use event bus for non-blocking operations
- **Event Handlers**: Subscribe to events through dedicated handlers
- **Event Sourcing**: Consider events as first-class citizens

#### Clean Code Principles
- **Single Responsibility**: Each component has one reason to change
- **Open/Closed**: Open for extension, closed for modification
- **Liskov Substitution**: Subtypes must be substitutable for base types
- **Interface Segregation**: Many client-specific interfaces
- **Dependency Inversion**: Depend on abstractions, not concretions

### Package Organization Rules

1. **Layer-Based Packages**: Organize by architectural layer, not by feature
2. **Domain Separation**: Clear boundaries between business domains
3. **Interface Definition**: Interfaces defined in the layer that uses them
4. **Implementation Location**: Implementations in appropriate architectural layers
5. **Circular Dependency Prevention**: Use dependency injection to break cycles

## Development Guidelines
- Follow strict layered dependency flow: higher layers can only depend on lower layers
- Define interfaces in the layer that consumes them (dependency inversion)
- Use dependency injection for all cross-layer dependencies
- Implement event-driven patterns for loose coupling between domains
- Apply repository pattern for data access abstraction
- Keep business logic in dedicated service layers
- Use event bus for cross-cutting concerns and async processing