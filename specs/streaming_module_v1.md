# Streaming Module v1.0 - Scalable Real-time Event Streaming

This specification introduces a major refactor of the streaming architecture in the nodelink project to address scalability issues and provide a foundation for future real-time features by completely replacing the current streaming implementation.

## Problem Statement

The current streaming implementation has mixed success:

### What Works Well (Migrate to New System)
1. **Task-Based Streaming**: Shell/Docker task output streaming works perfectly
   - ✅ Demand-driven: Only streams when client is listening
   - ✅ Automatic lifecycle: Starts with task, stops when task completes
   - ✅ Real-time: Line-by-line output streaming
   - ✅ Resource efficient: No waste, tied to actual work
   - 🔄 **Migrate**: Keep exact same functionality in new unified system

### Current Architecture Issues (Complete Replacement)
1. **Periodic Streaming Waste**: Metrics stream continuously regardless of client interest
2. **Poor Disconnection Handling**: Client disconnections don't trigger agent streaming stops
3. **Limited Scalability**: Each periodic feature implements its own streaming logic
4. **Mixed Patterns**: Task streams vs periodic streams handled differently
5. **No Unified Model**: Different streaming types use different APIs
6. **Scattered Implementation**: SSE logic spread across multiple packages

### Migration Goals

```
OLD SYSTEM (Remove Completely):
- server/pkg/metrics/sse_handler.go → DELETE
- server/pkg/sse/middleware.go → DELETE  
- Direct SSE endpoints → REPLACE
- Manual streaming lifecycle → REPLACE

NEW SYSTEM (Complete Replacement):
- Unified streaming module
- Single subscription API
- Automatic lifecycle management
- Pluggable stream handlers
- Server-side processing capabilities
```

## New Architecture Overview

The new Streaming Module v1.0 **completely replaces** the existing streaming infrastructure with a unified, scalable solution.

### Core Principles

1. **Preserve Task Streaming**: Keep existing task-based streaming exactly as-is
2. **Fix Periodic Streaming**: Make periodic streams demand-driven like tasks
3. **Unified Subscription Model**: Single API for all stream types
4. **Automatic Lifecycle**: No manual start/stop required for any stream type
5. **Event-Driven Architecture**: Reactive to subscription and task changes
6. **Resource Efficiency**: Zero waste for any stream type

### Stream Type Categories

```go
type StreamCategory int

const (
    TaskBasedStream    StreamCategory = iota  // Tied to task lifecycle
    PeriodicStream                           // Continuous until unsubscribed  
    ResourceStream                           // Tied to external resource lifecycle
    EventBasedStream                         // Triggered by events
    InternalStream                           // Server-internal processing only
)

type StreamDestination int

const (
    ClientDestination  StreamDestination = iota  // Stream to external clients
    ServerDestination                            // Process internally on server
    HybridDestination                           // Both client and server processing
)

type StreamTypeInfo struct {
    Name          string
    Category      StreamCategory
    Destination   StreamDestination  // NEW: Where the stream data goes
    Description   string
    AutoStart     bool    // Whether to auto-start when subscribed
    AutoStop      bool    // Whether to auto-stop when last client disconnects
    ResourceBased bool    // Whether stream is tied to specific resource
    MultiInstance bool    // Whether multiple instances can exist per agent
    Buffered      bool    // Whether to buffer events for late subscribers
    ProcessOnServer bool  // Whether server processes events internally
}

var StreamTypes = map[string]StreamTypeInfo{
    "shell_output": {
        Name:          "shell_output",
        Category:      TaskBasedStream,
        Destination:   ClientDestination,
        Description:   "Real-time shell command output",
        AutoStart:     false,  // Started by task creation
        AutoStop:      false,  // Stopped by task completion
        ResourceBased: true,   // Tied to task_id
        MultiInstance: true,   // Multiple tasks per agent
        Buffered:      true,   // Buffer output for late subscribers
        ProcessOnServer: false,
    },
    "docker_output": {
        Name:          "docker_output", 
        Category:      TaskBasedStream,
        Description:   "Real-time Docker operation output",
        AutoStart:     false,
        AutoStop:      false,
        ResourceBased: true,   // Tied to task_id
        MultiInstance: true,
    },
    "container_logs": {
        Name:          "container_logs",
        Category:      ResourceStream,
        Destination:   ClientDestination,
        Description:   "Real-time container log output",
        AutoStart:     true,   // Start when first client subscribes
        AutoStop:      true,   // Stop when last client unsubscribes
        ResourceBased: true,   // Tied to container_id
        MultiInstance: true,   // Multiple containers per agent
        Buffered:      false,
        ProcessOnServer: false,
    },
    "metrics": {
        Name:          "metrics",
        Category:      PeriodicStream,
        Destination:   HybridDestination,  // Both client viewing and server processing
        Description:   "System metrics (CPU, memory, disk, network)",
        AutoStart:     true,
        AutoStop:      true,
        ResourceBased: false,  // Agent-wide
        MultiInstance: false,  // One metrics stream per agent
        Buffered:      false,
        ProcessOnServer: true,  // Server stores metrics, triggers alerts, etc.
    },
    "health_check": {
        Name:          "health_check",
        Category:      PeriodicStream,
        Destination:   ServerDestination,  // NEW: Server-only processing
        Description:   "Agent health monitoring",
        AutoStart:     true,   // Auto-start when agent connects
        AutoStop:      true,   // Stop when agent disconnects
        ResourceBased: false,
        MultiInstance: false,
        Buffered:      false,
        ProcessOnServer: true,  // Server tracks health, no client access
    },
    "system_events": {
        Name:          "system_events",
        Category:      EventBasedStream,
        Destination:   ServerDestination,
        Description:   "System events for server processing",
        AutoStart:     true,
        AutoStop:      true,
        ResourceBased: false,
        MultiInstance: false,
        Buffered:      true,   // Buffer for processing reliability
        ProcessOnServer: true,
    },
    "log_aggregation": {
        Name:          "log_aggregation",
        Category:      PeriodicStream,
        Destination:   ServerDestination,
        Description:   "Application logs for server-side aggregation",
        AutoStart:     true,
        AutoStop:      true,
        ResourceBased: false,
        MultiInstance: false,
        Buffered:      true,
        ProcessOnServer: true,
    },
    "processed_metrics": {
        Name:          "processed_metrics",
        Category:      PeriodicStream,
        Destination:   HybridDestination,
        Description:   "Server-processed and enriched metrics data",
        AutoStart:     true,
        AutoStop:      true,
        ResourceBased: false,
        MultiInstance: false,
        Buffered:      true,    // Buffer processed results
        ProcessOnServer: true,  // Server processes raw data first
    },
    "alert_events": {
        Name:          "alert_events",
        Category:      EventBasedStream,
        Destination:   HybridDestination,
        Description:   "Server-generated alerts relayed to clients",
        AutoStart:     true,
        AutoStop:      true,
        ResourceBased: false,
        MultiInstance: false,
        Buffered:      true,    // Buffer alerts for client delivery
        ProcessOnServer: true,  // Server generates alerts from raw data
    },
    "deployment_status": {
        Name:          "deployment_status",
        Category:      TaskBasedStream,
        Destination:   HybridDestination,
        Description:   "Server-processed deployment events",
        AutoStart:     false,   // Started by deployment tasks
        AutoStop:      false,   // Stopped by task completion
        ResourceBased: true,    // Tied to deployment_id
        MultiInstance: true,
        Buffered:      true,
        ProcessOnServer: true,  // Server analyzes deployment output
    },
}
```

