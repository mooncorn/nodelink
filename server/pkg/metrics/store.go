package metrics

import (
	"fmt"
	"sync"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
)

// AgentMetrics stores metrics data for a single agent
type AgentMetrics struct {
	AgentID         string
	Connected       bool
	LastSeen        time.Time
	SystemInfo      *pb.SystemInfoResponse
	CurrentMetrics  *pb.MetricsDataResponse
	HistoricalData  *TimeSeriesStore
	LastUpdate      time.Time
	StreamingActive bool
	mu              sync.RWMutex
}

// TimeSeriesStore stores historical metrics data
type TimeSeriesStore struct {
	data      []*pb.MetricsTimePoint
	retention time.Duration
	mu        sync.RWMutex
}

// MetricsStore manages metrics for all agents
type MetricsStore struct {
	agentData map[string]*AgentMetrics
	retention time.Duration
	mu        sync.RWMutex
}

// NewMetricsStore creates a new metrics store
func NewMetricsStore(retention time.Duration) *MetricsStore {
	return &MetricsStore{
		agentData: make(map[string]*AgentMetrics),
		retention: retention,
	}
}

// UpdateSystemInfo updates system information for an agent
func (ms *MetricsStore) UpdateSystemInfo(agentID string, systemInfo *pb.SystemInfoResponse) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	agent := ms.getOrCreateAgent(agentID)
	agent.mu.Lock()
	agent.SystemInfo = systemInfo
	agent.LastUpdate = time.Now()
	agent.mu.Unlock()
}

// UpdateMetrics updates current metrics for an agent
func (ms *MetricsStore) UpdateMetrics(agentID string, metrics *pb.MetricsDataResponse) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	agent := ms.getOrCreateAgent(agentID)
	agent.mu.Lock()
	agent.CurrentMetrics = metrics
	agent.LastUpdate = time.Now()

	// Store in historical data
	if agent.HistoricalData == nil {
		agent.HistoricalData = NewTimeSeriesStore(ms.retention)
	}

	// Convert metrics to time point for historical storage
	timePoint := ms.metricsToTimePoint(metrics)
	agent.HistoricalData.AddPoint(timePoint)

	agent.mu.Unlock()
}

// SetStreamingStatus sets the streaming status for an agent
func (ms *MetricsStore) SetStreamingStatus(agentID string, active bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	agent := ms.getOrCreateAgent(agentID)
	agent.mu.Lock()
	agent.StreamingActive = active
	agent.mu.Unlock()
}

// GetSystemInfo returns system information for an agent
func (ms *MetricsStore) GetSystemInfo(agentID string) (*pb.SystemInfoResponse, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	agent, exists := ms.agentData[agentID]
	if !exists {
		return nil, ErrAgentNotFound
	}

	agent.mu.RLock()
	defer agent.mu.RUnlock()

	if agent.SystemInfo == nil {
		return nil, ErrNoSystemInfo
	}

	return agent.SystemInfo, nil
}

// GetCurrentMetrics returns current metrics for an agent
func (ms *MetricsStore) GetCurrentMetrics(agentID string) (*pb.MetricsDataResponse, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	agent, exists := ms.agentData[agentID]
	if !exists {
		return nil, ErrAgentNotFound
	}

	agent.mu.RLock()
	defer agent.mu.RUnlock()

	if agent.CurrentMetrics == nil {
		return nil, ErrNoMetrics
	}

	return agent.CurrentMetrics, nil
}

// GetHistoricalMetrics returns historical metrics for an agent
func (ms *MetricsStore) GetHistoricalMetrics(agentID string, start, end uint64, maxPoints uint32, metrics []string) (*pb.MetricsQueryResponse, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	agent, exists := ms.agentData[agentID]
	if !exists {
		return nil, ErrAgentNotFound
	}

	agent.mu.RLock()
	defer agent.mu.RUnlock()

	if agent.HistoricalData == nil {
		return &pb.MetricsQueryResponse{
			DataPoints:          []*pb.MetricsTimePoint{},
			QueryStartTimestamp: start,
			QueryEndTimestamp:   end,
			TotalPoints:         0,
			Truncated:           false,
		}, nil
	}

	return agent.HistoricalData.Query(start, end, maxPoints, metrics), nil
}

