package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type PingPongClient struct {
	conn              *grpc.ClientConn
	client            pb.AgentServiceClient
	stream            pb.AgentService_StreamPingPongClient
	ctx               context.Context
	cancel            context.CancelFunc
	agentID           string
	heartbeatTicker   *time.Ticker
	heartbeatInterval time.Duration
}

// NewPingPongClient creates a new ping/pong client
func NewPingPongClient(serverAddr string) (*PingPongClient, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := pb.NewAgentServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())

	return &PingPongClient{
		conn:              conn,
		client:            client,
		ctx:               ctx,
		cancel:            cancel,
		heartbeatInterval: 3 * time.Second, // Default 3 seconds
	}, nil
}

// SetHeartbeatInterval configures the interval between heartbeats
func (c *PingPongClient) SetHeartbeatInterval(interval time.Duration) {
	c.heartbeatInterval = interval
}

// Connect establishes the streaming connection
func (c *PingPongClient) Connect(agentID, agentToken string) error {
	md := metadata.New(map[string]string{
		"agent_id":    agentID,
		"agent_token": agentToken,
	})
	ctx := metadata.NewOutgoingContext(c.ctx, md)

	stream, err := c.client.StreamPingPong(ctx)
	if err != nil {
		return err
	}

	c.stream = stream
	c.agentID = agentID

	go c.listen()

	log.Println("Agent connected to ping/pong stream")
	return nil
}

// listen continuously listens for incoming pings
func (c *PingPongClient) listen() {
	for {
		ping, err := c.stream.Recv()
		if err == io.EOF {
			log.Println("ping/pong stream ended")
			break
		}
		if err != nil {
			log.Printf("Error receiving ping: %v", err)
			break
		}

		// Send pong
		c.Send(&pb.Pong{
			Timestamp:     time.Now().UTC().Unix(),
			PingTimestamp: ping.Timestamp,
		})
	}
}

// Send sends a pong response to the server
func (c *PingPongClient) Send(pong *pb.Pong) error {
	if c.stream == nil {
		return fmt.Errorf("not connected")
	}

	return c.stream.Send(pong)
}

// Close closes the connection
func (c *PingPongClient) Close() error {
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