### High-Level Architecture

```
┌─────────────────┐    Subscribe     ┌──────────────────┐    Demand Signal    ┌──────────────────┐
│     Client      │ ───────────────→ │   Subscription   │ ──────────────────→ │      Agent       │
│   (Web/API)     │                  │     Manager      │                     │   Stream Source  │
└─────────────────┘                  └──────────────────┘                     └──────────────────┘
         ↑                                     │                                        │
         │                            ┌───────▼────────┐                               ▼
         │                            │ Stream Router  │                    ┌──────────────────┐
         │                            │  - Task-based  │                    │   Data Producer  │
         │                            │  - Periodic    │                    │ - Tasks/Metrics  │
         │                            │  - Event-based │                    │ - Logs/Files     │
         │                            └───────┬────────┘                    └──────────────────┘
         │                                    │
         │              Real-time Data        │
         └────────────────────────────────────┘
```

## Core Components

### 1. Enhanced Subscription Manager

Handles both client subscriptions and server-internal processing:

```go
type SubscriptionManager struct {
    subscriptions     map[SubscriptionKey]*Subscription
    taskSubscriptions map[string]*TaskSubscription  // task_id -> subscription
    agentStreams      map[string]*AgentStreamState  // agent_id -> active streams
    serverProcessors  map[string]*ServerProcessor   // NEW: Server-internal processors
    eventBus          *EventBus
    streamRegistry    *StreamRegistry
    taskManager       *tasks.TaskManager
    mu                sync.RWMutex
}

// NEW: Server-internal stream processor
type ServerProcessor struct {
    StreamType   string
    AgentID      string
    Resource     string
    Handler      ServerStreamHandler
    Active       bool
    StartedAt    time.Time
    LastActivity time.Time
    Config       StreamConfig
}

// NEW: Interface for server-side stream processing
type ServerStreamHandler interface {
    ProcessEvent(event StreamEvent) error
    OnStreamStart(agentID, resource string) error
    OnStreamStop(agentID, resource string) error
    GetRequiredConfig() StreamConfig
}
```

### 2. Enhanced Stream Registry with Server Processing

```go
type StreamRegistry struct {
    handlers         map[string]StreamHandler
    serverHandlers   map[string]ServerStreamHandler  // NEW: Server-side handlers
    mu               sync.RWMutex
}

type StreamHandler interface {
    GetStreamType() string
    GetCategory() StreamCategory
    GetDestination() StreamDestination  // NEW: Where stream data goes
    
    // For periodic/event-based streams
    StartPeriodicStream(ctx context.Context, req StreamStartRequest) error
    StopPeriodicStream(ctx context.Context, req StreamStopRequest) error
    
    // For task-based streams  
    HandleTaskStream(ctx context.Context, taskID string, callback func(StreamEvent)) error
    
    // NEW: Server-internal processing
    RequiresServerProcessing() bool
    CreateServerProcessor(agentID, resource string) ServerStreamHandler
    
    // Common
    ValidateSubscription(req SubscriptionRequest) error
    ConfigureStream(config StreamConfig) StreamConfig
}
```

### 3. Server-Internal Stream Processors

