package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type HeartbeatClient struct {
	conn              *grpc.ClientConn
	client            pb.AgentServiceClient
	stream            pb.AgentService_HeartbeatStreamClient
	ctx               context.Context
	cancel            context.CancelFunc
	mu                sync.RWMutex
	listeners         []HeartbeatListener
	agentID           string
	heartbeatTicker   *time.Ticker
	startTime         time.Time
	heartbeatInterval time.Duration
}

type HeartbeatListener func(*pb.ServerMessage)

// NewHeartbeatClient creates a new heartbeat client
func NewHeartbeatClient(serverAddr string) (*HeartbeatClient, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := pb.NewAgentServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())

	return &HeartbeatClient{
		conn:              conn,
		client:            client,
		ctx:               ctx,
		cancel:            cancel,
		listeners:         make([]HeartbeatListener, 0),
		startTime:         time.Now(),
		heartbeatInterval: 3 * time.Second, // Default 3 seconds
	}, nil
}

func (c *HeartbeatClient) AddListener(listener HeartbeatListener) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, listener)
}

// SetHeartbeatInterval configures the interval between heartbeats
func (c *HeartbeatClient) SetHeartbeatInterval(interval time.Duration) {
	c.heartbeatInterval = interval
}

// Connect establishes the streaming connection
func (c *HeartbeatClient) Connect(agentID, agentToken string) error {
	md := metadata.New(map[string]string{
		"agent_id":    agentID,
		"agent_token": agentToken,
	})
	ctx := metadata.NewOutgoingContext(c.ctx, md)

	stream, err := c.client.HeartbeatStream(ctx)
	if err != nil {
		return err
	}

	c.stream = stream
	c.agentID = agentID

	go c.listen()
	go c.startPeriodicHeartbeat()

	log.Println("Agent connected to heartbeat stream")
	return nil
}

// listen continuously listens for incoming heartbeats
func (c *HeartbeatClient) listen() {
	for {
		heartbeat, err := c.stream.Recv()
		if err == io.EOF {
			log.Println("heartbeat stream ended")
			break
		}
		if err != nil {
			log.Printf("Error receiving heartbeat: %v", err)
			break
		}

		log.Printf("Agent received heartbeat: %+v", heartbeat)

		// Call all listeners
		c.mu.RLock()
		for _, listener := range c.listeners {
			go listener(heartbeat)
		}
		c.mu.RUnlock()
	}
}

// Send sends a heartbeat response to the server
func (c *HeartbeatClient) Send(heartbeat *pb.AgentMessage) error {
	if c.stream == nil {
		return fmt.Errorf("not connected")
	}

	return c.stream.Send(heartbeat)
}

// startPeriodicHeartbeat sends heartbeats to the server at regular intervals
func (c *HeartbeatClient) startPeriodicHeartbeat() {
	c.heartbeatTicker = time.NewTicker(c.heartbeatInterval)
	defer c.heartbeatTicker.Stop()

	for {
		select {
		case <-c.heartbeatTicker.C:
			c.sendHeartbeat()
		case <-c.ctx.Done():
			log.Println("Stopping periodic heartbeat")
			return
		}
	}
}

// sendHeartbeat creates and sends a heartbeat message
func (c *HeartbeatClient) sendHeartbeat() {
	uptime := int64(time.Since(c.startTime).Seconds())

	heartbeat := &pb.AgentMessage{
		AgentId:   c.agentID,
		MessageId: fmt.Sprintf("heartbeat-%d", time.Now().UnixNano()),
		Timestamp: time.Now().Unix(),
		Payload: &pb.AgentMessage_Heartbeat{
			Heartbeat: &pb.AgentHeartbeat{
				Version:       "1.0.0",
				UptimeSeconds: uptime,
				ActiveTasks:   0,
			},
		},
	}

	if err := c.Send(heartbeat); err != nil {
		log.Printf("Failed to send heartbeat: %v", err)
	} else {
		log.Printf("Sent heartbeat - uptime: %d seconds", uptime)
	}
}

// Close closes the connection
func (c *HeartbeatClient) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	if c.heartbeatTicker != nil {
		c.heartbeatTicker.Stop()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
