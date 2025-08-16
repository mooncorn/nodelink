package terminal

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
)

// Session represents a terminal session
type Session struct {
	ID         string
	Shell      string
	WorkingDir string
	Env        map[string]string
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	ctx        context.Context
	cancel     context.CancelFunc
	createdAt  time.Time
}

// Manager manages terminal sessions on the agent
type Manager struct {
	sessions    map[string]*Session
	mu          sync.RWMutex
	messageSend func(*pb.AgentMessage) error
}

// NewManager creates a new terminal manager
func NewManager(messageSend func(*pb.AgentMessage) error) *Manager {
	return &Manager{
		sessions:    make(map[string]*Session),
		messageSend: messageSend,
	}
}

// CreateSession creates a new terminal session
func (m *Manager) CreateSession(req *pb.TerminalCreateRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if session already exists
	if _, exists := m.sessions[req.SessionId]; exists {
		m.sendCreateResponse(req.SessionId, false, "session already exists", "")
		return
	}

	// Set default shell if not provided
	shell := req.Shell
	if shell == "" {
		shell = "bash"
		// Check if bash exists, fallback to sh
		if _, err := exec.LookPath("bash"); err != nil {
			shell = "sh"
		}
	}

	// Set working directory
	workingDir := req.WorkingDir
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			m.sendCreateResponse(req.SessionId, false, fmt.Sprintf("failed to get working directory: %v", err), "")
			return
		}
	}

	// Create context for the session
	ctx, cancel := context.WithCancel(context.Background())

	// Create command - start interactive shell
	cmd := exec.CommandContext(ctx, shell)
	cmd.Dir = workingDir

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range req.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Get pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		m.sendCreateResponse(req.SessionId, false, fmt.Sprintf("failed to create stdin pipe: %v", err), "")
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		cancel()
		m.sendCreateResponse(req.SessionId, false, fmt.Sprintf("failed to create stdout pipe: %v", err), "")
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		cancel()
		m.sendCreateResponse(req.SessionId, false, fmt.Sprintf("failed to create stderr pipe: %v", err), "")
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		cancel()
		m.sendCreateResponse(req.SessionId, false, fmt.Sprintf("failed to start shell: %v", err), "")
		return
	}

	// Create session
	session := &Session{
		ID:         req.SessionId,
		Shell:      shell,
		WorkingDir: workingDir,
		Env:        req.Env,
		cmd:        cmd,
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
		ctx:        ctx,
		cancel:     cancel,
		createdAt:  time.Now(),
	}

	m.sessions[req.SessionId] = session

	// Start output streaming goroutines
	go m.streamOutput(session, stdout, false)
	go m.streamOutput(session, stderr, true)

	// Monitor process termination
	go m.monitorSession(session)

	// Send success response
	m.sendCreateResponse(req.SessionId, true, "", shell)

	log.Printf("Terminal session %s created with shell %s", req.SessionId, shell)
}

// ExecuteCommand executes a command in a terminal session
func (m *Manager) ExecuteCommand(req *pb.TerminalCommandRequest) {
	m.mu.RLock()
	session, exists := m.sessions[req.SessionId]
	m.mu.RUnlock()

	if !exists {
		m.sendCommandResponse(req.SessionId, req.CommandId, "", "session not found", true, 1)
		return
	}

	// Write command to stdin
	command := req.Command + "\n"
	if _, err := session.stdin.Write([]byte(command)); err != nil {
		m.sendCommandResponse(req.SessionId, req.CommandId, "", fmt.Sprintf("failed to write command: %v", err), true, 1)
		return
	}

	// Commands in interactive shells don't have a direct way to track completion
	// We send a non-final response to indicate the command was sent
	m.sendCommandResponse(req.SessionId, req.CommandId, "", "", false, 0)
}