```go
// NEW: Health monitoring processor
type HealthCheckProcessor struct {
    agentID      string
    healthStore  *HealthStore
    alertManager *AlertManager
}

func (h *HealthCheckProcessor) ProcessEvent(event StreamEvent) error {
    healthData := event.Data.(*pb.HealthCheckResponse)
    
    // Store health metrics
    h.healthStore.UpdateAgentHealth(h.agentID, healthData)
    
    // Check for alerts
    if healthData.CpuUsage > 90.0 {
        h.alertManager.TriggerAlert("high_cpu", h.agentID, healthData)
    }
    
    if healthData.MemoryUsage > 95.0 {
        h.alertManager.TriggerAlert("high_memory", h.agentID, healthData)
    }
    
    return nil
}

// NEW: Log aggregation processor  
type LogAggregationProcessor struct {
    agentID       string
    logStore      *LogStore
    indexer       *LogIndexer
}

func (l *LogAggregationProcessor) ProcessEvent(event StreamEvent) error {
    logData := event.Data.(*pb.LogEvent)
    
    // Store in searchable log store
    l.logStore.StoreLogEvent(l.agentID, logData)
    
    // Index for search
    l.indexer.IndexLogEvent(logData)
    
    // Check for error patterns
    if logData.Level == "ERROR" {
        l.analyzeErrorPatterns(logData)
    }
    
    return nil
}

// NEW: Metrics processor for server-side analytics
type MetricsProcessor struct {
    agentID         string
    metricsStore    *metrics.MetricsStore
    alertManager    *AlertManager
    analyticsEngine *AnalyticsEngine
}

func (m *MetricsProcessor) ProcessEvent(event StreamEvent) error {
    metricsData := event.Data.(*pb.MetricsDataResponse)
    
    // Store metrics (existing functionality)
    m.metricsStore.UpdateMetrics(m.agentID, metricsData)
    
    // NEW: Server-side analytics
    m.analyticsEngine.ProcessMetrics(m.agentID, metricsData)
    
    // NEW: Predictive alerting
    if prediction := m.analyticsEngine.PredictResourceExhaustion(m.agentID); prediction.Risk > 0.8 {
        m.alertManager.TriggerPredictiveAlert("resource_exhaustion", m.agentID, prediction)
    }
    
    return nil
}

// NEW: Server processor that relays processed events to clients
type ProcessAndRelayHandler struct {
    agentID              string
    subscriptionManager  *SubscriptionManager
    processor            func(StreamEvent) (StreamEvent, bool)  // Process and decide if relay
}

func (p *ProcessAndRelayHandler) ProcessEvent(event StreamEvent) error {
    // Process the raw event
    processedEvent, shouldRelay := p.processor(event)
    
    if shouldRelay {
        // Relay processed event to subscribed clients
        p.subscriptionManager.BroadcastProcessedEvent(processedEvent)
    }
    
    return nil
}

// NEW: Enhanced metrics processor with client relay
type EnhancedMetricsProcessor struct {
    agentID              string
    metricsStore         *metrics.MetricsStore
    alertManager         *AlertManager
    analyticsEngine      *AnalyticsEngine
    subscriptionManager  *SubscriptionManager
}

func (m *EnhancedMetricsProcessor) ProcessEvent(event StreamEvent) error {
    metricsData := event.Data.(*pb.MetricsDataResponse)
    
    // 1. Store raw metrics (existing functionality)
    m.metricsStore.UpdateMetrics(m.agentID, metricsData)
    
    // 2. Server-side processing and enrichment
    enrichedMetrics := m.analyticsEngine.EnrichMetrics(m.agentID, metricsData)
    
    // 3. Generate alerts if needed
    alerts := m.checkAndGenerateAlerts(enrichedMetrics)
    
    // 4. Create processed event for clients
    processedEvent := StreamEvent{
        Type:       "processed_metrics",
        StreamType: "processed_metrics",
        AgentID:    m.agentID,
        Resource:   "",
        Data: &ProcessedMetrics{
            Raw:             metricsData,
            Enriched:        enrichedMetrics,
            Alerts:          alerts,
            Recommendations: m.generateRecommendations(enrichedMetrics),
            Trends:          m.calculateTrends(m.agentID),
        },
        Timestamp: time.Now(),
        Metadata: map[string]interface{}{
            "processing_time": time.Since(event.Timestamp),
            "alert_count":     len(alerts),
        },
    }
    
    // 5. Relay to subscribed clients
    m.subscriptionManager.BroadcastProcessedEvent(processedEvent)
    
    return nil
}

type ProcessedMetrics struct {
    Raw             *pb.MetricsDataResponse `json:"raw"`
    Enriched        *EnrichedMetrics        `json:"enriched"`
    Alerts          []Alert                 `json:"alerts"`
    Recommendations []Recommendation        `json:"recommendations"`
    Trends          *TrendAnalysis          `json:"trends"`
}

// NEW: Deployment monitor that processes and relays status
type DeploymentStatusProcessor struct {
    deploymentID         string
    agentID              string
    subscriptionManager  *SubscriptionManager
    deploymentStore      *DeploymentStore
}

func (d *DeploymentStatusProcessor) ProcessEvent(event StreamEvent) error {
    response := event.Data.(*pb.TaskResponse)
    
    if shellResp := response.GetShellExecute(); shellResp != nil {
        output := shellResp.Stdout + shellResp.Stderr
        
        // Process deployment output
        status := d.analyzeDeploymentOutput(output)
        
        // Update deployment store
        d.deploymentStore.UpdateDeploymentStatus(d.deploymentID, status)
        
        // Create processed event for clients
        processedEvent := StreamEvent{
            Type:       "deployment_status",
            StreamType: "deployment_status",
            AgentID:    d.agentID,
            Resource:   d.deploymentID,
            Data: &DeploymentStatus{
                DeploymentID: d.deploymentID,
                Status:       status.Phase,
                Progress:     status.Progress,
                Message:      status.Message,
                RawOutput:    output,
                Timestamp:    time.Now(),
                Services:     status.Services,
                Health:       status.Health,
            },
            Timestamp: time.Now(),
        }
        
        // Relay to clients watching this deployment
        d.subscriptionManager.BroadcastProcessedEvent(processedEvent)
        
        // Handle completion
        if response.IsFinal {
            finalEvent := StreamEvent{
                Type:       "deployment_complete",
                StreamType: "deployment_status",
                AgentID:    d.agentID,
                Resource:   d.deploymentID,
                Data: &DeploymentCompletion{
                    DeploymentID: d.deploymentID,
                    Success:      status.Success,
                    Duration:     status.Duration,
                    Summary:      status.Summary,
                },
                Timestamp: time.Now(),
            }
            d.subscriptionManager.BroadcastProcessedEvent(finalEvent)
        }
    }
    
    return nil
}
```

