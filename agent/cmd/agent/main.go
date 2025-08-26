package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mooncorn/nodelink/agent/pkg/grpc"
)

func main() {
	agentID := flag.String("agent_id", "agent1", "Agent ID")
	agentToken := flag.String("agent_token", "secret_token1", "Agent Auth Token")
	address := flag.String("address", "localhost:9090", "gRPC server address")
	version := flag.String("version", "dev", "Print version and exit")
	flag.Parse()

	log.Printf("Starting Agent (version %s)...", *version)

	// Create grpc client
	client, err := grpc.NewStreamClient(*address)
	if err != nil {
		log.Fatalf("Failed to create grpc client: %v", err)
	}
	defer client.Close()

	// Connect to the server
	if err := client.Connect(*agentID, *agentToken); err != nil {
		log.Fatalf("Failed to connect to grpc server: %v", err)
	}

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Println("Agent is running. Press Ctrl+C to exit.")
	<-c

	log.Println("Agent shutting down...")
}
