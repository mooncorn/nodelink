package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

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
			cmdStr := payload.ShellExecute.Cmd

			cmd := exec.Command("sh", "-c", cmdStr)
			stdout, _ := cmd.StdoutPipe()
			stderr, _ := cmd.StderrPipe()

			if err := cmd.Start(); err != nil {
				log.Printf("Failed to start command: %v", err)
				return
			}

			var seqStdout, seqStderr int

			sendChunk := func(output, typ string, seq int, isFinal bool, exitCode int) {
				data, _ := structpb.NewStruct(map[string]any{
					"output":    output,
					"type":      typ,
					"timestamp": time.Now().UnixNano(),
					"sequence":  seq,
					"is_final":  isFinal,
					"exit_code": exitCode,
				})
				client.Send(&eventstream.NodeToServerEvent{
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
			}

			// Stream stdout
			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := stdout.Read(buf)
					if n > 0 {
						seqStdout++
						sendChunk(string(buf[:n]), "stdout", seqStdout, false, 0)
					}
					if err != nil {
						break
					}
				}
			}()

			// Stream stderr
			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := stderr.Read(buf)
					if n > 0 {
						seqStderr++
						sendChunk(string(buf[:n]), "stderr", seqStderr, false, 0)
					}
					if err != nil {
						break
					}
				}
			}()

			err := cmd.Wait()
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
			}
			// Send final message
			sendChunk("", "stdout", seqStdout+1, true, exitCode)
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