### 4. Intelligent Stream Lifecycle Management

```go
// NEW: Handles both client and server stream needs
type StreamLifecycleManager struct {
    subscriptionManager *SubscriptionManager
    streamRegistry      *StreamRegistry
    taskManager         *tasks.TaskManager
}

func (slm *StreamLifecycleManager) EvaluateStreamNeed(streamType, agentID, resource string) StreamNeed {
    streamInfo := StreamTypes[streamType]
    
    // Check client subscriptions
    hasClientSubscriptions := slm.subscriptionManager.HasClientSubscriptions(streamType, agentID, resource)
    
    // Check server processing needs
    needsServerProcessing := streamInfo.ProcessOnServer
    
    return StreamNeed{
        HasClients:     hasClientSubscriptions,
        NeedsServer:    needsServerProcessing,
        ShouldStream:   hasClientSubscriptions || needsServerProcessing,
        Destination:    streamInfo.Destination,
    }
}

type StreamNeed struct {
    HasClients   bool
    NeedsServer  bool
    ShouldStream bool
    Destination  StreamDestination
}

// NEW: Start streaming based on actual need
func (slm *StreamLifecycleManager) StartStreamIfNeeded(streamType, agentID, resource string) error {
    need := slm.EvaluateStreamNeed(streamType, agentID, resource)
    
    if !need.ShouldStream {
        return nil // No need to stream
    }
    
    handler := slm.streamRegistry.GetHandler(streamType)
    if handler == nil {
        return fmt.Errorf("no handler for stream type: %s", streamType)
    }
    
    // Start the stream on agent
    err := handler.StartPeriodicStream(context.Background(), StreamStartRequest{
        StreamType: streamType,
        AgentID:    agentID,
        Resource:   resource,
        Config:     handler.ConfigureStream(StreamConfig{}),
    })
    
    if err != nil {
        return err
    }
    
    // Start server processor if needed
    if need.NeedsServer && handler.RequiresServerProcessing() {
        processor := handler.CreateServerProcessor(agentID, resource)
        slm.subscriptionManager.RegisterServerProcessor(streamType, agentID, resource, processor)
    }
    
    return nil
}
```

## Enhanced API Design

### 1. Server-Internal Stream Management

```http
# NEW: Server-internal stream management (admin only)
POST /admin/streams/server
Content-Type: application/json

{
  "stream_type": "health_check",
  "agent_id": "agent1",
  "config": {
    "interval": "30s",
    "metrics": ["cpu", "memory", "disk"]
  }
}

Response:
{
  "stream_id": "srv_12345",
  "status": "active",
  "processor": "health_check",
  "started_at": "2025-01-29T..."
}
```

### 2. Stream Analytics Endpoints

```http
# NEW: Get processed analytics data
GET /analytics/agents/{agentId}/health

Response:
{
  "agent_id": "agent1",
  "current_health": "healthy",
  "last_check": "2025-01-29T...",
  "metrics": {
    "cpu_usage": 25.4,
    "memory_usage": 65.2,
    "disk_usage": 45.8
  },
  "predictions": {
    "risk_level": "low",
    "estimated_capacity_remaining": "72h"
  },
  "alerts": []
}

# NEW: Get aggregated logs
GET /logs/search?agent_id=agent1&level=ERROR&since=1h

Response:
{
  "logs": [
    {
      "timestamp": "2025-01-29T...",
      "level": "ERROR", 
      "component": "docker",
      "message": "Container failed to start",
      "agent_id": "agent1"
    }
  ],
  "total": 15,
  "page": 1
}
```

## Implementation Examples

### 1. Health Check Stream Handler

```go
type HealthCheckStreamHandler struct {
    taskManager      *tasks.TaskManager
    healthStore      *HealthStore
    alertManager     *AlertManager
}

func (h *HealthCheckStreamHandler) GetStreamType() string {
    return "health_check"
}

func (h *HealthCheckStreamHandler) GetDestination() StreamDestination {
    return ServerDestination  // Server-only processing
}

func (h *HealthCheckStreamHandler) RequiresServerProcessing() bool {
    return true
}

func (h *HealthCheckStreamHandler) CreateServerProcessor(agentID, resource string) ServerStreamHandler {
    return &HealthCheckProcessor{
        agentID:      agentID,
        healthStore:  h.healthStore,
        alertManager: h.alertManager,
    }
}

// Start health monitoring automatically when agent connects
func (h *HealthCheckStreamHandler) StartPeriodicStream(ctx context.Context, req StreamStartRequest) error {
    taskReq := &pb.TaskRequest{
        AgentId: req.AgentID,
        Task: &pb.TaskRequest_HealthCheckRequest{
            HealthCheckRequest: &pb.HealthCheckRequest{
                Action:          pb.HealthCheckRequest_START,
                IntervalSeconds: 30,  // Check every 30 seconds
                IncludeMetrics:  true,
                IncludeServices: true,
            },
        },
    }
    
    _, err := h.taskManager.SendTask(taskReq, 30*time.Second)
    return err
}
```

### 2. Task Execution with Server Processing

```go
// NEW: Execute task with server-side event processing
func (tm *TaskManager) ExecuteTaskWithServerProcessing(req *pb.TaskRequest, processors []ServerStreamHandler) (*Task, error) {
    // Create task normally
    task, err := tm.SendTask(req, 5*time.Minute)
    if err != nil {
        return nil, err
    }
    
    // Register server processors for this task
    for _, processor := range processors {
        tm.streamIntegrator.RegisterTaskProcessor(task.ID, processor)
    }
    
    return task, nil
}

// NEW: Process task response with server handlers
func (tsi *TaskStreamIntegrator) ProcessTaskResponse(response *pb.TaskResponse) {
    // ...existing client streaming code...
    
    // NEW: Process with server handlers
    if processors := tsi.getTaskProcessors(response.TaskId); len(processors) > 0 {
        event := StreamEvent{
            Type:       "task_output",
            StreamType: tsi.getStreamTypeForTask(response),
            AgentID:    response.AgentId,
            Resource:   response.TaskId,
            Data:       response,
            Timestamp:  time.Now(),
        }
        
        // Process on server
        for _, processor := range processors {
            go processor.ProcessEvent(event)  // Async processing
        }
    }
}
```