// CloseSession closes a terminal session
func (m *Manager) CloseSession(req *pb.TerminalCloseRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[req.SessionId]
	if !exists {
		m.sendCloseResponse(req.SessionId, false, "session not found")
		return
	}

	// Cancel context and close pipes
	session.cancel()
	session.stdin.Close()
	session.stdout.Close()
	session.stderr.Close()

	// Kill the process if still running
	if session.cmd.Process != nil {
		session.cmd.Process.Kill()
	}

	// Remove from sessions
	delete(m.sessions, req.SessionId)

	// Send success response
	m.sendCloseResponse(req.SessionId, true, "")

	log.Printf("Terminal session %s closed", req.SessionId)
}

// streamOutput streams output from stdout or stderr
func (m *Manager) streamOutput(session *Session, reader io.ReadCloser, isStderr bool) {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		output := scanner.Text()

		if isStderr {
			m.sendCommandResponse(session.ID, "", "", output, false, 0)
		} else {
			m.sendCommandResponse(session.ID, "", output, "", false, 0)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading output from session %s: %v", session.ID, err)
	}
}

// monitorSession monitors the session process and cleans up when it exits
func (m *Manager) monitorSession(session *Session) {
	// Wait for process to complete
	err := session.cmd.Wait()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	// Clean up session
	m.mu.Lock()
	delete(m.sessions, session.ID)
	m.mu.Unlock()

	// Send final response
	m.sendCommandResponse(session.ID, "", "", fmt.Sprintf("Session ended with exit code %d", exitCode), true, int32(exitCode))

	log.Printf("Terminal session %s ended with exit code %d", session.ID, exitCode)
}

// sendCreateResponse sends a terminal create response
func (m *Manager) sendCreateResponse(sessionID string, success bool, errorMsg, shell string) {
	response := &pb.TerminalCreateResponse{
		SessionId: sessionID,
		Success:   success,
		Error:     errorMsg,
		Shell:     shell,
	}

	message := &pb.AgentMessage{
		Message: &pb.AgentMessage_TerminalCreateResponse{
			TerminalCreateResponse: response,
		},
	}

	if err := m.messageSend(message); err != nil {
		log.Printf("Failed to send terminal create response: %v", err)
	}
}

// sendCommandResponse sends a terminal command response
func (m *Manager) sendCommandResponse(sessionID, commandID, output, errorMsg string, isFinal bool, exitCode int32) {
	response := &pb.TerminalCommandResponse{
		SessionId: sessionID,
		CommandId: commandID,
		Output:    output,
		Error:     errorMsg,
		IsFinal:   isFinal,
		ExitCode:  exitCode,
	}

	message := &pb.AgentMessage{
		Message: &pb.AgentMessage_TerminalCommandResponse{
			TerminalCommandResponse: response,
		},
	}

	if err := m.messageSend(message); err != nil {
		log.Printf("Failed to send terminal command response: %v", err)
	}
}

// sendCloseResponse sends a terminal close response
func (m *Manager) sendCloseResponse(sessionID string, success bool, errorMsg string) {
	response := &pb.TerminalCloseResponse{
		SessionId: sessionID,
		Success:   success,
		Error:     errorMsg,
	}

	message := &pb.AgentMessage{
		Message: &pb.AgentMessage_TerminalCloseResponse{
			TerminalCloseResponse: response,
		},
	}

	if err := m.messageSend(message); err != nil {
		log.Printf("Failed to send terminal close response: %v", err)
	}
}

// GetSessionStats returns statistics about active sessions
func (m *Manager) GetSessionStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"active_sessions": len(m.sessions),
	}
}

// Cleanup closes all sessions (called during agent shutdown)
func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for sessionID, session := range m.sessions {
		session.cancel()
		session.stdin.Close()
		session.stdout.Close()
		session.stderr.Close()

		if session.cmd.Process != nil {
			session.cmd.Process.Kill()
		}

		log.Printf("Cleaned up terminal session %s", sessionID)
	}

	m.sessions = make(map[string]*Session)
}
