import { useState, useEffect } from "react"
import { useParams, Link, useSearchParams } from "react-router-dom"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { 
  Activity, 
  HardDrive, 
  MemoryStick, 
  Cpu, 
  Terminal, 
  Command, 
  ArrowLeft,
  Wifi,
  WifiOff
} from "lucide-react"
import { apiService, type Node, type SystemInfo, type SystemMetrics, type StatusChangeEvent } from "@/lib/api"

export default function NodeDetailPage() {
  const { nodeId } = useParams<{ nodeId: string }>()
  const [searchParams] = useSearchParams()
  const defaultTab = searchParams.get('tab') || 'overview'
  const [node, setNode] = useState<Node | null>(null)
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null)
  const [systemMetrics, setSystemMetrics] = useState<SystemMetrics | null>(null)
  const [loading, setLoading] = useState(true)
  const [connectionStatus, setConnectionStatus] = useState<'connected' | 'disconnected' | 'connecting'>('connecting')

  useEffect(() => {
    if (!nodeId) return

    const fetchNodeData = async () => {
      try {
        // Get node details
        const nodeData = await apiService.getNode(nodeId)
        setNode(nodeData)

        // Get system info
        try {
          const info = await apiService.getSystemInfo(nodeId)
          setSystemInfo(info)
        } catch (error) {
          console.warn('Failed to load system info:', error)
        }

        // Get system metrics - metrics are only available via SSE, so we'll skip HTTP fetch
        // Metrics will be populated via SSE connection below
      } catch (error) {
        console.error('Failed to fetch node data:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchNodeData()

    // Set up SSE connection for real-time node updates
    const eventSource = apiService.connectToSpecificAgentEvents(nodeId)
    
    // Set up SSE connection for real-time metrics
    const metricsEventSource = apiService.connectToMetricsStream(nodeId)
    
    if (eventSource) {
      eventSource.onopen = () => {
        setConnectionStatus('connected')
      }

      eventSource.onerror = () => {
        setConnectionStatus('disconnected')
      }

      eventSource.onmessage = (event) => {
        try {
          const parsedData = JSON.parse(event.data)
          
          // Handle status change events for specific agent
          if (parsedData.event === 'status_change') {
            const statusChange: StatusChangeEvent = parsedData.data
            if (statusChange.agent) {
              setNode(statusChange.agent)
            }
          }
        } catch (error) {
          console.error('Error parsing SSE message:', error)
        }
      }
    }
    
    if (metricsEventSource) {
      metricsEventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          console.log('Received metrics data:', data)
          
          // Handle metrics payload
          if (data.metrics) {
            console.log('Setting system metrics:', data.metrics)
            setSystemMetrics(data.metrics)
          }
          
          // Handle system info payload
          if (data.system_info) {
            console.log('Setting system info:', data.system_info)
            setSystemInfo(data.system_info)
          }
        } catch (error) {
          console.error('Error parsing metrics event:', error)
        }
      }

      metricsEventSource.onerror = () => {
        console.warn('Metrics SSE connection error')
      }

      metricsEventSource.onopen = () => {
        console.log('Metrics SSE connection opened')
      }
    }

    // Fallback: refresh data every 30 seconds
    const interval = setInterval(fetchNodeData, 30000)

    // Cleanup function handles both event sources and interval
    return () => {
      eventSource?.close()
      metricsEventSource?.close()
      clearInterval(interval)
    }
  }, [nodeId])

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    
    if (days > 0) {
      return `${days} days, ${hours} hours, ${minutes} minutes`
    } else if (hours > 0) {
      return `${hours} hours, ${minutes} minutes`
    } else {
      return `${minutes} minutes`
    }
  }

  const formatBytes = (bytes: number) => {
    const gb = bytes / (1024 * 1024 * 1024)
    return gb.toFixed(1)
  }

  const formatBytesShort = (bytes: number) => {
    if (bytes >= 1024 * 1024 * 1024) {
      return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
    } else if (bytes >= 1024 * 1024) {
      return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    } else if (bytes >= 1024) {
      return `${(bytes / 1024).toFixed(1)} KB`
    }
    return `${bytes} B`
  }

  const getNodeDisplayName = (node: Node) => {
    return node.metadata?.name || node.agent_id
  }

  const getNodeHost = (node: Node) => {
    return node.metadata?.host || "Unknown host"
  }

  if (loading) {
    return (
      <div className="flex-1 space-y-4 p-4 pt-6">
        <div className="flex items-center space-x-2">
          <Button variant="outline" size="sm" asChild>
            <Link to="/nodes">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div className="h-8 bg-muted rounded w-48 animate-pulse"></div>
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[...Array(3)].map((_, i) => (
            <Card key={i}>
              <CardHeader className="animate-pulse">
                <div className="h-4 bg-muted rounded w-3/4"></div>
              </CardHeader>
              <CardContent>
                <div className="h-8 bg-muted rounded w-1/2 animate-pulse"></div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    )
  }

  if (!node) {
    return (
      <div className="flex-1 space-y-4 p-4 pt-6">
        <div className="flex items-center space-x-2">
          <Button variant="outline" size="sm" asChild>
            <Link to="/nodes">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <h2 className="text-3xl font-bold tracking-tight">Node Not Found</h2>
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1 space-y-4 p-4 pt-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button variant="outline" size="sm" asChild>
            <Link to="/nodes">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h2 className="text-3xl font-bold tracking-tight">{getNodeDisplayName(node)}</h2>
            <p className="text-muted-foreground">{getNodeHost(node)}</p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Badge variant={node.status === "online" ? "default" : "destructive"}>
            {node.status}
          </Badge>
          <div className="flex items-center space-x-1">
            {connectionStatus === 'connected' ? (
              <Wifi className="h-4 w-4 text-green-500" />
            ) : (
              <WifiOff className="h-4 w-4 text-red-500" />
            )}
            <span className="text-sm text-muted-foreground">
              {connectionStatus === 'connected' ? 'Live' : 'Offline'}
            </span>
          </div>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">CPU Usage</CardTitle>
            <Cpu className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {systemMetrics && systemMetrics.cpu_usage_percent != null ? `${systemMetrics.cpu_usage_percent.toFixed(1)}%` : 'N/A'}
            </div>
            {systemMetrics && systemMetrics.cpu_usage_percent != null && (
              <Progress value={systemMetrics.cpu_usage_percent} className="mt-2" />
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Memory Usage</CardTitle>
            <MemoryStick className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {systemMetrics && systemMetrics.memory && systemMetrics.memory.used_percent != null ? `${systemMetrics.memory.used_percent.toFixed(1)}%` : 'N/A'}
            </div>
            {systemMetrics && systemMetrics.memory && systemMetrics.memory.used_percent != null && (
              <>
                <Progress value={systemMetrics.memory.used_percent} className="mt-2" />
                <p className="text-xs text-muted-foreground mt-1">
                  {formatBytesShort(systemMetrics.memory.used)} / {formatBytesShort(systemMetrics.memory.total)}
                </p>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Disk Usage</CardTitle>
            <HardDrive className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {systemMetrics && systemMetrics.disks && systemMetrics.disks.length > 0 && systemMetrics.disks[0].used_percent != null
                ? `${systemMetrics.disks[0].used_percent.toFixed(1)}%`
                : 'N/A'
              }
            </div>
            {systemMetrics && systemMetrics.disks && systemMetrics.disks.length > 0 && systemMetrics.disks[0].used_percent != null && (
              <>
                <Progress value={systemMetrics.disks[0].used_percent} className="mt-2" />
                <p className="text-xs text-muted-foreground mt-1">
                  {formatBytesShort(systemMetrics.disks[0].used)} / {formatBytesShort(systemMetrics.disks[0].total)}
                </p>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Uptime</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-lg font-bold">
              {systemInfo ? formatUptime(systemInfo.uptime_seconds) : 'N/A'}
            </div>
          </CardContent>
        </Card>
      </div>

      <Tabs defaultValue={defaultTab} className="w-full">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="system">System Info</TabsTrigger>
          <TabsTrigger value="actions">Actions</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <div className="grid gap-6 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Node Status</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm font-medium">Agent ID</p>
                    <p className="text-sm text-muted-foreground">{node.agent_id}</p>
                  </div>
                  <div>
                    <p className="text-sm font-medium">Status</p>
                    <Badge variant={node.status === "online" ? "default" : "destructive"}>
                      {node.status}
                    </Badge>
                  </div>
                  <div>
                    <p className="text-sm font-medium">Last Seen</p>
                    <p className="text-sm text-muted-foreground">
                      {new Date(node.last_seen).toLocaleString()}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm font-medium">Connected At</p>
                    <p className="text-sm text-muted-foreground">
                      {node.connected_at ? new Date(node.connected_at).toLocaleString() : 'N/A'}
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>

            {systemInfo && (
              <Card>
                <CardHeader>
                  <CardTitle>System Information</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm font-medium">Hostname</p>
                      <p className="text-sm text-muted-foreground">{systemInfo.hostname}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium">Operating System</p>
                      <p className="text-sm text-muted-foreground">{systemInfo.platform} {systemInfo.os_version}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium">Architecture</p>
                      <p className="text-sm text-muted-foreground">{systemInfo.arch}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium">CPU Cores</p>
                      <p className="text-sm text-muted-foreground">{systemInfo.cpu_count}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </TabsContent>

        <TabsContent value="system" className="space-y-4">
          <Tabs defaultValue="overview" className="w-full">
            <TabsList className="grid w-full grid-cols-4">
              <TabsTrigger value="overview">Overview</TabsTrigger>
              <TabsTrigger value="cpu">CPU</TabsTrigger>
              <TabsTrigger value="memory">Memory</TabsTrigger>
              <TabsTrigger value="storage">Storage</TabsTrigger>
            </TabsList>

            <TabsContent value="overview" className="space-y-4">
              {/* System Information Grid */}
              {systemInfo && (
                <Card>
                  <CardHeader>
                    <CardTitle>System Information</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
                      <div>
                        <p className="text-sm font-medium">Hostname</p>
                        <p className="text-sm text-muted-foreground">{systemInfo.hostname}</p>
                      </div>
                      <div>
                        <p className="text-sm font-medium">Platform</p>
                        <p className="text-sm text-muted-foreground">{systemInfo.platform}</p>
                      </div>
                      <div>
                        <p className="text-sm font-medium">OS Version</p>
                        <p className="text-sm text-muted-foreground">{systemInfo.os_version}</p>
                      </div>
                      <div>
                        <p className="text-sm font-medium">Kernel Version</p>
                        <p className="text-sm text-muted-foreground">{systemInfo.kernel_version}</p>
                      </div>
                      <div>
                        <p className="text-sm font-medium">Architecture</p>
                        <p className="text-sm text-muted-foreground">{systemInfo.arch}</p>
                      </div>
                      <div>
                        <p className="text-sm font-medium">CPU Cores</p>
                        <p className="text-sm text-muted-foreground">{systemInfo.cpu_count}</p>
                      </div>
                      <div>
                        <p className="text-sm font-medium">Total Memory</p>
                        <p className="text-sm text-muted-foreground">{formatBytes(systemInfo.total_memory)} GB</p>
                      </div>
                      <div>
                        <p className="text-sm font-medium">Uptime</p>
                        <p className="text-sm text-muted-foreground">{formatUptime(systemInfo.uptime_seconds)}</p>
                      </div>
                    </div>
                    <div className="mt-4">
                      <p className="text-sm font-medium mb-2">Network Interfaces</p>
                      <div className="flex flex-wrap gap-2">
                        {systemInfo.network_interfaces.map((iface, index) => (
                          <Badge key={index} variant="outline">{iface}</Badge>
                        ))}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              )}
            </TabsContent>

            <TabsContent value="cpu" className="space-y-4">
              {systemMetrics ? (
                <div className="space-y-4">
                  <Card>
                    <CardHeader>
                      <CardTitle>CPU Usage</CardTitle>
                      <CardDescription>Current CPU utilization</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-4">
                        <div>
                          <div className="flex justify-between items-center mb-2">
                            <span className="text-sm font-medium">Overall CPU Usage</span>
                            <span className="text-sm font-medium">{systemMetrics.cpu_usage_percent != null ? systemMetrics.cpu_usage_percent.toFixed(1) : 'N/A'}%</span>
                          </div>
                          <Progress value={systemMetrics.cpu_usage_percent || 0} className="h-4" />
                        </div>
                        <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mt-4">
                          <div>
                            <p className="text-sm font-medium">Load Average (1m)</p>
                            <p className="text-sm text-muted-foreground">{systemMetrics.load_average_1m != null ? systemMetrics.load_average_1m.toFixed(2) : 'N/A'}</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium">Load Average (5m)</p>
                            <p className="text-sm text-muted-foreground">{systemMetrics.load_average_5m != null ? systemMetrics.load_average_5m.toFixed(2) : 'N/A'}</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium">Load Average (15m)</p>
                            <p className="text-sm text-muted-foreground">{systemMetrics.load_average_15m != null ? systemMetrics.load_average_15m.toFixed(2) : 'N/A'}</p>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>

                  {/* Top Processes */}
                  <Card>
                    <CardHeader>
                      <CardTitle>Top Processes</CardTitle>
                      <CardDescription>Processes sorted by CPU usage</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        {systemMetrics.processes
                          .sort((a, b) => b.cpu_percent - a.cpu_percent)
                          .slice(0, 10)
                          .map((process) => (
                            <div key={process.pid} className="flex items-center justify-between p-2 rounded-md bg-muted/50">
                              <div className="flex-1">
                                <div className="flex items-center space-x-2">
                                  <span className="text-sm font-medium">{process.name}</span>
                                  <Badge variant="outline" className="text-xs">PID: {process.pid}</Badge>
                                </div>
                                <p className="text-xs text-muted-foreground">
                                  Memory: {formatBytesShort(process.memory_rss)} | Threads: {process.num_threads}
                                </p>
                              </div>
                              <div className="text-right">
                                <div className="text-sm font-medium">{process.cpu_percent != null ? process.cpu_percent.toFixed(1) : 'N/A'}%</div>
                                <div className="w-16">
                                  <Progress value={Math.min(process.cpu_percent || 0, 100)} className="h-2" />
                                </div>
                              </div>
                            </div>
                          ))}
                      </div>
                    </CardContent>
                  </Card>
                </div>
              ) : (
                <Card>
                  <CardContent className="text-center py-8">
                    <p className="text-sm text-muted-foreground">CPU metrics not available</p>
                  </CardContent>
                </Card>
              )}
            </TabsContent>

            <TabsContent value="memory" className="space-y-4">
              {systemMetrics ? (
                <div className="space-y-4">
                  <Card>
                    <CardHeader>
                      <CardTitle>Memory Usage</CardTitle>
                      <CardDescription>Current memory utilization</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-4">
                        <div>
                          <div className="flex justify-between items-center mb-2">
                            <span className="text-sm font-medium">Used Memory</span>
                            <span className="text-sm font-medium">{systemMetrics.memory && systemMetrics.memory.used_percent != null ? systemMetrics.memory.used_percent.toFixed(1) : 'N/A'}%</span>
                          </div>
                          <Progress value={systemMetrics.memory?.used_percent || 0} className="h-4" />
                          <p className="text-xs text-muted-foreground mt-1">
                            {systemMetrics.memory ? `${formatBytesShort(systemMetrics.memory.used)} of ${formatBytesShort(systemMetrics.memory.total)} used` : 'N/A'}
                          </p>
                        </div>
                        
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                          <div>
                            <p className="text-sm font-medium">Total</p>
                            <p className="text-sm text-muted-foreground">{formatBytesShort(systemMetrics.memory.total)}</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium">Available</p>
                            <p className="text-sm text-muted-foreground">{formatBytesShort(systemMetrics.memory.available)}</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium">Free</p>
                            <p className="text-sm text-muted-foreground">{formatBytesShort(systemMetrics.memory.free)}</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium">Cached</p>
                            <p className="text-sm text-muted-foreground">{formatBytesShort(systemMetrics.memory.cached)}</p>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>

                  {/* Top Memory Consuming Processes */}
                  <Card>
                    <CardHeader>
                      <CardTitle>Top Memory Consumers</CardTitle>
                      <CardDescription>Processes sorted by memory usage</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        {systemMetrics.processes
                          .sort((a, b) => b.memory_rss - a.memory_rss)
                          .slice(0, 10)
                          .map((process) => (
                            <div key={process.pid} className="flex items-center justify-between p-2 rounded-md bg-muted/50">
                              <div className="flex-1">
                                <div className="flex items-center space-x-2">
                                  <span className="text-sm font-medium">{process.name}</span>
                                  <Badge variant="outline" className="text-xs">PID: {process.pid}</Badge>
                                </div>
                                <p className="text-xs text-muted-foreground">
                                  CPU: {process.cpu_percent != null ? process.cpu_percent.toFixed(1) : 'N/A'}% | Threads: {process.num_threads}
                                </p>
                              </div>
                              <div className="text-right">
                                <div className="text-sm font-medium">{formatBytesShort(process.memory_rss)}</div>
                                <div className="text-xs text-muted-foreground">
                                  {systemMetrics.memory && process.memory_rss != null ? ((process.memory_rss / systemMetrics.memory.total) * 100).toFixed(1) : 'N/A'}%
                                </div>
                              </div>
                            </div>
                          ))}
                      </div>
                    </CardContent>
                  </Card>
                </div>
              ) : (
                <Card>
                  <CardContent className="text-center py-8">
                    <p className="text-sm text-muted-foreground">Memory metrics not available</p>
                  </CardContent>
                </Card>
              )}
            </TabsContent>

            <TabsContent value="storage" className="space-y-4">
              {systemMetrics && systemMetrics.disks.length > 0 ? (
                <div className="space-y-4">
                  {systemMetrics.disks.map((disk, index) => (
                    <Card key={index}>
                      <CardHeader>
                        <CardTitle>{disk.device}</CardTitle>
                        <CardDescription>
                          {disk.filesystem} mounted at {disk.mountpoint}
                        </CardDescription>
                      </CardHeader>
                      <CardContent>
                        <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mb-4">
                          <div>
                            <p className="text-sm font-medium">Total Storage</p>
                            <p className="text-sm text-muted-foreground">{formatBytesShort(disk.total)}</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium">Used Storage</p>
                            <p className="text-sm text-muted-foreground">{formatBytesShort(disk.used)}</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium">Free Storage</p>
                            <p className="text-sm text-muted-foreground">{formatBytesShort(disk.free)}</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium">Usage</p>
                            <p className="text-sm text-muted-foreground">{disk.used_percent != null ? disk.used_percent.toFixed(1) : 'N/A'}%</p>
                          </div>
                        </div>
                        <div>
                          <p className="text-sm font-medium mb-2">Storage Usage</p>
                          <Progress value={disk.used_percent || 0} className="h-4" />
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              ) : (
                <Card>
                  <CardContent className="text-center py-8">
                    <p className="text-sm text-muted-foreground">Storage metrics not available</p>
                  </CardContent>
                </Card>
              )}
            </TabsContent>
          </Tabs>
        </TabsContent>

        <TabsContent value="actions" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <Activity className="h-5 w-5" />
                  <span>System Monitoring</span>
                </CardTitle>
                <CardDescription>
                  View detailed system metrics and resource usage
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Button asChild className="w-full" disabled={node.status === 'offline'}>
                  <Link to={`/nodes/${node.agent_id}?tab=system`}>
                    <Activity className="mr-2 h-4 w-4" />
                    View System Info
                  </Link>
                </Button>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <Command className="h-5 w-5" />
                  <span>Command Execution</span>
                </CardTitle>
                <CardDescription>
                  Execute commands remotely on this node
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Button asChild className="w-full" disabled={node.status === 'offline'}>
                  <Link to={`/nodes/${node.agent_id}/command`}>
                    <Command className="mr-2 h-4 w-4" />
                    Execute Commands
                  </Link>
                </Button>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <Terminal className="h-5 w-5" />
                  <span>Terminal Access</span>
                </CardTitle>
                <CardDescription>
                  Interactive terminal session with the node
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Button asChild className="w-full" disabled={node.status === 'offline'}>
                  <Link to={`/nodes/${node.agent_id}/terminal`}>
                    <Terminal className="mr-2 h-4 w-4" />
                    Open Terminal
                  </Link>
                </Button>
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
