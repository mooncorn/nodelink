package command

import (
	"fmt"
	"regexp"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/internal/interfaces"
)

// CommandProcessor handles command execution events
type CommandProcessor struct {
	ansiRegex *regexp.Regexp
}

// NewCommandProcessor creates a new command processor
func NewCommandProcessor() *CommandProcessor {
	// ANSI escape code regex
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)

	return &CommandProcessor{
		ansiRegex: ansiRegex,
	}
}

// ProcessEvent processes command execution events
func (cp *CommandProcessor) ProcessEvent(event *pb.TaskResponse) (*interfaces.ProcessedEvent, error) {
	// Since we're using the new proto structure, we need to adapt this
	// For now, we'll handle CommandStreamResponse events

	// This method might need to be adapted based on how events flow through the system
	// For command streaming, events would typically be CommandStreamResponse messages

	return &interfaces.ProcessedEvent{
		OriginalEvent: event,
		ProcessedData: event, // For now, pass through
		ShouldRelay:   true,
		TargetRoom:    fmt.Sprintf("command_output_%s", event.TaskId),
		EventType:     "command_output",
	}, nil
}

// ProcessCommandStreamResponse processes streaming command responses
func (cp *CommandProcessor) ProcessCommandStreamResponse(response *pb.CommandStreamResponse) *ProcessedCommandOutput {
	// Process command output based on stream type
	var stdout, stderr string

	switch response.Type {
	case pb.CommandStreamResponse_STDOUT:
		stdout = response.Data
	case pb.CommandStreamResponse_STDERR:
		stderr = response.Data
	case pb.CommandStreamResponse_ERROR:
		stderr = response.Data
	}

	// Clean ANSI codes
	cleanStdout := cp.ansiRegex.ReplaceAllString(stdout, "")
	cleanStderr := cp.ansiRegex.ReplaceAllString(stderr, "")

	// Determine if there are errors
	hasErrors := response.Type == pb.CommandStreamResponse_STDERR ||
		response.Type == pb.CommandStreamResponse_ERROR ||
		(response.Type == pb.CommandStreamResponse_EXIT && response.ExitCode != 0)

	var errorType string
	if hasErrors {
		switch response.Type {
		case pb.CommandStreamResponse_STDERR:
			errorType = "stderr"
		case pb.CommandStreamResponse_ERROR:
			errorType = "error"
		case pb.CommandStreamResponse_EXIT:
			if response.ExitCode != 0 {
				errorType = "exit_code"
			}
		}
	}

	// Combine clean output
	cleanOutput := cleanStdout
	if cleanStderr != "" {
		if cleanOutput != "" {
			cleanOutput += "\n"
		}
		cleanOutput += cleanStderr
	}

	return &ProcessedCommandOutput{
		Stdout:      stdout,
		Stderr:      stderr,
		ExitCode:    int(response.ExitCode),
		ErrorType:   errorType,
		CleanOutput: cleanOutput,
		HasErrors:   hasErrors,
		Timestamp:   time.Now(),
		StreamType:  response.Type.String(),
		IsFinal:     response.IsFinal,
	}
}

// GetEventType returns the event type this processor handles
func (cp *CommandProcessor) GetEventType() string {
	return "command_output"
}

// ProcessedCommandOutput represents processed command output
type ProcessedCommandOutput struct {
	Stdout      string    `json:"stdout"`
	Stderr      string    `json:"stderr"`
	ExitCode    int       `json:"exit_code,omitempty"`
	ErrorType   string    `json:"error_type,omitempty"`
	CleanOutput string    `json:"clean_output"` // ANSI codes removed
	HasErrors   bool      `json:"has_errors"`
	Timestamp   time.Time `json:"timestamp"`
	StreamType  string    `json:"stream_type"`
	IsFinal     bool      `json:"is_final"`
}
