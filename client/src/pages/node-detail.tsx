import { useState, useEffect } from "react"
import { useParams, Link } from "react-router-dom"
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
  Network, 
  Terminal, 
  Command, 
  Settings,
  ArrowLeft
} from "lucide-react"
import { apiService, type Node, type SystemInfo, type SystemMetrics } from "@/lib/api"

export default function NodeDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [node, setNode] = useState<Node | null>(null)
  const [systemData, setSystemData] = useState<(SystemInfo & SystemMetrics) | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchNodeData = async () => {
      if (!id) return
      
      try {
        const [nodes, status] = await Promise.all([
          apiService.getNodes(),
          apiService.getNodeStatus(id)
        ])
        
        const currentNode = nodes.find(n => n.id === id)
        setNode(currentNode || null)
        setSystemData(status)
      } catch (error) {
        console.error('Failed to fetch node data:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchNodeData()
    // Refresh data every 10 seconds
    const interval = setInterval(fetchNodeData, 10000)
    return () => clearInterval(interval)
  }, [id])

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
            <h2 className="text-3xl font-bold tracking-tight">{node.name}</h2>
            <div className="flex items-center space-x-2 mt-2">
              <Badge variant={node.status === "online" ? "default" : "destructive"}>
                {node.status}
              </Badge>
              <span className="text-sm text-muted-foreground">{node.host}</span>
            </div>
          </div>
        </div>
        
        <div className="flex space-x-2">
          <Button variant="outline" size="sm" asChild>
            <Link to={`/nodes/${id}/system`}>
              <Activity className="mr-2 h-4 w-4" />
              System
            </Link>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link to={`/nodes/${id}/command`}>
              <Command className="mr-2 h-4 w-4" />
              Command
            </Link>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link to={`/nodes/${id}/terminal`}>
              <Terminal className="mr-2 h-4 w-4" />
              Terminal
            </Link>
          </Button>
          <Button variant="outline" size="sm">
            <Settings className="mr-2 h-4 w-4" />
            Settings
          </Button>
        </div>
      </div>

      {systemData && (
        <Tabs defaultValue="overview" className="space-y-4">
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="system">System Info</TabsTrigger>
            <TabsTrigger value="network">Network</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">CPU Usage</CardTitle>
                  <Cpu className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{systemData.cpu.usage}%</div>
                  <Progress value={systemData.cpu.usage} className="mt-2" />
                  <p className="text-xs text-muted-foreground mt-2">
                    {systemData.cpu.cores} cores @ {systemData.cpu.frequency}
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Memory Usage</CardTitle>
                  <MemoryStick className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{systemData.memory.usage}%</div>
                  <Progress value={systemData.memory.usage} className="mt-2" />
                  <p className="text-xs text-muted-foreground mt-2">
                    {systemData.memory.used}MB / {systemData.memory.total}MB
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Disk Usage</CardTitle>
                  <HardDrive className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{systemData.disk.usage}%</div>
                  <Progress value={systemData.disk.usage} className="mt-2" />
                  <p className="text-xs text-muted-foreground mt-2">
                    {systemData.disk.used}GB / {systemData.disk.total}GB
                  </p>
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          <TabsContent value="system" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>System Information</CardTitle>
                <CardDescription>Detailed system specifications</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Operating System</p>
                    <p className="text-sm">{systemData.os}</p>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Kernel</p>
                    <p className="text-sm">{systemData.kernel}</p>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Architecture</p>
                    <p className="text-sm">{systemData.arch}</p>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Hostname</p>
                    <p className="text-sm">{systemData.hostname}</p>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Uptime</p>
                    <p className="text-sm">{systemData.uptime}</p>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">CPU Model</p>
                    <p className="text-sm">{systemData.cpu.model}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="network" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Network Interfaces</CardTitle>
                <CardDescription>Network configuration and statistics</CardDescription>
              </CardHeader>
              <CardContent>
                {systemData.network.interfaces.map((iface, index) => (
                  <div key={index} className="flex items-center justify-between p-4 border rounded-lg">
                    <div className="flex items-center space-x-3">
                      <Network className="h-5 w-5 text-muted-foreground" />
                      <div>
                        <p className="font-medium">{iface.name}</p>
                        <p className="text-sm text-muted-foreground">{iface.ip}</p>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="text-sm">RX: {iface.rx}</p>
                      <p className="text-sm">TX: {iface.tx}</p>
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      )}
    </div>
  )
}
