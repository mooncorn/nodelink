package terminal

import (
	"github.com/gin-gonic/gin"
	"github.com/mooncorn/nodelink/server/internal/common"
	"github.com/mooncorn/nodelink/server/internal/sse"
)

// SSEHandler handles SSE streaming for terminal output using the new simplified architecture
type SSEHandler struct {
	terminalHandler   *Handler
	sessionManager    common.TerminalSessionManager
	streamBuilder     *sse.StreamBuilder
	broadcaster       *sse.Broadcaster
}

// NewSSEHandler creates a new simplified SSE handler for terminal streaming
func NewSSEHandler(terminalHandler *Handler, sseManager common.SSEManager) *SSEHandler {
	return &SSEHandler{
		terminalHandler: terminalHandler,
		sessionManager:  terminalHandler.sessionManager,
		streamBuilder:   sse.NewStreamBuilder(sseManager),
		broadcaster:     sse.NewBroadcaster(sseManager),
	}
}

// RegisterRoutes registers SSE routes for terminal streaming
func (h *SSEHandler) RegisterRoutes(router gin.IRouter) {
	router.GET("/terminals/:sessionId/stream", h.handleTerminalStream)
}

// handleTerminalStream handles SSE connections for terminal output
func (h *SSEHandler) handleTerminalStream(c *gin.Context) {
	h.streamBuilder.ForTerminal("").RequireAuth(h.sessionManager).Handle(c)
}

// BroadcastOutput broadcasts terminal output to the session room
func (h *SSEHandler) BroadcastOutput(sessionID string, output []byte) {
	h.broadcaster.TerminalOutput(sessionID, output)
}

// BroadcastStatus broadcasts terminal session status changes
func (h *SSEHandler) BroadcastStatus(sessionID, status, message string) {
	h.broadcaster.TerminalStatus(sessionID, status, message)
}