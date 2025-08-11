package sse

import (
	"context"
	"testing"
	"time"
)

type TestMessage struct {
	Content string `json:"content"`
	ID      int    `json:"id"`
}

func TestClientOperations(t *testing.T) {
	client := NewClient[TestMessage]("test-client", 10)
	defer client.Close()

	// Test metadata operations
	client.SetMetadata("name", "Test Client")
	client.SetMetadata("type", "test")

	if name, exists := client.GetMetadata("name"); !exists || name != "Test Client" {
		t.Errorf("Expected metadata 'name' to be 'Test Client', got %v", name)
	}

	// Test room operations
	client.JoinRoom("room1")
	client.JoinRoom("room2")

	if !client.IsInRoom("room1") {
		t.Error("Expected client to be in room1")
	}

	rooms := client.GetRooms()
	if len(rooms) != 2 {
		t.Errorf("Expected client to be in 2 rooms, got %d", len(rooms))
	}

	client.LeaveRoom("room1")
	if client.IsInRoom("room1") {
		t.Error("Expected client to not be in room1 after leaving")
	}

	// Test client info
	info := client.GetInfo()
	if info.ID != "test-client" {
		t.Errorf("Expected client ID to be 'test-client', got %s", info.ID)
	}

	if len(info.Metadata) != 2 {
		t.Errorf("Expected 2 metadata items, got %d", len(info.Metadata))
	}
}

func TestManagerOperations(t *testing.T) {
	config := ManagerConfig{
		BufferSize:     10,
		EnableRooms:    true,
		EnableMetadata: true,
	}

	handler := NewDefaultEventHandler[TestMessage](false)
	manager := NewManager(config, handler)
	manager.Start()
	defer manager.Stop()

	// Test adding clients
	client1 := manager.AddClient("client1")
	_ = manager.AddClient("client2") // We'll remove this one

	if manager.GetClientCount() != 2 {
		t.Errorf("Expected 2 clients, got %d", manager.GetClientCount())
	}

	// Test getting client
	retrievedClient, exists := manager.GetClient("client1")
	if !exists || retrievedClient.ID != "client1" {
		t.Error("Failed to retrieve client1")
	}

	// Test removing client
	manager.RemoveClient("client2")
	if manager.GetClientCount() != 1 {
		t.Errorf("Expected 1 client after removal, got %d", manager.GetClientCount())
	}

	// Test room operations
	err := manager.JoinRoom("client1", "testroom")
	if err != nil {
		t.Errorf("Failed to join room: %v", err)
	}

	// Test sending message to client
	testMsg := TestMessage{Content: "Hello", ID: 1}
	err = manager.SendToClient("client1", testMsg, "test")
	if err != nil {
		t.Errorf("Failed to send message to client: %v", err)
	}

	// Test broadcasting
	err = manager.Broadcast(testMsg, "broadcast")
	if err != nil {
		t.Errorf("Failed to broadcast message: %v", err)
	}

	// Test room messaging
	err = manager.SendToRoom("testroom", testMsg, "room")
	if err != nil {
		t.Errorf("Failed to send message to room: %v", err)
	}

	// Verify message reception (with timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	select {
	case msg := <-client1.Channel:
		if msg.Data.Content != "Hello" {
			t.Errorf("Expected message content 'Hello', got '%s'", msg.Data.Content)
		}
	case <-ctx.Done():
		t.Error("Timeout waiting for message")
	}
}

func TestConcurrentOperations(t *testing.T) {
	config := ManagerConfig{
		BufferSize:     100,
		EnableRooms:    true,
		EnableMetadata: true,
	}

	handler := NewDefaultEventHandler[TestMessage](false)
	manager := NewManager(config, handler)
	manager.Start()
	defer manager.Stop()

	// Add multiple clients concurrently
	numClients := 10
	for i := 0; i < numClients; i++ {
		go func(id int) {
			clientID := ClientID("client_" + string(rune(id+'0')))
			manager.AddClient(clientID)
		}(i)
	}

	// Wait a bit for all clients to be added
	time.Sleep(100 * time.Millisecond)

	if manager.GetClientCount() != numClients {
		t.Errorf("Expected %d clients, got %d", numClients, manager.GetClientCount())
	}

	// Send messages concurrently
	for i := 0; i < numClients; i++ {
		go func(id int) {
			testMsg := TestMessage{Content: "Concurrent", ID: id}
			manager.Broadcast(testMsg, "concurrent")
		}(i)
	}

	// Clean up
	for i := 0; i < numClients; i++ {
		clientID := ClientID("client_" + string(rune(i+'0')))
		manager.RemoveClient(clientID)
	}

	if manager.GetClientCount() != 0 {
		t.Errorf("Expected 0 clients after cleanup, got %d", manager.GetClientCount())
	}
}

func TestEventHandling(t *testing.T) {
	var connectCount, disconnectCount, messageCount int

	handler := EventHandler[TestMessage]{
		OnConnect: func(client *Client[TestMessage]) {
			connectCount++
		},
		OnDisconnect: func(client *Client[TestMessage]) {
			disconnectCount++
		},
		OnMessage: func(message Message[TestMessage]) {
			messageCount++
		},
		OnError: func(clientID ClientID, err error) {
			t.Logf("Error for client %s: %v", clientID, err)
		},
	}

	config := ManagerConfig{BufferSize: 10}
	manager := NewManager(config, handler)
	manager.Start()
	defer manager.Stop()

	// Add and remove clients
	_ = manager.AddClient("test") // Just for testing events
	manager.RemoveClient("test")

	// Send a message
	testMsg := TestMessage{Content: "Test", ID: 1}
	manager.Broadcast(testMsg, "test")

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	if connectCount != 1 {
		t.Errorf("Expected 1 connect event, got %d", connectCount)
	}

	if disconnectCount != 1 {
		t.Errorf("Expected 1 disconnect event, got %d", disconnectCount)
	}

	if messageCount != 1 {
		t.Errorf("Expected 1 message event, got %d", messageCount)
	}
}
