package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mooncorn/nodelink/agent/pkg/grpc"
	eventstream "github.com/mooncorn/nodelink/proto"
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
	client.AddListener(func(event *eventstream.Event) {
		log.Printf("Agent processed event: %+v", event)

		// Specific handling for different event types
		switch payload := event.Payload.(type) {
		case *eventstream.Event_LogMessage:
			log.Printf("Agent received log message: %s", payload.LogMessage.Msg)
		}
	})

	// Connect to the server
	if err := client.Connect(*agentID, *agentToken); err != nil {
		log.Fatalf("Failed to connect to event stream: %v", err)
	}

	// Send demo events periodically
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		counter := 1

		for range ticker.C {
			if err := client.SendEvent(&eventstream.Event{
				Payload: &eventstream.Event_LogMessage{
					LogMessage: &eventstream.LogMessage{
						Msg: fmt.Sprintf("Event from agent to server: %d", counter),
					},
				},
			}); err != nil {
				log.Printf("Failed to publish event: %v", err)
			}
			counter++
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Println("Agent is running. Press Ctrl+C to exit.")
	<-c

	log.Println("Agent shutting down...")
}
