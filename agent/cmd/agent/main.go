package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mooncorn/nodelink/agent/pkg/grpc"
)

// Set during build time
var (
	Version       = "dev"
	ServerAddress = "localhost:9090"
)

func main() {
	address := flag.String("address", ServerAddress, "gRPC server address")
	version := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *version {
		log.Printf("Nodelink Agent version: %s", Version)
		os.Exit(0)
	}

	// Get agentID from environment variable
	agentID := os.Getenv("AGENT_ID")
	if agentID == "" {
		log.Fatal("AGENT_ID environment variable is required")
	}

	// Get agentToken from environment variable
	agentToken := os.Getenv("AGENT_TOKEN")
	if agentToken == "" {
		log.Fatal("AGENT_TOKEN environment variable is required")
	}

	log.Printf("Starting Agent (version %s)...", Version)

	// Create grpc client
	client, err := grpc.NewStreamClient(*address)
	if err != nil {
		log.Fatalf("Failed to create grpc client: %v", err)
	}
	defer client.Close()

	// Connect to the server
	if err := client.Connect(agentID, agentToken); err != nil {
		log.Fatalf("Failed to connect to grpc server: %v", err)
	}

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Println("Agent is running. Press Ctrl+C to exit.")
	<-c

	log.Println("Agent shutting down...")
}
