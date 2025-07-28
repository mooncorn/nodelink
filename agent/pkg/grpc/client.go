package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	eventstream "github.com/mooncorn/nodelink/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// EventClient handles client-side event streaming
type EventClient struct {
	conn      *grpc.ClientConn
	client    eventstream.EventServiceClient
	stream    eventstream.EventService_StreamEventsClient
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	listeners []EventListener
}

// EventListener defines a function that processes incoming events
type EventListener func(*eventstream.ServerToNodeEvent)

// NewEventClient creates a new event client
func NewEventClient(serverAddr string) (*EventClient, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := eventstream.NewEventServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())

	return &EventClient{
		conn:      conn,
		client:    client,
		ctx:       ctx,
		cancel:    cancel,
		listeners: make([]EventListener, 0),
	}, nil
}

// AddListener adds an event listener that will be called for each received event
func (c *EventClient) AddListener(listener EventListener) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, listener)
}

// Connect establishes the streaming connection
func (c *EventClient) Connect(agentID, agentToken string) error {
	md := metadata.New(map[string]string{
		"agent_id":    agentID,
		"agent_token": agentToken,
	})
	ctx := metadata.NewOutgoingContext(c.ctx, md)

	stream, err := c.client.StreamEvents(ctx)
	if err != nil {
		return err
	}

	c.stream = stream

	// Start listening for incoming events
	go c.listen()

	log.Println("Agent connected to event stream")
	return nil
}

// listen continuously listens for incoming events
func (c *EventClient) listen() {
	for {
		event, err := c.stream.Recv()
		if err == io.EOF {
			log.Println("Event stream ended")
			break
		}
		if err != nil {
			log.Printf("Error receiving event: %v", err)
			break
		}

		log.Printf("Agent received event: %+v", event)

		// Call all listeners
		c.mu.RLock()
		for _, listener := range c.listeners {
			go listener(event)
		}
		c.mu.RUnlock()
	}
}

// SendEvent sends an event to the server
func (c *EventClient) Send(event *eventstream.NodeToServerEvent) error {
	if c.stream == nil {
		return fmt.Errorf("not connected")
	}

	log.Printf("Agent sending event: %+v", event)
	return c.stream.Send(event)
}

// Close closes the connection
func (c *EventClient) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
