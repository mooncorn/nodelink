# Event-Driven Architecture Implementation

## Overview

The nodelink architecture has been successfully refactored from a circular dependency pattern to a true event-driven architecture using a publish-subscribe pattern. This eliminates tight coupling between components and improves maintainability.

## Architecture Components

### Core Event System

1. **Event Bus** (`/server/pkg/events/bus.go`)
   - Central messaging hub using publish-subscribe pattern
   - Thread-safe operations with mutex protection
   - Supports both synchronous and asynchronous event handling
   - Includes panic recovery for robust event processing

2. **Event Interfaces** (`/server/pkg/interfaces/interfaces.go`)
   - `Event`: Core event structure with type, data, source, and timestamp
   - `EventHandler`: Interface for components that handle events
   - `EventBus`: Interface for publish-subscribe messaging

### Communication Flow

```
Client Request (HTTP/SSE) 
    ↓
TaskManager.SendTask()
    ↓
EventBus.Publish("task.send", TaskRequest)
    ↓
TaskServer.handleTaskSendEvent()
    ↓
gRPC Send to Agent
    ↓ 
Agent Processing
    ↓
gRPC Response to TaskServer
    ↓
TaskManager.GetResponseChannel()
    ↓
SSE/HTTP Response to Client
```

## Event Types

### Task Events
- `task.send`: Published when TaskManager needs to send a task to an agent
- `task.send.failed`: Published when task sending fails
- `task.response`: Published when agent responds (future enhancement)

### Metrics Events
- Processed through existing event router system
- Forwarded to metrics store and SSE handlers

## Eliminated Circular Dependencies

### Before (Circular)
```
TaskManager → TaskSender interface → TaskServer
TaskServer → ResponseChannel → TaskManager
```

### After (Event-Driven)
```
TaskManager → EventBus ← TaskServer
    ↓               ↑
EventBus.Publish   EventBus.Subscribe
("task.send")      ("task.send")
```

## Implementation Details

### TaskManager Changes
- Constructor now requires `EventBus` instead of `TaskSender`
- `SendTask()` publishes events instead of direct method calls
- Removed `SetTaskSender()` method (no longer needed)

### TaskServer Changes  
- Constructor accepts `EventBus` parameter
- Subscribes to "task.send" events in constructor
- `handleTaskSendEvent()` processes task send events
- Publishes "task.send.failed" events for error handling
- Still implements `TaskSender` interface for backward compatibility

### Main.go Integration
- Creates single `EventBus` instance
- Passes EventBus to both TaskManager and TaskServer
- No direct dependency injection between TaskManager and TaskServer

## Benefits

1. **Loose Coupling**: Components communicate through events, not direct references
2. **Scalability**: Easy to add new event handlers without modifying existing code
3. **Testability**: Components can be tested in isolation with mock event buses
4. **Maintainability**: Changes to one component don't require changes to others
5. **Flexibility**: Event flow can be modified without changing component interfaces

## Backward Compatibility

- TaskServer still implements the `TaskSender` interface
- Existing HTTP/SSE endpoints work unchanged
- Event processing system remains functional
- Legacy metrics handling continues to work

## Future Enhancements

1. Add more event types for comprehensive decoupling
2. Implement event persistence for reliability
3. Add event replay capabilities for debugging
4. Create event monitoring dashboard
5. Implement event-based testing framework

## Configuration

The event bus supports both synchronous and asynchronous modes:
- **Async Mode** (default): Better performance, non-blocking event publishing
- **Sync Mode**: Immediate error feedback, easier debugging

Current configuration in main.go:
```go
eventBus := events.NewEventBus(true) // Async mode enabled
```