// GetAllAgents returns a summary of all agents
func (ms *MetricsStore) GetAllAgents() map[string]*AgentSummary {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	summary := make(map[string]*AgentSummary)
	for agentID, agent := range ms.agentData {
		agent.mu.RLock()
		summary[agentID] = &AgentSummary{
			AgentID:         agentID,
			Connected:       agent.Connected,
			LastSeen:        agent.LastSeen,
			LastUpdate:      agent.LastUpdate,
			StreamingActive: agent.StreamingActive,
			HasSystemInfo:   agent.SystemInfo != nil,
			HasMetrics:      agent.CurrentMetrics != nil,
		}
		agent.mu.RUnlock()
	}

	return summary
}

// RegisterAgent registers an agent as connected
func (ms *MetricsStore) RegisterAgent(agentID string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	agent := ms.getOrCreateAgent(agentID)
	agent.mu.Lock()
	defer agent.mu.Unlock()

	agent.Connected = true
	agent.LastSeen = time.Now()
	if agent.LastUpdate.IsZero() {
		agent.LastUpdate = time.Now()
	}
}

// UnregisterAgent marks an agent as disconnected
func (ms *MetricsStore) UnregisterAgent(agentID string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if agent, exists := ms.agentData[agentID]; exists {
		agent.mu.Lock()
		agent.Connected = false
		agent.mu.Unlock()
	}
}

// RemoveAgent removes an agent from the store
func (ms *MetricsStore) RemoveAgent(agentID string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.agentData, agentID)
}

// CleanupOldData removes old data based on retention policy
func (ms *MetricsStore) CleanupOldData() int {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	var cleaned int
	cutoff := time.Now().Add(-ms.retention)

	for agentID, agent := range ms.agentData {
		agent.mu.Lock()

		// Remove agents that haven't updated in retention period
		if agent.LastUpdate.Before(cutoff) {
			delete(ms.agentData, agentID)
			cleaned++
		} else if agent.HistoricalData != nil {
			// Clean old historical data
			cleaned += agent.HistoricalData.Cleanup(cutoff)
		}

		agent.mu.Unlock()
	}

	return cleaned
}

// getOrCreateAgent gets or creates an agent metrics entry
func (ms *MetricsStore) getOrCreateAgent(agentID string) *AgentMetrics {
	agent, exists := ms.agentData[agentID]
	if !exists {
		agent = &AgentMetrics{
			AgentID:         agentID,
			Connected:       false,
			LastSeen:        time.Now(),
			StreamingActive: false,
		}
		ms.agentData[agentID] = agent
	}
	return agent
}

// metricsToTimePoint converts metrics data to a time point for historical storage
func (ms *MetricsStore) metricsToTimePoint(metrics *pb.MetricsDataResponse) *pb.MetricsTimePoint {
	values := make(map[string]float64)

	// CPU metrics
	if metrics.Cpu != nil {
		values["cpu.usage_percent"] = metrics.Cpu.UsagePercent
		values["cpu.user_percent"] = metrics.Cpu.UserPercent
		values["cpu.system_percent"] = metrics.Cpu.SystemPercent
		values["cpu.idle_percent"] = metrics.Cpu.IdlePercent
		values["cpu.iowait_percent"] = metrics.Cpu.IowaitPercent
		values["cpu.temperature_celsius"] = metrics.Cpu.TemperatureCelsius
	}

	// Memory metrics
	if metrics.Memory != nil {
		values["memory.usage_percent"] = metrics.Memory.UsagePercent
		values["memory.total_bytes"] = float64(metrics.Memory.TotalBytes)
		values["memory.used_bytes"] = float64(metrics.Memory.UsedBytes)
		values["memory.available_bytes"] = float64(metrics.Memory.AvailableBytes)
		values["memory.swap_usage_percent"] = metrics.Memory.SwapUsagePercent
	}

	// Load metrics
	if metrics.Load != nil {
		values["load.load1"] = metrics.Load.Load1
		values["load.load5"] = metrics.Load.Load5
		values["load.load15"] = metrics.Load.Load15
	}

	// Disk metrics (aggregated)
	if len(metrics.Disks) > 0 {
		var totalDiskUsage, totalDiskSpace float64
		for _, disk := range metrics.Disks {
			totalDiskUsage += float64(disk.UsedBytes)
			totalDiskSpace += float64(disk.TotalBytes)
		}
		if totalDiskSpace > 0 {
			values["disk.usage_percent"] = (totalDiskUsage / totalDiskSpace) * 100
		}
	}

	// Process metrics
	if metrics.Processes != nil {
		values["processes.total"] = float64(metrics.Processes.TotalProcesses)
		values["processes.running"] = float64(metrics.Processes.RunningProcesses)
		values["processes.sleeping"] = float64(metrics.Processes.SleepingProcesses)
		values["processes.zombie"] = float64(metrics.Processes.ZombieProcesses)
	}

	return &pb.MetricsTimePoint{
		Timestamp: metrics.Timestamp,
		Values:    values,
	}
}