## Usage Examples

### 1. Server Processing with Client Relay

```typescript
// Client receives server-processed events with enriched data
async function setupEnhancedMonitoring() {
    const client = new StreamingClient('http://localhost:8080');
    
    // Subscribe to processed metrics (not raw metrics)
    const processedSubscription = await client.subscribe({
        stream_type: 'processed_metrics',
        agent_id: 'agent1',
        config: {
            include_trends: true,
            alert_levels: ['warning', 'critical']
        }
    });
    
    processedSubscription.on('processed_metrics', (data) => {
        const processed = data.data;
        
        // Display enriched data
        updateDashboard({
            trends: processed.enriched.trends,
            alerts: processed.alerts,
            recommendations: processed.recommendations
        });
        
        // Handle server-generated alerts
        if (processed.alerts.length > 0) {
            showAlertNotifications(processed.alerts);
        }
        
        // Show predictions
        if (processed.enriched.predicted_issues.length > 0) {
            displayPredictions(processed.enriched.predicted_issues);
        }
    });
}
```

### 2. Deployment Monitoring with Server Analysis

```typescript
async function monitorDeployment(deploymentId: string) {
    const client = new StreamingClient('http://localhost:8080');
    
    // Subscribe to deployment status (server processes raw command output)
    const deploymentSubscription = await client.subscribe({
        stream_type: 'deployment_status',
        resource: deploymentId,
        config: {
            include_raw_output: false,  // Only processed status
            status_updates_only: true
        }
    });
    
    deploymentSubscription.on('deployment_status', (data) => {
        const status = data.data;
        
        // Update deployment UI with processed status
        updateDeploymentProgress({
            progress: status.progress,
            currentPhase: status.message,
            services: status.services,
            health: status.health
        });
    });
    
    deploymentSubscription.on('deployment_complete', (data) => {
        const completion = data.data;
        
        // Show completion summary (processed by server)
        showDeploymentSummary({
            success: completion.success,
            duration: completion.duration,
            summary: completion.summary
        });
    });
}
```

### 3. Alert Processing and Relay

```go
// Server processes raw metrics and generates alerts for clients
type AlertProcessor struct {
    agentID              string
    alertRules           []AlertRule
    subscriptionManager  *SubscriptionManager
}

func (a *AlertProcessor) ProcessEvent(event StreamEvent) error {
    metricsData := event.Data.(*pb.MetricsDataResponse)
    
    // Check alert rules
    alerts := []Alert{}
    for _, rule := range a.alertRules {
        if alert := rule.Evaluate(metricsData); alert != nil {
            alerts = append(alerts, *alert)
        }
    }
    
    // If alerts generated, relay to clients
    if len(alerts) > 0 {
        alertEvent := StreamEvent{
            Type:       "alert_events",
            StreamType: "alert_events",
            AgentID:    a.agentID,
            Data: &AlertCollection{
                AgentID:   a.agentID,
                Alerts:    alerts,
                Timestamp: time.Now(),
                Source:    "metrics_analysis",
            },
            Timestamp: time.Now(),
        }
        
        // Relay to subscribed clients
        a.subscriptionManager.BroadcastProcessedEvent(alertEvent)
    }
    
    return nil
}
```

### 4. Multi-Stage Processing Pipeline

```go
// Example: Log processing pipeline that enriches and relays events
type LogProcessingPipeline struct {
    agentID              string
    logAnalyzer          *LogAnalyzer
    threatDetector       *ThreatDetector
    subscriptionManager  *SubscriptionManager
}

func (l *LogProcessingPipeline) ProcessEvent(event StreamEvent) error {
    logData := event.Data.(*pb.LogEvent)
    
    // Stage 1: Parse and structure logs
    structuredLog := l.logAnalyzer.ParseLog(logData)
    
    // Stage 2: Threat detection
    threats := l.threatDetector.AnalyzeLog(structuredLog)
    
    // Stage 3: Enrichment
    enrichedLog := &EnrichedLogEvent{
        Original:    logData,
        Structured:  structuredLog,
        Threats:     threats,
        Severity:    l.calculateSeverity(structuredLog, threats),
        Category:    l.categorizeLog(structuredLog),
        Actionable:  len(threats) > 0,
    }
    
    // Stage 4: Relay processed event to clients
    processedEvent := StreamEvent{
        Type:       "processed_logs",
        StreamType: "processed_logs",
        AgentID:    l.agentID,
        Data:       enrichedLog,
        Timestamp:  time.Now(),
    }
    
    l.subscriptionManager.BroadcastProcessedEvent(processedEvent)
    
    return nil
}
```

## Benefits of Process-and-Relay Architecture

1. **Intelligent Event Processing**:
   - Server adds context and intelligence to raw agent events
   - Clients receive enriched, actionable data
   - Reduces client-side processing burden

2. **Real-time Analytics**:
   - Server processes events in real-time
   - Generates alerts, predictions, and recommendations
   - Clients get immediate insights

3. **Efficient Bandwidth Usage**:
   - Clients can subscribe to processed events only
   - Reduces unnecessary raw data transmission
   - Smart filtering at server level

4. **Centralized Intelligence**:
   - Server maintains global state and context
   - Cross-agent correlation and analysis
   - Consistent business logic application

