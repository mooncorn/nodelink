# Generic SSE Connection Manager

A highly generic and reusable Server-Sent Events (SSE) connection manager package for Go, designed to work with any custom data type and support multiple use cases.

## Features

- **Generic Type Support**: Works with any data type using Go generics
- **Room/Group Management**: Organize clients into rooms for targeted messaging
- **Client Metadata**: Attach custom metadata to clients
- **Thread-Safe Operations**: All operations are thread-safe
- **Event Handling**: Customizable event handlers for connection lifecycle
- **Gin Integration**: Built-in middleware for Gin framework
- **Broadcast & Unicast**: Support for both broadcasting and targeted messaging

## Package Structure

```
pkg/
├── types.go      # Core types and interfaces
├── client.go     # Client management functions
├── manager.go    # Main SSE manager implementation
├── middleware.go # Gin middleware and HTTP handlers
└── event.go      # Default event handlers
```

## Quick Start

### 1. Define Your Data Type

```go
type ChatMessage struct {
    Message string `json:"message"`
    From    string `json:"sender"`
    To      string `json:"receiver"`
}
```

### 2. Create and Configure Manager

```go
import "github.com/mooncorn/nodelink/server/pkg"

// Configuration
config := pkg.ManagerConfig{
    BufferSize:     100,  // Channel buffer size
    EnableRooms:    true, // Enable room functionality
    EnableMetadata: true, // Enable client metadata
}

// Event handler
eventHandler := pkg.NewDefaultEventHandler[ChatMessage](true) // Enable logging

// Create manager
manager := pkg.NewManager(config, eventHandler)
manager.Start()
defer manager.Stop()
```

### 3. Set Up Routes

```go
router := gin.Default()

// SSE endpoint
router.GET("/stream", 
    pkg.SSEHeaders(), 
    pkg.SSEConnection(manager), 
    func(c *gin.Context) {
        // Send welcome message
        client, _ := pkg.GetClientFromContext[ChatMessage](c)
        manager.SendToClient(client.ID, ChatMessage{
            Message: "Welcome!",
            From:    "server",
        }, "welcome")
        
        // Handle stream
        pkg.HandleSSEStream[ChatMessage](c)
    })
```

## API Reference

### Manager Methods

#### Basic Operations
- `NewManager[T](config, handler)` - Create new manager
- `Start()` - Start message processing
- `Stop()` - Stop manager and close connections
- `AddClient(clientID)` - Add new client
- `RemoveClient(clientID)` - Remove client
- `GetClient(clientID)` - Get client by ID
- `GetClientCount()` - Get total client count

#### Messaging
- `SendToClient(clientID, data, eventType)` - Send to specific client
- `Broadcast(data, eventType)` - Send to all clients
- `SendToRoom(room, data, eventType)` - Send to room members

#### Room Management
- `JoinRoom(clientID, room)` - Add client to room
- `LeaveRoom(clientID, room)` - Remove client from room

### Client Methods

#### Metadata Management
- `SetMetadata(key, value)` - Set client metadata
- `GetMetadata(key)` - Get client metadata
- `GetInfo()` - Get read-only client info

#### Room Management
- `JoinRoom(room)` - Join a room
- `LeaveRoom(room)` - Leave a room
- `IsInRoom(room)` - Check room membership
- `GetRooms()` - Get all joined rooms

### Middleware Functions

- `SSEHeaders()` - Set SSE headers
- `SSEConnection[T](manager)` - Handle client connections
- `HandleSSEStream[T](c)` - Handle SSE streaming
- `GetClientFromContext[T](c)` - Get client from context
- `GetClientIDFromContext(c)` - Get client ID from context

## Usage Examples

### Send Message to Specific Client

```go
router.POST("/send/:clientID", func(c *gin.Context) {
    clientID := pkg.ClientID(c.Param("clientID"))
    
    var message ChatMessage
    c.ShouldBindJSON(&message)
    
    manager.SendToClient(clientID, message, "message")
    c.JSON(200, gin.H{"status": "sent"})
})
```

