import { useState, useEffect } from "react"
import { useParams, Link } from "react-router-dom"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { HardDrive, MemoryStick, Cpu, Network, ArrowLeft } from "lucide-react"
import { apiService, type SystemInfo, type SystemMetrics } from "@/lib/api"

export default function NodeSystemPage() {
  const { nodeId } = useParams()
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null)
  const [systemMetrics, setSystemMetrics] = useState<SystemMetrics | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const loadSystemData = async () => {
      if (!nodeId) return
      
      try {
        const [info, metrics] = await Promise.all([
          apiService.getSystemInfo(nodeId),
          apiService.getSystemMetrics(nodeId)
        ])
        setSystemInfo(info)
        setSystemMetrics(metrics)
      } catch (error) {
        console.error('Failed to load system data:', error)
      } finally {
        setLoading(false)
      }
    }

    loadSystemData()
  }, [nodeId])

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Link to={`/nodes/${nodeId}`}>
            <Button variant="ghost" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">System Information - {nodeId}</h1>
            <p className="text-muted-foreground">
              Loading system information...
            </p>
          </div>
        </div>
        <div className="space-y-4">
          {[...Array(6)].map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <div className="h-6 bg-muted rounded animate-pulse" />
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <div className="h-4 bg-muted rounded animate-pulse" />
                  <div className="h-4 bg-muted rounded animate-pulse w-3/4" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    )
  }

  if (!systemInfo || !systemMetrics) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Link to={`/nodes/${nodeId}`}>
            <Button variant="ghost" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">System Information - {nodeId}</h1>
            <p className="text-muted-foreground">
              Failed to load system information
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link to={`/nodes/${nodeId}`}>
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
        </Link>
        <div>
          <h1 className="text-3xl font-bold">System Information - {nodeId}</h1>
          <p className="text-muted-foreground">
            Detailed system information and resource usage
          </p>
        </div>
      </div>

      <Tabs defaultValue="overview" className="w-full">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="cpu">CPU</TabsTrigger>
          <TabsTrigger value="memory">Memory</TabsTrigger>
          <TabsTrigger value="storage">Storage</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">CPU Usage</CardTitle>
                <Cpu className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{systemMetrics.cpu.usage}%</div>
                <Progress value={systemMetrics.cpu.usage} className="mt-2" />
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Memory Usage</CardTitle>
                <MemoryStick className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{systemMetrics.memory.usage}%</div>
                <Progress value={systemMetrics.memory.usage} className="mt-2" />
                <p className="text-xs text-muted-foreground mt-1">
                  {(systemMetrics.memory.used / 1024).toFixed(1)} GB / {(systemMetrics.memory.total / 1024).toFixed(1)} GB
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Disk Usage</CardTitle>
                <HardDrive className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{systemMetrics.disk.usage}%</div>
                <Progress value={systemMetrics.disk.usage} className="mt-2" />
                <p className="text-xs text-muted-foreground mt-1">
                  {systemMetrics.disk.used} GB / {systemMetrics.disk.total} GB
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Network</CardTitle>
                <Network className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{systemMetrics.network.interfaces.length}</div>
                <p className="text-xs text-muted-foreground">
                  Active interfaces
                </p>
              </CardContent>
            </Card>
          </div>

          <div className="grid gap-6 md:grid-cols-2">
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
                    <p className="text-sm text-muted-foreground">{systemInfo.os}</p>
                  </div>
                  <div>
                    <p className="text-sm font-medium">Kernel</p>
                    <p className="text-sm text-muted-foreground">{systemInfo.kernel}</p>
                  </div>
                  <div>
                    <p className="text-sm font-medium">Architecture</p>
                    <p className="text-sm text-muted-foreground">{systemInfo.arch}</p>
                  </div>
                </div>
                <div>
                  <p className="text-sm font-medium">Uptime</p>
                  <p className="text-sm text-muted-foreground">{systemInfo.uptime}</p>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Network Interfaces</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {systemMetrics.network.interfaces.map((iface, index) => (
                    <div key={index} className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium">{iface.name}</p>
                        <p className="text-xs text-muted-foreground">{iface.ip}</p>
                      </div>
                      <div className="text-right">
                        <Badge variant="secondary">
                          ↑ {iface.tx} ↓ {iface.rx}
                        </Badge>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="cpu" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>CPU Information</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm font-medium">Model</p>
                  <p className="text-sm text-muted-foreground">{systemMetrics.cpu.model}</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Cores</p>
                  <p className="text-sm text-muted-foreground">{systemMetrics.cpu.cores}</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Frequency</p>
                  <p className="text-sm text-muted-foreground">{systemMetrics.cpu.frequency}</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Current Usage</p>
                  <p className="text-sm text-muted-foreground">{systemMetrics.cpu.usage}%</p>
                </div>
              </div>
              <div>
                <p className="text-sm font-medium mb-2">Usage Graph</p>
                <Progress value={systemMetrics.cpu.usage} className="h-4" />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="memory" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Memory Information</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm font-medium">Total Memory</p>
                  <p className="text-sm text-muted-foreground">{(systemMetrics.memory.total / 1024).toFixed(1)} GB</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Used Memory</p>
                  <p className="text-sm text-muted-foreground">{(systemMetrics.memory.used / 1024).toFixed(1)} GB</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Free Memory</p>
                  <p className="text-sm text-muted-foreground">{(systemMetrics.memory.free / 1024).toFixed(1)} GB</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Usage</p>
                  <p className="text-sm text-muted-foreground">{systemMetrics.memory.usage}%</p>
                </div>
              </div>
              <div>
                <p className="text-sm font-medium mb-2">Memory Usage</p>
                <Progress value={systemMetrics.memory.usage} className="h-4" />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="storage" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Storage Information</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm font-medium">Total Storage</p>
                  <p className="text-sm text-muted-foreground">{systemMetrics.disk.total} GB</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Used Storage</p>
                  <p className="text-sm text-muted-foreground">{systemMetrics.disk.used} GB</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Free Storage</p>
                  <p className="text-sm text-muted-foreground">{systemMetrics.disk.free} GB</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Usage</p>
                  <p className="text-sm text-muted-foreground">{systemMetrics.disk.usage}%</p>
                </div>
              </div>
              <div>
                <p className="text-sm font-medium mb-2">Storage Usage</p>
                <Progress value={systemMetrics.disk.usage} className="h-4" />
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
