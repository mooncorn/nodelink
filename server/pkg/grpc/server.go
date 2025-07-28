package grpc

import (
	"fmt"
	"io"
	"log"
	"sync"

	eventstream "github.com/mooncorn/nodelink/proto"
	"google.golang.org/grpc/metadata"
)

var AllowedAgents map[string]string = map[string]string{
	"agent1": "secret_token1",
}

// EventServer implements the EventService
type EventServer struct {
	eventstream.UnimplementedEventServiceServer
	mu     sync.RWMutex
	agents map[string]eventstream.EventService_StreamEventsServer

	listeners []EventListener
}

// EventListener defines a function that processes incoming events
type EventListener func(*eventstream.NodeToServerEvent)

// NewEventServer creates a new event server
func NewEventServer() *EventServer {
	return &EventServer{
		agents:    make(map[string]eventstream.EventService_StreamEventsServer),
		listeners: make([]EventListener, 0),
	}
}

// AddListener adds an event listener that will be called for each received event
func (s *EventServer) AddListener(listener EventListener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, listener)
}

// StreamEvents implements bidirectional streaming
func (s *EventServer) StreamEvents(stream eventstream.EventService_StreamEventsServer) error {
	// Authenticate agent
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return fmt.Errorf("missing metadata")
	}

	agentIDs := md["agent_id"]
	tokens := md["agent_token"]
	if len(agentIDs) == 0 || len(tokens) == 0 {
		return fmt.Errorf("missing agent credentials")
	}

	agentID := agentIDs[0]
	token := tokens[0]

	expectedToken, allowed := AllowedAgents[agentID]
	if !allowed || token != expectedToken {
		return fmt.Errorf("unauthorized agent")
	}

	s.mu.Lock()
	s.agents[agentID] = stream
	s.mu.Unlock()

	log.Printf("Agent %s connected", agentID)

	// Remove agent when done
	defer func() {
		s.mu.Lock()
		delete(s.agents, agentID)
		s.mu.Unlock()
		log.Printf("agent %s disconnected", agentID)
	}()

	// Listen for incoming events from agent
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error receiving event from %s: %v", agentID, err)
			return err
		}

		log.Printf("Received event from %s: %+v", agentID, event)

		// Call all listeners
		s.mu.RLock()
		for _, listener := range s.listeners {
			go listener(event)
		}
		s.mu.RUnlock()
	}

	return nil
}

// Broadcast sends an event to all connected agents
func (s *EventServer) Broadcast(event *eventstream.ServerToNodeEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for agentId, agentStream := range s.agents {
		go func(id string, c eventstream.EventService_StreamEventsServer) {
			if err := c.Send(event); err != nil {
				log.Printf("Error sending event to agent %s: %v", id, err)
			}
		}(agentId, agentStream)
	}
}
