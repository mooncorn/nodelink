package processors

import (
	"fmt"
	"regexp"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/pkg/interfaces"
)

// ShellProcessor handles shell execution events
type ShellProcessor struct {
	ansiRegex *regexp.Regexp
}

// NewShellProcessor creates a new shell processor
func NewShellProcessor() *ShellProcessor {
	// ANSI escape code regex
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)

	return &ShellProcessor{
		ansiRegex: ansiRegex,
	}
}

// ProcessEvent processes shell execution events
func (sp *ShellProcessor) ProcessEvent(event *pb.TaskResponse) (*interfaces.ProcessedEvent, error) {
	shellResp := event.GetShellExecute()
	if shellResp == nil {
		return nil, fmt.Errorf("not a shell response")
	}

	// Process shell output
	processedOutput := sp.processShellOutput(shellResp)

	return &interfaces.ProcessedEvent{
		OriginalEvent: event,
		ProcessedData: processedOutput,
		ShouldRelay:   true,
		TargetRoom:    fmt.Sprintf("shell_output_%s", event.TaskId),
		EventType:     "shell_output",
	}, nil
}

// GetEventType returns the event type this processor handles
func (sp *ShellProcessor) GetEventType() string {
	return "shell_output"
}

// processShellOutput processes shell output data
func (sp *ShellProcessor) processShellOutput(shellResp *pb.ShellExecuteResponse) *ProcessedShellOutput {
	stdout := shellResp.GetStdout()
	stderr := shellResp.GetStderr()

	// Clean ANSI codes
	cleanStdout := sp.ansiRegex.ReplaceAllString(stdout, "")
	cleanStderr := sp.ansiRegex.ReplaceAllString(stderr, "")

	// Determine if there are errors
	hasErrors := stderr != "" || shellResp.GetExitCode() != 0

	var errorType string
	if hasErrors {
		if stderr != "" {
			errorType = "stderr"
		} else if shellResp.GetExitCode() != 0 {
			errorType = "exit_code"
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

	return &ProcessedShellOutput{
		Stdout:      stdout,
		Stderr:      stderr,
		ExitCode:    int(shellResp.GetExitCode()),
		ErrorType:   errorType,
		CleanOutput: cleanOutput,
		HasErrors:   hasErrors,
		Timestamp:   time.Now(),
	}
}

// ProcessedShellOutput represents processed shell output
type ProcessedShellOutput struct {
	Stdout      string    `json:"stdout"`
	Stderr      string    `json:"stderr"`
	ExitCode    int       `json:"exit_code,omitempty"`
	ErrorType   string    `json:"error_type,omitempty"`
	CleanOutput string    `json:"clean_output"` // ANSI codes removed
	HasErrors   bool      `json:"has_errors"`
	Timestamp   time.Time `json:"timestamp"`
}