5. **Flexible Client Needs**:
   - Clients can choose raw, processed, or both
   - Different clients can get different views
   - Event filtering and routing per client

This architecture enables powerful scenarios like:
- **Smart Dashboards**: Receive only processed insights, not raw data
- **Predictive Monitoring**: Server generates predictions, clients show recommendations
- **Security Monitoring**: Server analyzes logs/events, clients get threat alerts
- **Deployment Orchestration**: Server tracks deployment state, clients show progress
- **Performance Optimization**: Server identifies patterns, clients get optimization suggestions

The system automatically handles the complexity of processing raw agent events and intelligently routing the results to interested clients.

## Migration Strategy - Complete Replacement

### Phase 1: Implement New System (Weeks 1-2)

Replace existing streaming infrastructure entirely:

```go
// NEW: server/pkg/streaming/manager.go (replaces all old SSE code)
type StreamingManager struct {
    subscriptionManager *SubscriptionManager
    connectionManager   *ConnectionManager
    streamRegistry      *StreamRegistry
    eventBus           *EventBus
    taskIntegrator     *TaskStreamIntegrator
}

// NEW: server/pkg/streaming/subscription_manager.go
// ...existing code...

// NEW: server/pkg/streaming/handlers/ (directory)
// - metrics.go (COMPLETELY REPLACES server/pkg/metrics/sse_handler.go)
// - shell_output.go (migrates task SSE functionality)
// - docker_output.go (migrates docker task functionality)
// - container_logs.go (new feature using protobuf extensions)
// - health_check.go (new feature using protobuf extensions)
```

### Phase 2: Complete Metrics Migration (Week 2-3)

**Completely Replace Metrics Streaming Implementation**:

#### Remove Old Metrics SSE Code
```go
// DELETE: server/pkg/metrics/sse_handler.go (entire file)
// DELETE: All references to old metrics SSE system

// The old system had these problems:
// - Direct SSE management per agent
// - Manual client reference counting  
// - Scattered lifecycle management
// - No server-side processing capabilities
```

#### New Unified Metrics Handler
```go
// NEW: server/pkg/streaming/handlers/metrics.go
package handlers

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"

    pb "github.com/mooncorn/nodelink/proto"
    "github.com/mooncorn/nodelink/server/pkg/streaming"
    "github.com/mooncorn/nodelink/server/pkg/tasks"
    "github.com/mooncorn/nodelink/server/pkg/metrics"
)

type MetricsStreamHandler struct {
    taskManager         *tasks.TaskManager
    metricsStore       *metrics.MetricsStore
    subscriptionManager *streaming.SubscriptionManager
    
    // Track active streams per agent (replaces old client counter)
    activeStreams map[string]*MetricsStream
    mu           sync.RWMutex
}

type MetricsStream struct {
    AgentID       string
    ClientCount   int
    TaskID        string    // Task ID of the metrics streaming task
    Config        streaming.StreamConfig
    StartedAt     time.Time
}

func NewMetricsStreamHandler(taskManager *tasks.TaskManager, store *metrics.MetricsStore, subscriptionManager *streaming.SubscriptionManager) *MetricsStreamHandler {
    return &MetricsStreamHandler{
        taskManager:         taskManager,
        metricsStore:       store,
        subscriptionManager: subscriptionManager,
        activeStreams:      make(map[string]*MetricsStream),
    }
}

func (h *MetricsStreamHandler) GetStreamType() string {
    return "metrics"
}

func (h *MetricsStreamHandler) GetCategory() streaming.StreamCategory {
    return streaming.PeriodicStream
}

func (h *MetricsStreamHandler) GetDestination() streaming.StreamDestination {
    return streaming.HybridDestination // Both client viewing and server processing
}

// REPLACES: Old automatic streaming start logic
func (h *MetricsStreamHandler) StartPeriodicStream(ctx context.Context, req streaming.StreamStartRequest) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    stream, exists := h.activeStreams[req.AgentID]
    if exists {
        // Stream already active, just increment client count
        stream.ClientCount++
        log.Printf("Added client to existing metrics stream for agent %s (total clients: %d)", req.AgentID, stream.ClientCount)
        return nil
    }
    
    // Start new metrics stream using EXISTING protobuf messages
    intervalSeconds := uint32(5) // default
    if req.Config.Interval > 0 {
        intervalSeconds = uint32(req.Config.Interval.Seconds())
    }
    
    taskReq := &pb.TaskRequest{
        AgentId: req.AgentID,
        Task: &pb.TaskRequest_MetricsRequest{
            MetricsRequest: &pb.MetricsRequest{
                RequestType: &pb.MetricsRequest_StreamRequest{
                    StreamRequest: &pb.MetricsStreamRequest{
                        Action:          pb.MetricsStreamRequest_START,
                        IntervalSeconds: intervalSeconds,
                        Metrics:         req.Config.Filters, // Use existing filter support
                    },
                },
            },
        },
    }
    
    task, err := h.taskManager.SendTask(taskReq, 5*time.Minute)
    if err != nil {
        return fmt.Errorf("failed to start metrics streaming: %w", err)
    }
    
    // Track the stream
    h.activeStreams[req.AgentID] = &MetricsStream{
        AgentID:     req.AgentID,
        ClientCount: 1,
        TaskID:      task.ID,
        Config:      req.Config,
        StartedAt:   time.Now(),
    }
    
    // Update metrics store status
    h.metricsStore.SetStreamingStatus(req.AgentID, true)
    
    log.Printf("Started metrics streaming for agent %s (task: %s)", req.AgentID, task.ID)
    return nil
}

// REPLACES: Old automatic streaming stop logic
func (h *MetricsStreamHandler) StopPeriodicStream(ctx context.Context, req streaming.StreamStopRequest) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    stream, exists := h.activeStreams[req.AgentID]
    if !exists {
        log.Printf("No active metrics stream for agent %s", req.AgentID)
        return nil
    }
    
    stream.ClientCount--
    log.Printf("Removed client from metrics stream for agent %s (remaining clients: %d)", req.AgentID, stream.ClientCount)
    
    if stream.ClientCount <= 0 {
        // Stop the stream - last client disconnected
        taskReq := &pb.TaskRequest{
            AgentId: req.AgentID,
            Task: &pb.TaskRequest_MetricsRequest{
                MetricsRequest: &pb.MetricsRequest{
                    RequestType: &pb.MetricsRequest_StreamRequest{
                        StreamRequest: &pb.MetricsStreamRequest{
                            Action: pb.MetricsStreamRequest_STOP,
                        },
                    },
                },
            },
        }
        
        _, err := h.taskManager.SendTask(taskReq, 30*time.Second)
        if err != nil {
            log.Printf("Failed to stop metrics streaming for agent %s: %v", req.AgentID, err)
        }
        
        // Clean up tracking
        delete(h.activeStreams, req.AgentID)
        h.metricsStore.SetStreamingStatus(req.AgentID, false)
        
        log.Printf("Stopped metrics streaming for agent %s (task: %s)", req.AgentID, stream.TaskID)
    }
    
    return nil
}

func (h *MetricsStreamHandler) RequiresServerProcessing() bool {
    return true // Metrics need server-side storage and processing
}

func (h *MetricsStreamHandler) CreateServerProcessor(agentID, resource string) streaming.ServerStreamHandler {
    return &MetricsServerProcessor{
        agentID:              agentID,
        metricsStore:        h.metricsStore,
        subscriptionManager: h.subscriptionManager,
    }
}

// REPLACES: Old metrics processing logic
type MetricsServerProcessor struct {
    agentID              string
    metricsStore        *metrics.MetricsStore
    subscriptionManager *streaming.SubscriptionManager
}

func (p *MetricsServerProcessor) ProcessEvent(event streaming.StreamEvent) error {
    response := event.Data.(*pb.TaskResponse)
    
    if metricsResp := response.GetMetricsResponse(); metricsResp != nil {
        switch data := metricsResp.ResponseType.(type) {
        case *pb.MetricsResponse_SystemInfo:
            // Store system info (existing functionality)
            p.metricsStore.UpdateSystemInfo(p.agentID, data.SystemInfo)
            
        case *pb.MetricsResponse_MetricsData:
            // Store metrics data (existing functionality)
            p.metricsStore.UpdateMetrics(p.agentID, data.MetricsData)
            
            // NEW: Broadcast to subscribed clients via unified system
            streamEvent := streaming.StreamEvent{
                Type:       "metrics_data",
                StreamType: "metrics",
                AgentID:    p.agentID,
                Resource:   "",
                Data:       data.MetricsData,
                Timestamp:  time.Now(),
                Metadata: map[string]interface{}{
                    "metrics_timestamp": data.MetricsData.Timestamp,
                },
            }
            
            p.subscriptionManager.BroadcastEvent(streamEvent)
            
        case *pb.MetricsResponse_QueryResponse:
            log.Printf("Received historical query response for agent %s with %d points",
                p.agentID, len(data.QueryResponse.DataPoints))
        }
    }
    
    return nil
}

func (p *MetricsServerProcessor) OnStreamStart(agentID, resource string) error {
    log.Printf("Metrics server processing started for agent %s", agentID)
    return nil
}

func (p *MetricsServerProcessor) OnStreamStop(agentID, resource string) error {
    log.Printf("Metrics server processing stopped for agent %s", agentID)
    return nil
}

func (p *MetricsServerProcessor) GetRequiredConfig() streaming.StreamConfig {
    return streaming.StreamConfig{
        Interval:   5 * time.Second,
        BufferSize: 100,
    }
}
```