### Broadcast to All Clients

```go
router.POST("/broadcast", func(c *gin.Context) {
    var message ChatMessage
    c.ShouldBindJSON(&message)
    
    manager.Broadcast(message, "broadcast")
    c.JSON(200, gin.H{"status": "broadcasted"})
})
```

### Room Operations

```go
// Join room
router.POST("/join/:clientID/:room", func(c *gin.Context) {
    clientID := pkg.ClientID(c.Param("clientID"))
    room := c.Param("room")
    
    manager.JoinRoom(clientID, room)
    c.JSON(200, gin.H{"status": "joined"})
})

// Send to room
router.POST("/room/:room", func(c *gin.Context) {
    room := c.Param("room")
    
    var message ChatMessage
    c.ShouldBindJSON(&message)
    
    manager.SendToRoom(room, message, "room_message")
    c.JSON(200, gin.H{"status": "sent to room"})
})
```

## Client-Side Usage

### JavaScript/HTML Example

```html
<!DOCTYPE html>
<html>
<head>
    <title>SSE Client</title>
</head>
<body>
    <div id="messages"></div>
    
    <script>
        const eventSource = new EventSource('/stream?client_id=my_client');
        
        eventSource.addEventListener('message', function(event) {
            const data = JSON.parse(event.data);
            console.log('Received:', data);
            
            const messageDiv = document.createElement('div');
            messageDiv.textContent = `${data.from}: ${data.message}`;
            document.getElementById('messages').appendChild(messageDiv);
        });
        
        eventSource.addEventListener('welcome', function(event) {
            const data = JSON.parse(event.data);
            console.log('Welcome message:', data);
        });
        
        eventSource.onerror = function(event) {
            console.error('SSE error:', event);
        };
    </script>
</body>
</html>
```

## Configuration Options

### ManagerConfig

```go
type ManagerConfig struct {
    BufferSize     int  // Channel buffer size (default: 100)
    EnableRooms    bool // Enable room functionality
    EnableMetadata bool // Enable client metadata
}
```

### EventHandler

```go
type EventHandler[T any] struct {
    OnConnect    func(client *Client[T])
    OnDisconnect func(client *Client[T])
    OnMessage    func(message Message[T])
    OnError      func(clientID ClientID, err error)
}
```

## Custom Event Handlers

You can create custom event handlers:

```go
handler := pkg.EventHandler[ChatMessage]{
    OnConnect: func(client *pkg.Client[ChatMessage]) {
        log.Printf("New client: %s", client.ID)
        // Add custom logic here
    },
    OnDisconnect: func(client *pkg.Client[ChatMessage]) {
        log.Printf("Client disconnected: %s", client.ID)
        // Cleanup logic here
    },
    OnMessage: func(message pkg.Message[ChatMessage]) {
        log.Printf("Message processed: %+v", message)
    },
    OnError: func(clientID pkg.ClientID, err error) {
        log.Printf("Error for %s: %v", clientID, err)
    },
}
```

## Thread Safety

All operations are thread-safe:
- Client metadata operations use RWMutex
- Manager client/room operations use RWMutex
- Message processing is handled in separate goroutines
- Context cancellation for clean client disconnection

## Error Handling

The package provides comprehensive error handling:
- Client channel overflow detection
- Automatic client cleanup on disconnection
- Configurable error callbacks
- Context-based cancellation

## Best Practices

1. **Buffer Sizing**: Set appropriate buffer sizes based on expected message volume
2. **Room Management**: Use rooms for organizing clients by topics/channels
3. **Metadata**: Store client-specific information in metadata
4. **Error Handling**: Implement custom error handlers for production use
5. **Cleanup**: Always call `manager.Stop()` on application shutdown
6. **Client IDs**: Use meaningful client IDs when possible

## License

This package is part of the nodelink project.
