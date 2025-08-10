package processors

import (
	"fmt"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
	"github.com/mooncorn/nodelink/server/pkg/interfaces"
	"github.com/mooncorn/nodelink/server/pkg/metrics"
)

// MetricsProcessor handles metrics events
type MetricsProcessor struct {
	store *metrics.MetricsStore
}

// NewMetricsProcessor creates a new metrics processor
func NewMetricsProcessor(store *metrics.MetricsStore) *MetricsProcessor {
	return &MetricsProcessor{
		store: store,
	}
}

// ProcessEvent processes metrics events
func (mp *MetricsProcessor) ProcessEvent(event *pb.TaskResponse) (*interfaces.ProcessedEvent, error) {
	metricsResp := event.GetMetricsResponse()
	if metricsResp == nil {
		return nil, fmt.Errorf("not a metrics response")
	}

	// Store metrics in database
	if metricsData := metricsResp.GetMetricsData(); metricsData != nil {
		mp.store.UpdateMetrics(event.AgentId, metricsData)
	}
	if systemInfo := metricsResp.GetSystemInfo(); systemInfo != nil {
		mp.store.UpdateSystemInfo(event.AgentId, systemInfo)
	}

	// Calculate aggregated metrics
	aggregated := mp.calculateAggregatedMetrics(event.AgentId, metricsResp)

	return &interfaces.ProcessedEvent{
		OriginalEvent: event,
		ProcessedData: aggregated,
		ShouldRelay:   true,
		TargetRoom:    fmt.Sprintf("metrics_%s", event.AgentId),
		EventType:     "metrics",
	}, nil
}

// GetEventType returns the event type this processor handles
func (mp *MetricsProcessor) GetEventType() string {
	return "metrics"
}

// calculateAggregatedMetrics calculates aggregated metrics and health score
func (mp *MetricsProcessor) calculateAggregatedMetrics(agentID string, current *pb.MetricsResponse) *AggregatedMetrics {
	alerts := mp.checkAlerts(current)
	healthScore := mp.calculateHealthScore(current, alerts)
	trend := mp.calculateTrend(agentID, current)

	return &AggregatedMetrics{
		AgentID:     agentID,
		Current:     current,
		Trend:       trend,
		Alerts:      alerts,
		HealthScore: healthScore,
		LastUpdated: time.Now(),
	}
}

// checkAlerts checks for metric alerts
func (mp *MetricsProcessor) checkAlerts(metrics *pb.MetricsResponse) []MetricAlert {
	var alerts []MetricAlert

	// Check for metrics data response
	if metricsData := metrics.GetMetricsData(); metricsData != nil {
		// CPU usage alert
		if metricsData.Cpu != nil && metricsData.Cpu.UsagePercent > 85.0 {
			alerts = append(alerts, MetricAlert{
				Type:         "cpu_usage",
				Severity:     "warning",
				Message:      "High CPU usage detected",
				Threshold:    85.0,
				CurrentValue: metricsData.Cpu.UsagePercent,
			})
		}

		// Memory usage alert
		if metricsData.Memory != nil && metricsData.Memory.UsagePercent > 90.0 {
			alerts = append(alerts, MetricAlert{
				Type:         "memory_usage",
				Severity:     "critical",
				Message:      "High memory usage detected",
				Threshold:    90.0,
				CurrentValue: metricsData.Memory.UsagePercent,
			})
		}

		// Disk usage alerts
		for _, disk := range metricsData.Disks {
			if disk.UsagePercent > 95.0 {
				alerts = append(alerts, MetricAlert{
					Type:         "disk_usage",
					Severity:     "critical",
					Message:      fmt.Sprintf("High disk usage on %s", disk.Device),
					Threshold:    95.0,
					CurrentValue: disk.UsagePercent,
				})
			}
		}

		// Load average alert
		if metricsData.Load != nil && metricsData.Load.Load1 > 5.0 {
			alerts = append(alerts, MetricAlert{
				Type:         "load_average",
				Severity:     "warning",
				Message:      "High load average detected",
				Threshold:    5.0,
				CurrentValue: metricsData.Load.Load1,
			})
		}
	}

	return alerts
}

// calculateHealthScore calculates a health score (0-100) based on metrics
func (mp *MetricsProcessor) calculateHealthScore(metrics *pb.MetricsResponse, alerts []MetricAlert) int {
	score := 100

	// Deduct points for alerts
	for _, alert := range alerts {
		switch alert.Severity {
		case "critical":
			score -= 20
		case "warning":
			score -= 10
		}
	}

	// Additional scoring based on metrics
	if metricsData := metrics.GetMetricsData(); metricsData != nil {
		// CPU score
		if metricsData.Cpu != nil {
			cpuUsage := metricsData.Cpu.UsagePercent
			if cpuUsage > 70 {
				score -= int((cpuUsage - 70) / 3) // Gradually decrease score
			}
		}

		// Memory score
		if metricsData.Memory != nil {
			memUsage := metricsData.Memory.UsagePercent
			if memUsage > 80 {
				score -= int((memUsage - 80) / 2) // Gradually decrease score
			}
		}
	}

	// Ensure score doesn't go below 0
	if score < 0 {
		score = 0
	}

	return score
}

// calculateTrend calculates the trend of metrics (simplified implementation)
func (mp *MetricsProcessor) calculateTrend(agentID string, current *pb.MetricsResponse) string {
	// This would normally compare with historical data
	// For now, return "stable" as a placeholder
	// TODO: Implement actual trend calculation using historical metrics
	return "stable"
}

// AggregatedMetrics represents aggregated metrics data
type AggregatedMetrics struct {
	AgentID     string              `json:"agent_id"`
	Current     *pb.MetricsResponse `json:"current"`
	Trend       string              `json:"trend"` // "increasing", "decreasing", "stable"
	Alerts      []MetricAlert       `json:"alerts,omitempty"`
	HealthScore int                 `json:"health_score"` // 0-100
	LastUpdated time.Time           `json:"last_updated"`
}

// MetricAlert represents a metric alert
type MetricAlert struct {
	Type         string  `json:"type"`
	Severity     string  `json:"severity"`
	Message      string  `json:"message"`
	Threshold    float64 `json:"threshold"`
	CurrentValue float64 `json:"current_value"`
}