#### Replace HTTP Endpoints
```go
// DELETE: server/pkg/metrics/http_handler.go StreamAgentMetrics method
// DELETE: All direct metrics SSE endpoints

// MODIFY: server/pkg/metrics/http_handler.go
func (h *MetricsHandler) RegisterRoutes(router gin.IRouter, streamingManager *streaming.StreamingManager) {
    // REMOVE: Old direct SSE endpoint
    // router.GET("/agents/:agentId/metrics/stream", h.StreamAgentMetrics)
    
    // KEEP: Non-streaming endpoints unchanged
    router.GET("/agents/:agentId/system", h.GetAgentSystem)
    router.POST("/agents/:agentId/system/refresh", h.RefreshSystemInfo)
    router.GET("/agents/:agentId/metrics", h.GetCurrentMetrics)
    router.GET("/agents/:agentId/metrics/history", h.GetHistoricalMetrics)
    router.POST("/agents/:agentId/metrics/start", h.StartMetricsStreaming)
    router.POST("/agents/:agentId}/metrics/stop", h.StopMetricsStreaming)
    
    // NEW: Redirect old streaming endpoint to new system
    router.GET("/agents/:agentId/metrics/stream", func(c *gin.Context) {
        agentID := c.Param("agentId")
        intervalStr := c.DefaultQuery("interval", "5")
        
        // Create subscription using new system
        subscription, err := streamingManager.Subscribe(streaming.SubscriptionRequest{
            StreamType: "metrics",
            AgentID:    agentID,
            Config: streaming.StreamConfig{
                Interval: time.Duration(parseInterval(intervalStr)) * time.Second,
                Filters:  []string{}, // All metrics
            },
        })
        
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        
        // Handle SSE using new system
        streamingManager.HandleLegacySSE(c, subscription)
    })
}
```

