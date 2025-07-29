package grpc

import (
	"fmt"
	"io"
	"log"
	"sync"

	pb "github.com/mooncorn/nodelink/proto"
	"google.golang.org/grpc/metadata"
)

var AllowedAgents map[string]string = map[string]string{
	"agent1": "secret_token1",
	"agent2": "secret_token2",
}

type TaskServer struct {
	pb.UnimplementedAgentServiceServer

	mu     sync.RWMutex
	agents map[string]pb.AgentService_StreamTasksServer
	respCh chan<- *pb.TaskResponse
}

func NewTaskServer(respCh chan<- *pb.TaskResponse) *TaskServer {
	return &TaskServer{
		agents: make(map[string]pb.AgentService_StreamTasksServer),
		respCh: respCh,
	}
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

	// Remove agent when done
	defer func() {
		s.mu.Lock()
		delete(s.agents, agentID)
		s.mu.Unlock()
		log.Printf("agent %s disconnected", agentID)
	}()

	// Listen for incoming task responses from agent
	for {
		task, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error receiving task from %s: %v", agentID, err)
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
