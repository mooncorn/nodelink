package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	pb "github.com/mooncorn/nodelink/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TaskClient handles client-side task streaming
type TaskClient struct {
	conn      *grpc.ClientConn
	client    pb.AgentServiceClient
	stream    pb.AgentService_StreamTasksClient
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	listeners []TaskListener
}

type TaskListener func(*pb.TaskRequest)

// NewTaskClient creates a new task client
func NewTaskClient(serverAddr string) (*TaskClient, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := pb.NewAgentServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())

	return &TaskClient{
		conn:      conn,
		client:    client,
		ctx:       ctx,
		cancel:    cancel,
		listeners: make([]TaskListener, 0),
	}, nil
}

func (c *TaskClient) AddListener(listener TaskListener) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, listener)
}

// Connect establishes the streaming connection
func (c *TaskClient) Connect(agentID, agentToken string) error {
	md := metadata.New(map[string]string{
		"agent_id":    agentID,
		"agent_token": agentToken,
	})
	ctx := metadata.NewOutgoingContext(c.ctx, md)

	stream, err := c.client.StreamTasks(ctx)
	if err != nil {
		return err
	}

	c.stream = stream

	// Start listening for incoming tasks
	go c.listen()

	log.Println("Agent connected to task stream")
	return nil
}

// listen continuously listens for incoming tasks
func (c *TaskClient) listen() {
	for {
		task, err := c.stream.Recv()
		if err == io.EOF {
			log.Println("task stream ended")
			break
		}
		if err != nil {
			log.Printf("Error receiving task: %v", err)
			break
		}

		log.Printf("Agent received task: %+v", task)

		// Call all listeners
		c.mu.RLock()
		for _, listener := range c.listeners {
			go listener(task)
		}
		c.mu.RUnlock()
	}
}

// Send sends a task response to the server
func (c *TaskClient) Send(task *pb.TaskResponse) error {
	if c.stream == nil {
		return fmt.Errorf("not connected")
	}

	log.Printf("Agent sending task: %+v", task)
	return c.stream.Send(task)
}

// Close closes the connection
func (c *TaskClient) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
