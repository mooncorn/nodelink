package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	pb "github.com/mooncorn/nodelink/agent/internal/proto"
	"github.com/mooncorn/nodelink/agent/pkg/command"
	"github.com/mooncorn/nodelink/agent/pkg/metrics"
	"github.com/mooncorn/nodelink/agent/pkg/terminal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type StreamClient struct {
	conn              *grpc.ClientConn
	client            pb.AgentServiceClient
	stream            pb.AgentService_StreamCommunicationClient
	ctx               context.Context
	cancel            context.CancelFunc
	agentID           string
	heartbeatTicker   *time.Ticker
	heartbeatInterval time.Duration
	commandExecutor   *command.Executor
	terminalManager   *terminal.Manager
	metricsHandler    *metrics.Handler
}

// NewStreamClient creates a new stream client
func NewStreamClient(serverAddr string, opts ...grpc.DialOption) (*StreamClient, error) {
	// If no dial options provided, determine TLS configuration based on address
	if len(opts) == 0 {
		if isProdAddress(serverAddr) {
			// Use TLS for production
			creds := credentials.NewTLS(&tls.Config{})
			opts = append(opts, grpc.WithTransportCredentials(creds))
			log.Println("Using TLS connection for production address")
		} else {
			// Use insecure connection for localhost/dev
			opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
			log.Println("Using insecure connection for development address")
		}
	}

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		return nil, err
	}

	client := pb.NewAgentServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())

	streamClient := &StreamClient{
		conn:              conn,
		client:            client,
		ctx:               ctx,
		cancel:            cancel,
		heartbeatInterval: 3 * time.Second, // Default 3 seconds
		commandExecutor:   command.NewExecutor(5 * time.Minute),
		metricsHandler:    metrics.NewHandler(),
	}

	// Initialize terminal manager with message sender
	streamClient.terminalManager = terminal.NewManager(streamClient.Send)

	// Set message sender for metrics handler
	streamClient.metricsHandler.SetMessageSender(streamClient)

	return streamClient, nil
}

// SetHeartbeatInterval configures the interval between heartbeats
func (c *StreamClient) SetHeartbeatInterval(interval time.Duration) {
	c.heartbeatInterval = interval
}

// Connect establishes the streaming connection
func (c *StreamClient) Connect(agentID, agentToken string) error {
	md := metadata.New(map[string]string{
		"agent_id":    agentID,
		"agent_token": agentToken,
	})
	ctx := metadata.NewOutgoingContext(c.ctx, md)

	stream, err := c.client.StreamCommunication(ctx)
	if err != nil {
		return err
	}

	c.stream = stream
	c.agentID = agentID

	go c.listen()

	log.Println("Agent connected to communication stream")
	return nil
}

// listen continuously listens for incoming messages from server
func (c *StreamClient) listen() {
	for {
		serverMsg, err := c.stream.Recv()
		if err == io.EOF {
			log.Println("communication stream ended")
			break
		}
		if err != nil {
			log.Printf("Error receiving message: %v", err)
			break
		}

		// Handle different message types
		switch msg := serverMsg.Message.(type) {
		case *pb.ServerMessage_Ping:
			// Handle ping message
			c.sendPong(&pb.Pong{
				Timestamp:     time.Now().UTC().Unix(),
				PingTimestamp: msg.Ping.Timestamp,
			})
		case *pb.ServerMessage_CommandRequest:
			// Handle command request
			c.handleCommandRequest(msg.CommandRequest)
		case *pb.ServerMessage_TerminalCreateRequest:
			// Handle terminal create request
			c.terminalManager.CreateSession(msg.TerminalCreateRequest)
		case *pb.ServerMessage_TerminalCommandRequest:
			// Handle terminal command request
			c.terminalManager.ExecuteCommand(msg.TerminalCommandRequest)
		case *pb.ServerMessage_TerminalCloseRequest:
			// Handle terminal close request
			c.terminalManager.CloseSession(msg.TerminalCloseRequest)
		case *pb.ServerMessage_MetricsRequest:
			// Handle metrics request
			c.metricsHandler.HandleMetricsRequest(msg.MetricsRequest)
		case *pb.ServerMessage_SystemInfoRequest:
			// Handle system info request
			c.metricsHandler.HandleSystemInfoRequest(msg.SystemInfoRequest)
		default:
			log.Printf("Unknown message type received: %T", msg)
		}
	}
}

// handleCommandRequest processes a command request and sends the response
func (c *StreamClient) handleCommandRequest(req *pb.CommandRequest) {
	log.Printf("Executing command: %s", req.Command)

	// Execute command
	response := c.commandExecutor.Execute(req)

	// Send response back to server
	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_CommandResponse{
			CommandResponse: response,
		},
	}

	if err := c.stream.Send(agentMsg); err != nil {
		log.Printf("Error sending command response: %v", err)
	}
}

// sendPong sends a pong response to the server
func (c *StreamClient) sendPong(pong *pb.Pong) {
	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_Pong{
			Pong: pong,
		},
	}

	if err := c.stream.Send(agentMsg); err != nil {
		log.Printf("Error sending pong: %v", err)
	}
}

// Send sends an agent message to the server
func (c *StreamClient) Send(msg *pb.AgentMessage) error {
	if c.stream == nil {
		return fmt.Errorf("not connected")
	}

	return c.stream.Send(msg)
}

// Close closes the connection
func (c *StreamClient) Close() error {
	// Clean up terminal sessions
	if c.terminalManager != nil {
		c.terminalManager.Cleanup()
	}

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

// isProdAddress determines if the given address is a production address that requires TLS
func isProdAddress(address string) bool {
	// Check for localhost or local development addresses
	if strings.HasPrefix(address, "localhost:") ||
		strings.HasPrefix(address, "127.0.0.1:") ||
		strings.HasPrefix(address, "0.0.0.0:") {
		return false
	}

	// Default to TLS for any other external address
	return true
}
