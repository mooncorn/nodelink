package grpc

import (
	"fmt"
	"io"
	"log"
	"sync"

	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/pkg/interfaces"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var AllowedAgents map[string]string = map[string]string{
	"agent1": "secret_token1",
	"agent2": "secret_token2",
}

type TaskServer struct {
	pb.UnimplementedAgentServiceServer

	mu           sync.RWMutex
	agents       map[string]pb.AgentService_StreamTasksServer
	respCh       chan<- *pb.TaskResponse
	metricsStore MetricsStore
	eventBus     interfaces.EventBus
}

// MetricsStore interface for registering agents
type MetricsStore interface {
	RegisterAgent(agentID string)
	UnregisterAgent(agentID string)
}

func NewTaskServer(respCh chan<- *pb.TaskResponse, metricsStore MetricsStore, eventBus interfaces.EventBus) *TaskServer {
	server := &TaskServer{
		agents:       make(map[string]pb.AgentService_StreamTasksServer),
		respCh:       respCh,
		metricsStore: metricsStore,
		eventBus:     eventBus,
	}

	// Subscribe to task events
	if eventBus != nil {
		eventBus.Subscribe("task.send", server.handleTaskSendEvent)
	}

	return server
}

func (s *TaskServer) StreamTasks(stream pb.AgentService_StreamTasksServer) error {
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

	// Register agent in metrics store immediately
	if s.metricsStore != nil {
		s.metricsStore.RegisterAgent(agentID)
	}

	// Remove agent when done
	defer func() {
		s.mu.Lock()
		delete(s.agents, agentID)
		s.mu.Unlock()

		// Unregister agent from metrics store
		if s.metricsStore != nil {
			s.metricsStore.UnregisterAgent(agentID)
		}

		log.Printf("agent %s disconnected", agentID)
	}()

	// Listen for incoming task responses from agent
	for {
		task, err := stream.Recv()
		if err == io.EOF {
			log.Printf("Agent %s closed connection gracefully", agentID)
			break
		}
		if err != nil {
			// Check if it's a network error that might be recoverable
			if status.Code(err) == codes.Unavailable ||
				status.Code(err) == codes.DeadlineExceeded ||
				status.Code(err) == codes.Canceled {
				log.Printf("Agent %s connection lost (recoverable): %v", agentID, err)
			} else {
				log.Printf("Agent %s connection error (non-recoverable): %v", agentID, err)
			}
			return err
		}

		log.Printf("Received task from %s: %+v", agentID, task)

		// Send response to task manager via channel
		if s.respCh != nil {
			select {
			case s.respCh <- task:
			default:
				log.Printf("Task response channel full, dropping response for task %s", task.TaskId)
			}
		}
	}

	return nil
}

func (s *TaskServer) Send(task *pb.TaskRequest) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// check if agent is connected
	stream, ok := s.agents[task.AgentId]
	if !ok {
		return fmt.Errorf("agent with this id is not connected: %s", task.AgentId)
	}

	// send task
	err := stream.Send(task)
	if err != nil {
		return fmt.Errorf("failed to send task to agent: %v", err)
	}

	return nil
}

// SendTask implements the TaskSender interface
func (s *TaskServer) SendTask(request *pb.TaskRequest) error {
	return s.Send(request)
}

// handleTaskSendEvent handles task send events from the event bus
func (s *TaskServer) handleTaskSendEvent(event interfaces.Event) {
	taskRequest, ok := event.Data.(*pb.TaskRequest)
	if !ok {
		log.Printf("Invalid task request data in event: %v", event.Data)
		return
	}

	if err := s.Send(taskRequest); err != nil {
		log.Printf("Failed to send task %s to agent %s: %v", taskRequest.TaskId, taskRequest.AgentId, err)

		// Publish a task failure event
		if s.eventBus != nil {
			failureEvent := interfaces.Event{
				Type:      "task.send.failed",
				Data:      map[string]interface{}{"task_id": taskRequest.TaskId, "error": err.Error()},
				Source:    "task_server",
				Timestamp: event.Timestamp,
			}
			s.eventBus.Publish(failureEvent)
		}
	}
}
