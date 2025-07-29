# SSE Automatic Streaming Management

This document describes the automatic streaming management feature that optimizes resource usage by automatically starting and stopping metrics streaming based on SSE client connections.

## Problem Solved

Previously, if an SSE client connected to stream metrics and then disconnected without explicitly calling the stop streaming endpoint, the agent would continue collecting and streaming metrics indefinitely, even though no one was listening. This resulted in:

- Wasted CPU and memory resources on agents
- Unnecessary network traffic
- Agents reporting `streaming_active: true` when no clients were connected

## Solution

The SSE handler now includes intelligent client tracking that automatically manages the streaming lifecycle:

### Key Components

1. **Client Reference Counting**: Tracks how many SSE clients are connected to each agent's metrics stream
2. **Automatic Start**: When the first SSE client connects, automatically starts metrics streaming
3. **Automatic Stop**: When the last SSE client disconnects, automatically stops metrics streaming
4. **Connection Lifecycle Hooks**: Uses SSE manager's `OnDisconnect` event to handle cleanup

### Implementation Details

#### SSE Handler (`server/pkg/metrics/sse_handler.go`)

- **Client Counter**: `clientCounter map[string]int` tracks active connections per agent
- **Connection Tracking**: `trackClientConnection()` increments counter when clients join
- **Disconnection Handling**: `handleClientDisconnect()` decrements counter and stops streaming when reaching zero
- **Automatic Start**: `ensureStreamingStarted()` starts streaming only for the first client
- **Room-Based Detection**: Monitors which metrics rooms clients disconnect from

#### Modified SSE Manager (`server/pkg/sse/manager.go`)

- **Event Order**: `OnDisconnect` handler is called before removing client from rooms
- **Room Access**: Allows disconnect handler to see which rooms the client was in
- **Clean Disconnection**: Proper resource cleanup on client disconnect

### Usage

#### Automatic Mode (Recommended)
```typescript
// Simply connect via SSE - streaming starts automatically
const eventSource = client.streamMetrics('agent1', 5);
eventSource.addEventListener('metrics', (event) => {
  const metrics = JSON.parse(event.data);
  // Handle metrics...
});

// When done, close the connection - streaming stops automatically if last client
eventSource.close();
```

#### Manual Mode (Still Available)
```typescript
// Explicit control for special use cases
await client.startMetricsStreaming('agent1', { interval_seconds: 5 });
const metrics = await client.getCurrentMetrics('agent1');
await client.stopMetricsStreaming('agent1');
```

### Benefits

1. **Resource Efficiency**: Agents only collect metrics when someone is listening
2. **Zero Configuration**: Works automatically without any setup
3. **Multiple Client Support**: Multiple SSE clients can share the same stream
4. **Clean Cleanup**: Proper resource cleanup on browser close or network disconnection
5. **Backward Compatibility**: Manual start/stop endpoints still work
6. **Task Cancellation Integration**: Metrics streaming tasks can be cancelled via the standard task cancellation API

### Logging

The system provides detailed logging for debugging:

```
SSE Client connected: client_12345
Agent agent1 now has 1 SSE clients  
First SSE client connected to agent agent1, starting metrics streaming
Started metrics streaming for agent agent1 (task: task_67890)

SSE Client disconnected: client_12345
Agent agent1 now has 0 SSE clients
Last SSE client disconnected from agent agent1, stopping metrics streaming  
Stopped metrics streaming for agent agent1 (task: task_54321)
```

### Testing

To verify the feature works correctly:

1. Check agent status before any connections: `streaming_active: false`
2. Connect SSE client: `streaming_active` should become `true`
3. Disconnect SSE client: `streaming_active` should become `false`
4. Verify agent logs show streaming start/stop events

### Edge Cases Handled

- **Browser Tab Close**: Automatic cleanup when browser closes SSE connection
- **Network Disconnection**: Proper cleanup when network connection is lost
- **Multiple Clients**: Correct reference counting with multiple simultaneous connections
- **Rapid Connect/Disconnect**: Debouncing prevents unnecessary start/stop cycling
- **Task Cancellation**: Metrics streaming tasks can be cancelled via standard task cancellation API
- **Server Restart**: Proper cleanup when server restarts or connections are interrupted

## Migration Notes

Existing applications will continue to work without any changes. The automatic streaming management is purely additive and doesn't break existing manual streaming control workflows.

For new applications, the recommended pattern is to use SSE streaming directly without manual start/stop calls, letting the system handle the lifecycle automatically.
