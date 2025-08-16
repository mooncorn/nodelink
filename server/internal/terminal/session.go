package terminal

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mooncorn/nodelink/server/internal/common"
)

// SessionManager manages terminal sessions in memory
type SessionManager struct {
	sessions      map[string]*common.TerminalSession
	userSessions  map[string][]string // userID -> list of session IDs
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// NewSessionManager creates a new terminal session manager
func NewSessionManager() *SessionManager {
	manager := &SessionManager{
		sessions:     make(map[string]*common.TerminalSession),
		userSessions: make(map[string][]string),
		stopCleanup:  make(chan struct{}),
	}

	// Start cleanup routine
	manager.startCleanupRoutine()

	return manager
}

// CreateSession creates a new terminal session
func (sm *SessionManager) CreateSession(userID, agentID, shell, workingDir string, env map[string]string) (*common.TerminalSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if user has reached max sessions
	if len(sm.userSessions[userID]) >= common.MaxTerminalSessionsPerUser {
		return nil, common.ErrMaxTerminalSessionsReached
	}

	// Set default shell if not provided
	if shell == "" {
		shell = common.DefaultTerminalShell
	}

	// Generate unique session ID
	sessionID := uuid.New().String()

	// Create session
	session := &common.TerminalSession{
		SessionID:    sessionID,
		UserID:       userID,
		AgentID:      agentID,
		Shell:        shell,
		WorkingDir:   workingDir,
		Status:       common.TerminalStatusActive,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Env:          env,
	}

	// Store session
	sm.sessions[sessionID] = session
	sm.userSessions[userID] = append(sm.userSessions[userID], sessionID)

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*common.TerminalSession, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, common.ErrTerminalSessionNotFound
	}

	return session, nil
}

// GetUserSessions returns all sessions for a user
func (sm *SessionManager) GetUserSessions(userID string) []*common.TerminalSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessionIDs := sm.userSessions[userID]
	sessions := make([]*common.TerminalSession, 0, len(sessionIDs))

	for _, sessionID := range sessionIDs {
		if session, exists := sm.sessions[sessionID]; exists {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// CloseSession closes and removes a terminal session
func (sm *SessionManager) CloseSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return common.ErrTerminalSessionNotFound
	}

	// Update session status
	session.Status = common.TerminalStatusClosed

	// Remove from maps
	delete(sm.sessions, sessionID)

	// Remove from user sessions
	userSessions := sm.userSessions[session.UserID]
	for i, id := range userSessions {
		if id == sessionID {
			sm.userSessions[session.UserID] = append(userSessions[:i], userSessions[i+1:]...)
			break
		}
	}

	// Clean up empty user entry
	if len(sm.userSessions[session.UserID]) == 0 {
		delete(sm.userSessions, session.UserID)
	}

	return nil
}

// UpdateLastActivity updates the last activity time for a session
func (sm *SessionManager) UpdateLastActivity(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return common.ErrTerminalSessionNotFound
	}

	session.LastActivity = time.Now()
	return nil
}

// CleanupInactiveSessions removes sessions that have been inactive for too long
func (sm *SessionManager) CleanupInactiveSessions(maxInactivity time.Duration) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	cleanedCount := 0
	sessionsToDelete := make([]string, 0)

	// Find sessions to cleanup
	for sessionID, session := range sm.sessions {
		if now.Sub(session.LastActivity) > maxInactivity {
			sessionsToDelete = append(sessionsToDelete, sessionID)
		}
	}

	// Remove sessions
	for _, sessionID := range sessionsToDelete {
		session := sm.sessions[sessionID]
		session.Status = common.TerminalStatusClosed

		delete(sm.sessions, sessionID)

		// Remove from user sessions
		userSessions := sm.userSessions[session.UserID]
		for i, id := range userSessions {
			if id == sessionID {
				sm.userSessions[session.UserID] = append(userSessions[:i], userSessions[i+1:]...)
				break
			}
		}

		// Clean up empty user entry
		if len(sm.userSessions[session.UserID]) == 0 {
			delete(sm.userSessions, session.UserID)
		}

		cleanedCount++
	}

	return cleanedCount
}

// GetSessionStats returns statistics about current sessions
func (sm *SessionManager) GetSessionStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]interface{}{
		"total_sessions": len(sm.sessions),
		"total_users":    len(sm.userSessions),
	}
}

// ValidateSessionAccess checks if a user has access to a session
func (sm *SessionManager) ValidateSessionAccess(sessionID, userID string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return common.ErrTerminalSessionNotFound
	}

	if session.UserID != userID {
		return common.ErrUnauthorizedTerminalAccess
	}

	if session.Status == common.TerminalStatusClosed {
		return common.ErrTerminalSessionClosed
	}

	return nil
}

// startCleanupRoutine starts the background cleanup routine
func (sm *SessionManager) startCleanupRoutine() {
	sm.cleanupTicker = time.NewTicker(common.TerminalCleanupInterval)

	go func() {
		for {
			select {
			case <-sm.cleanupTicker.C:
				cleaned := sm.CleanupInactiveSessions(common.DefaultTerminalTimeout)
				if cleaned > 0 {
					// Log cleanup activity if needed
				}
			case <-sm.stopCleanup:
				sm.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// Stop stops the session manager and cleanup routine
func (sm *SessionManager) Stop() {
	close(sm.stopCleanup)
	if sm.cleanupTicker != nil {
		sm.cleanupTicker.Stop()
	}
}
