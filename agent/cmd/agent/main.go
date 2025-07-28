package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/mooncorn/nodelink/agent/pkg/grpc"
	eventstream "github.com/mooncorn/nodelink/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	agentID := flag.String("agent_id", "", "Agent ID")
	agentToken := flag.String("agent_token", "", "Agent Auth Token")
	flag.Parse()

	log.Println("Starting Agent...")

	// Create event client
	client, err := grpc.NewEventClient("localhost:9090")
	if err != nil {
		log.Fatalf("Failed to create event client: %v", err)
	}
	defer client.Close()

	// Add a simple event listener that logs all events
	client.AddListener(func(event *eventstream.ServerToNodeEvent) {
		// Specific handling for different event types
		switch payload := event.Task.(type) {
		case *eventstream.ServerToNodeEvent_LogMessage:
			log.Printf("Agent received log message: %s", payload.LogMessage.Msg)
		case *eventstream.ServerToNodeEvent_ShellExecute:
			cmd := payload.ShellExecute.Cmd

			// Execute the command
			out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
			output := string(out)
			if err != nil {
				output += "\nError: " + err.Error()
			}

			data, _ := structpb.NewStruct(map[string]interface{}{
				"output": output,
			})

			err = client.Send(&eventstream.NodeToServerEvent{
				Event: &eventstream.NodeToServerEvent_EventResponse{
					EventResponse: &eventstream.EventResponse{
						EventRef: event.EventId,
						Status:   eventstream.EventResponse_SUCCESS,
						Response: &eventstream.EventResponse_Result{
							Result: &eventstream.EventResponseResult{
								Data: data,
							},
						},
					},
				},
			})
			if err != nil {
				log.Printf("\nFailed to send event response: %v", err)
			}
		}
	})

	// Connect to the server
	if err := client.Connect(*agentID, *agentToken); err != nil {
		log.Fatalf("Failed to connect to event stream: %v", err)
	}

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Println("Agent is running. Press Ctrl+C to exit.")
	<-c

	log.Println("Agent shutting down...")
}
