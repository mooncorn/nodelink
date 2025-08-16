package command

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
)

// Executor handles command execution on the agent side
type Executor struct {
	maxTimeout time.Duration
}

// NewExecutor creates a new command executor
func NewExecutor(maxTimeout time.Duration) *Executor {
	if maxTimeout == 0 {
		maxTimeout = 5 * time.Minute // Default 5 minutes
	}
	return &Executor{
		maxTimeout: maxTimeout,
	}
}

// Execute runs a command and returns the response
func (e *Executor) Execute(req *pb.CommandRequest) *pb.CommandResponse {
	response := &pb.CommandResponse{
		RequestId: req.RequestId,
	}

	// Validate timeout
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout <= 0 || timeout > e.maxTimeout {
		timeout = e.maxTimeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Prepare command
	var cmd *exec.Cmd
	if len(req.Args) > 0 {
		cmd = exec.CommandContext(ctx, req.Command, req.Args...)
	} else {
		// If no args, treat the command as a shell command
		cmd = exec.CommandContext(ctx, "sh", "-c", req.Command)
	}

	// Set working directory if specified
	if req.WorkingDir != "" {
		cmd.Dir = req.WorkingDir
	}

	// Set environment variables
	if len(req.Env) > 0 {
		env := cmd.Environ()
		for key, value := range req.Env {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	// Execute command and capture output
	stdout, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			response.Timeout = true
			response.Error = "command timed out"
		} else if exitError, ok := err.(*exec.ExitError); ok {
			response.ExitCode = int32(exitError.ExitCode())
			response.Stderr = string(exitError.Stderr)
		} else {
			response.Error = err.Error()
		}
	}

	response.Stdout = strings.TrimSpace(string(stdout))

	return response
}