#### Update Client Library
```typescript
// MODIFY: client/metrics-client.ts
export class MetricsClient {
    private streamingClient: StreamingClient;
    
    constructor(baseURL: string) {
        this.streamingClient = new StreamingClient(baseURL);
    }
    
    // REPLACE: Old direct SSE method
    streamMetrics(agentId: string, interval: number = 5): EventSource {
        console.warn('streamMetrics is deprecated. Use subscribeToMetrics instead.');
        
        // Use new unified subscription system internally
        const subscription = this.streamingClient.subscribeToMetrics(agentId, {
            interval: `${interval}s`,
            metrics: [] // All metrics
        });
        
        return subscription.getEventSource();
    }
    
    // NEW: Preferred method using unified system
    async subscribeToMetrics(agentId: string, options?: MetricsSubscriptionOptions): Promise<StreamSubscription> {
        return this.streamingClient.subscribe({
            stream_type: 'metrics',
            agent_id: agentId,
            config: {
                interval: options?.interval || '5s',
                filters: options?.metrics || [],
                buffer_size: options?.bufferSize || 100
            }
        });
    }
    
    // ...existing code... (non-streaming methods remain unchanged)
}
```

### Phase 3: Integration with Task System (Week 4)

**Complete Task System Integration**:
```go
// MODIFY: server/pkg/tasks/taskmgr.go
type TaskManager struct {
    // ...existing code...
    
    // REMOVE: Old streaming integrations
    // streamIntegrator *streaming.TaskStreamIntegrator
    
    // ADD: Unified streaming integration
    streamingManager *streaming.StreamingManager
}

// MODIFY: processResponse method
func (tm *TaskManager) processResponse(taskResponse *pb.TaskResponse) {
    // ...existing code...
    
    // REPLACE: Old per-stream-type handling
    // if tm.streamIntegrator != nil {
    //     tm.streamIntegrator.ProcessTaskResponse(taskResponse)
    // }
    
    // NEW: Unified streaming processing
    if tm.streamingManager != nil {
        tm.streamingManager.ProcessTaskResponse(taskResponse)
    }
    
    // ...existing code...
}
```

### Phase 4: Complete Route Registration (Week 5)

```go
// MODIFY: server/cmd/server/main.go
func setupRoutes(r *gin.Engine, streamingManager *streaming.StreamingManager) {
    // REMOVE: All old streaming routes
    // r.GET("/stream", handlers.HandleStream)
    // r.GET("/agents/:agentId/metrics/stream", metricsHandler.StreamAgentMetrics)
    
    // ADD: New unified routes
    streamingHandler := streaming.NewHTTPHandler(streamingManager)
    
    r.POST("/subscriptions", streamingHandler.CreateSubscription)
    r.GET("/streams/:subscriptionId", streamingHandler.StreamEvents)
    r.DELETE("/subscriptions/:subscriptionId", streamingHandler.DeleteSubscription)
    r.GET("/subscriptions", streamingHandler.ListSubscriptions)
    
    // MODIFY: Metrics routes to use streaming manager
    metricsHandler := metrics.NewHTTPHandler(metricsStore)
    metricsHandler.RegisterRoutes(r.Group("/"), streamingManager) // Pass streaming manager
    
    // ADD: Backward compatibility for transition period
    r.GET("/stream", streamingHandler.LegacyTaskStream)
}
```

## Complete Migration Examples

### 1. Metrics Streaming Migration

**Before (Old System)**:
```typescript
// Old direct SSE approach
const eventSource = new EventSource('/agents/agent1/metrics/stream?interval=5');
eventSource.addEventListener('metrics', (event) => {
    const data = JSON.parse(event.data);
    updateDashboard(data);
});
```

**After (New Unified System)**:
```typescript
// New subscription-based approach
const client = new StreamingClient('http://localhost:8080');
const subscription = await client.subscribeToMetrics('agent1', {
    interval: '5s',
    metrics: ['cpu', 'memory']
});

subscription.on('metrics', (data) => {
    updateDashboard(data.data);
});

// Automatic cleanup
await subscription.unsubscribe();
```

### 2. Server-Side Metrics Processing

**Before (Old System)**:
```go
// Old: Direct metrics storage only
func (h *MetricsSSEHandler) ProcessMetricsResponse(agentID string, response *pb.MetricsResponse) {
    // Only stored metrics, no processing or intelligent routing
    h.store.UpdateMetrics(agentID, response.GetMetricsData())
    h.BroadcastMetrics(agentID, response.GetMetricsData())
}
```

**After (New Unified System)**:
```go
// New: Intelligent processing with server capabilities
func (p *MetricsServerProcessor) ProcessEvent(event streaming.StreamEvent) error {
    response := event.Data.(*pb.TaskResponse)
    metricsData := response.GetMetricsResponse().GetMetricsData()
    
    // 1. Store metrics (existing functionality)
    p.metricsStore.UpdateMetrics(p.agentID, metricsData)
    
    // 2. Server-side analytics (NEW)
    if p.analyticsEngine != nil {
        p.analyticsEngine.ProcessMetrics(p.agentID, metricsData)
    }
    
    // 3. Intelligent routing to clients (NEW)
    streamEvent := streaming.StreamEvent{
        Type:       "metrics_data",
        StreamType: "metrics",
        AgentID:    p.agentID,
        Data:       metricsData,
        Timestamp:  time.Now(),
    }
    
    p.subscriptionManager.BroadcastEvent(streamEvent)
    
    return nil
}
```

## Benefits of Complete Metrics Migration

1. **Unified Architecture**: Metrics streaming now uses the same subscription model as tasks
2. **Automatic Lifecycle**: No manual start/stop - handled by subscription system
3. **Server Processing**: Metrics can be processed server-side before client delivery
4. **Resource Efficiency**: Agents only stream when clients or server need data
5. **Scalability**: Easy to add metrics processing, filtering, and analytics
6. **Consistency**: Same API patterns for all streaming types

The complete migration ensures metrics streaming is fully integrated into the new unified architecture while maintaining all existing functionality and adding powerful new server-side processing capabilities.