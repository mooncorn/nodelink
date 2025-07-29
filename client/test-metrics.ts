import { MetricsClient, MetricsFormatter } from './metrics-client';
import EventSource from 'eventsource';

// Set up EventSource polyfill for Node.js environment
if (typeof window === 'undefined') {
  (global as any).EventSource = EventSource;
}

const client = new MetricsClient('http://localhost:8080');
const agentId = 'agent1';

async function testMetricsAPI() {
  console.log('🔍 Testing Agent Metrics API');
  console.log('================================\n');

  try {
    // Test 1: Get all agents
    console.log('1. Getting all agents...');
    const agents = await client.getAllAgents();
    console.log(`Found ${Object.keys(agents.agents).length} agents:`);
    Object.entries(agents.agents).forEach(([id, summary]) => {
      console.log(`  - ${id}: streaming=${summary.streaming_active}, metrics=${summary.has_metrics}`);
    });
    console.log();

    // Test 2: Refresh system info
    console.log('2. Refreshing system information...');
    const refreshTask = await client.refreshSystemInfo(agentId);
    console.log(`System info refresh requested (task: ${refreshTask.task_id})`);
    
    // Wait a bit for the request to complete
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Test 3: Get system info
    console.log('\n3. Getting system information...');
    try {
      const systemInfo = await client.getSystemInfo(agentId);
      console.log('System Information:');
      console.log(`  Hostname: ${systemInfo.software.hostname}`);
      console.log(`  OS: ${systemInfo.software.os.name} ${systemInfo.software.os.version}`);
      console.log(`  Architecture: ${systemInfo.hardware.architecture}`);
      console.log(`  CPU: ${systemInfo.hardware.cpu.model} (${systemInfo.hardware.cpu.cores} cores)`);
      console.log(`  Memory: ${MetricsFormatter.formatBytes(systemInfo.hardware.memory.total_bytes)}`);
      console.log(`  Uptime: ${MetricsFormatter.formatUptime(systemInfo.software.uptime_seconds)}`);
      console.log(`  Network Interfaces: ${systemInfo.network_interfaces.length}`);
    } catch (error: any) {
      console.log(`  ❌ System info not available: ${error.message}`);
    }

    // Test 4: Start metrics streaming
    console.log('\n4. Starting metrics streaming...');
    const streamTask = await client.startMetricsStreaming(agentId, {
      interval_seconds: 3,
      metrics: []
    });
    console.log(`Metrics streaming started (task: ${streamTask.task_id})`);
    
    // Wait for metrics to be collected
    await new Promise(resolve => setTimeout(resolve, 5000));

    // Test 5: Get current metrics
    console.log('\n5. Getting current metrics...');
    try {
      const metrics = await client.getCurrentMetrics(agentId);
      console.log('Current Metrics:');
      console.log(`  CPU Usage: ${MetricsFormatter.formatPercentage(metrics.cpu.usage_percent)}`);
      console.log(`  Memory Usage: ${MetricsFormatter.formatPercentage(metrics.memory.usage_percent)} (${MetricsFormatter.formatBytes(metrics.memory.used_bytes)}/${MetricsFormatter.formatBytes(metrics.memory.total_bytes)})`);
      console.log(`  Load Average: ${MetricsFormatter.formatLoadAverage(metrics.load.load1)} ${MetricsFormatter.formatLoadAverage(metrics.load.load5)} ${MetricsFormatter.formatLoadAverage(metrics.load.load15)}`);
      console.log(`  Processes: ${metrics.processes.total_processes} total${metrics.processes.running_processes ? `, ${metrics.processes.running_processes} running` : ''}`);
      if (metrics.disks.length > 0) {
        console.log(`  Disk Usage (${metrics.disks[0].device}): ${MetricsFormatter.formatPercentage(metrics.disks[0].usage_percent)}`);
      }
    } catch (error: any) {
      console.log(`  ❌ Current metrics not available: ${error.message}`);
    }

    // Test 6: Test SSE streaming
    console.log('\n6. Testing real-time metrics streaming (15 seconds)...');
    const eventSource = client.streamMetrics(agentId, 3);
    let metricsCount = 0;
    
    eventSource.onopen = () => {
      console.log('  🔗 SSE connection opened');
    };
    
    // Listen for the specific "metrics" event type
    eventSource.addEventListener('metrics', (event: any) => {
      try {
        const metrics = JSON.parse(event.data);
        metricsCount++;
        console.log(`  📊 Metric #${metricsCount}: CPU=${MetricsFormatter.formatPercentage(metrics.cpu?.usage_percent || 0)}, Memory=${MetricsFormatter.formatPercentage(metrics.memory?.usage_percent || 0)}`);
      } catch (error) {
        console.log(`  📊 Received metrics update (unparseable): ${event.data.substring(0, 100)}...`);
      }
    });

    eventSource.onerror = (error) => {
      console.log(`  ❌ SSE Error:`, error);
    };

    // Stream for 15 seconds to catch more metrics
    await new Promise(resolve => setTimeout(resolve, 15000));
    console.log(`  📈 Received ${metricsCount} metrics during streaming period`);
    eventSource.close();

    // Test 7: Get historical metrics
    console.log('\n7. Getting historical metrics...');
    const endTime = Math.floor(Date.now() / 1000);
    const startTime = endTime - 300; // last 5 minutes
    
    try {
      const history = await client.getHistoricalMetrics(agentId, {
        metrics: ['cpu.usage_percent', 'memory.usage_percent', 'load.load1'],
        startTime,
        endTime,
        maxPoints: 10
      });
      
      // Handle the actual server response structure
      const dataPoints = history.data_points || history.data || [];
      console.log(`Historical metrics (${dataPoints.length} points available, showing last 5):`);
      dataPoints.slice(-5).forEach((point, index) => {
        const time = new Date(point.timestamp * 1000).toLocaleTimeString();
        // Handle both response formats
        const cpuValue = (point as any).values?.['cpu.usage_percent'] || (point as any)['cpu.usage_percent'] || 0;
        const memValue = (point as any).values?.['memory.usage_percent'] || (point as any)['memory.usage_percent'] || 0;
        const loadValue = (point as any).values?.['load.load1'] || (point as any)['load.load1'] || 0;
        console.log(`  ${time}: CPU=${MetricsFormatter.formatPercentage(cpuValue)}, Memory=${MetricsFormatter.formatPercentage(memValue)}, Load=${MetricsFormatter.formatLoadAverage(loadValue)}`);
      });
    } catch (error: any) {
      console.log(`  ❌ Historical metrics not available: ${error.message}`);
    }

    // Test 8: Stop metrics streaming
    console.log('\n8. Stopping metrics streaming...');
    const stopTask = await client.stopMetricsStreaming(agentId);
    console.log(`Metrics streaming stopped (task: ${stopTask.task_id})`);

    console.log('\n✅ All tests completed!');

  } catch (error: any) {
    console.error(`❌ Test failed: ${error.message}`);
    console.error(error.stack);
  }
}

// Run the test
testMetricsAPI().catch(console.error);
