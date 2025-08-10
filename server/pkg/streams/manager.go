package streams

import (
	"sync"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/pkg/interfaces"
	"github.com/mooncorn/nodelink/server/pkg/sse"
)

// Manager handles the lifecycle of different stream types
type Manager struct {
	mu            sync.RWMutex
	activeStreams map[string]map[string]*StreamInstance // streamType -> resourceID -> instance
	sseManager    *sse.Manager[*pb.TaskResponse]
	cleanupTicker *time.Ticker
	stopCh        chan struct{}
}

// StreamInstance represents an active stream
type StreamInstance struct {
	StreamType StreamType
	ResourceID string
	CreatedAt  time.Time
	LastUsed   time.Time
	Active     bool
}

// NewManager creates a new stream manager
func NewManager(sseManager *sse.Manager[*pb.TaskResponse]) *Manager {
	manager := &Manager{
		activeStreams: make(map[string]map[string]*StreamInstance),
		sseManager:    sseManager,
		cleanupTicker: time.NewTicker(5 * time.Minute),
		stopCh:        make(chan struct{}),
	}

	go manager.cleanupLoop()
	return manager
}

// CreateStream creates a new stream instance
func (sm *Manager) CreateStream(streamType, resourceID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	streamTypeConfig, exists := GetStreamType(streamType)
	if !exists {
		return interfaces.ErrStreamTypeNotFound
	}

	if sm.activeStreams[streamType] == nil {
		sm.activeStreams[streamType] = make(map[string]*StreamInstance)
	}

	instance := &StreamInstance{
		StreamType: streamTypeConfig,
		ResourceID: resourceID,
		CreatedAt:  time.Now(),
		LastUsed:   time.Now(),
		Active:     true,
	}

	sm.activeStreams[streamType][resourceID] = instance

	// Configure SSE room buffering if needed
	if streamTypeConfig.Buffered {
		roomName := sm.getRoomName(streamType, resourceID)
		sm.sseManager.EnableRoomBuffering(roomName, streamTypeConfig.BufferSize)
	}

	return nil
}

// CloseStream closes a stream instance
func (sm *Manager) CloseStream(streamType, resourceID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if streams, exists := sm.activeStreams[streamType]; exists {
		if instance, exists := streams[resourceID]; exists {
			instance.Active = false
			delete(streams, resourceID)

			// Clean up SSE room
			roomName := sm.getRoomName(streamType, resourceID)
			sm.sseManager.DisableRoomBuffering(roomName)
		}
	}

	return nil
}

// SendToStream sends data to a specific stream
func (sm *Manager) SendToStream(streamType, resourceID string, data interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	streams, exists := sm.activeStreams[streamType]
	if !exists {
		return interfaces.ErrStreamNotFound
	}

	instance, exists := streams[resourceID]
	if !exists || !instance.Active {
		return interfaces.ErrStreamNotFound
	}

	instance.LastUsed = time.Now()

	// Send to SSE room
	roomName := sm.getRoomName(streamType, resourceID)
	response, ok := data.(*pb.TaskResponse)
	if !ok {
		return interfaces.ErrInvalidData
	}

	sm.sseManager.SendToRoom(roomName, response, streamType)
	return nil
}

// GetActiveStreams returns all active streams
func (sm *Manager) GetActiveStreams() map[string]map[string]*StreamInstance {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]map[string]*StreamInstance)
	for streamType, streams := range sm.activeStreams {
		result[streamType] = make(map[string]*StreamInstance)
		for resourceID, instance := range streams {
			if instance.Active {
				result[streamType][resourceID] = instance
			}
		}
	}

	return result
}

// Stop stops the stream manager
func (sm *Manager) Stop() {
	close(sm.stopCh)
	sm.cleanupTicker.Stop()
}

// getRoomName generates a room name for SSE
func (sm *Manager) getRoomName(streamType, resourceID string) string {
	return streamType + "_" + resourceID
}

// cleanupLoop performs periodic cleanup of inactive streams
func (sm *Manager) cleanupLoop() {
	for {
		select {
		case <-sm.cleanupTicker.C:
			sm.performCleanup()
		case <-sm.stopCh:
			return
		}
	}
}

// performCleanup removes inactive streams that should be auto-cleaned
func (sm *Manager) performCleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	cleanupThreshold := 10 * time.Minute

	for streamType, streams := range sm.activeStreams {
		streamTypeConfig, exists := GetStreamType(streamType)
		if !exists || !streamTypeConfig.AutoCleanup {
			continue
		}

		for resourceID, instance := range streams {
			if !instance.Active && now.Sub(instance.LastUsed) > cleanupThreshold {
				delete(streams, resourceID)

				// Clean up SSE room
				roomName := sm.getRoomName(streamType, resourceID)
				sm.sseManager.DisableRoomBuffering(roomName)
			}
		}
	}
}