// NewTimeSeriesStore creates a new time series store
func NewTimeSeriesStore(retention time.Duration) *TimeSeriesStore {
	return &TimeSeriesStore{
		data:      make([]*pb.MetricsTimePoint, 0),
		retention: retention,
	}
}

// AddPoint adds a new data point
func (ts *TimeSeriesStore) AddPoint(point *pb.MetricsTimePoint) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.data = append(ts.data, point)

	// Keep data sorted by timestamp
	if len(ts.data) > 1 && ts.data[len(ts.data)-2].Timestamp > point.Timestamp {
		// Simple bubble sort for last element if out of order
		for i := len(ts.data) - 1; i > 0 && ts.data[i-1].Timestamp > ts.data[i].Timestamp; i-- {
			ts.data[i], ts.data[i-1] = ts.data[i-1], ts.data[i]
		}
	}
}

// Query returns data points in the specified time range
func (ts *TimeSeriesStore) Query(start, end uint64, maxPoints uint32, metrics []string) *pb.MetricsQueryResponse {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	var filtered []*pb.MetricsTimePoint
	for _, point := range ts.data {
		if point.Timestamp >= start && point.Timestamp <= end {
			// Filter metrics if specified
			if len(metrics) > 0 {
				filteredPoint := &pb.MetricsTimePoint{
					Timestamp: point.Timestamp,
					Values:    make(map[string]float64),
				}
				for _, metric := range metrics {
					if value, exists := point.Values[metric]; exists {
						filteredPoint.Values[metric] = value
					}
				}
				filtered = append(filtered, filteredPoint)
			} else {
				filtered = append(filtered, point)
			}
		}
	}

	// Limit results if needed
	truncated := false
	if maxPoints > 0 && uint32(len(filtered)) > maxPoints {
		filtered = filtered[:maxPoints]
		truncated = true
	}

	return &pb.MetricsQueryResponse{
		DataPoints:          filtered,
		QueryStartTimestamp: start,
		QueryEndTimestamp:   end,
		TotalPoints:         uint32(len(filtered)),
		Truncated:           truncated,
	}
}

// Cleanup removes old data points
func (ts *TimeSeriesStore) Cleanup(cutoff time.Time) int {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	cutoffUnix := uint64(cutoff.Unix())
	originalLen := len(ts.data)

	// Find first point to keep
	keepIndex := 0
	for i, point := range ts.data {
		if point.Timestamp >= cutoffUnix {
			keepIndex = i
			break
		}
	}

	// Remove old points
	if keepIndex > 0 {
		ts.data = ts.data[keepIndex:]
	}

	return originalLen - len(ts.data)
}

// AgentSummary provides a summary of agent metrics status
type AgentSummary struct {
	AgentID         string    `json:"agent_id"`
	Connected       bool      `json:"connected"`
	LastSeen        time.Time `json:"last_seen"`
	LastUpdate      time.Time `json:"last_update"`
	StreamingActive bool      `json:"streaming_active"`
	HasSystemInfo   bool      `json:"has_system_info"`
	HasMetrics      bool      `json:"has_metrics"`
}

// Common errors
var (
	ErrAgentNotFound = fmt.Errorf("agent not found")
	ErrNoSystemInfo  = fmt.Errorf("no system information available")
	ErrNoMetrics     = fmt.Errorf("no metrics available")
)
