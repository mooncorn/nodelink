import { useState, useEffect, useRef } from "react"
import { useParams, Link } from "react-router-dom"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Terminal, ArrowLeft, Wifi, WifiOff, Play, Square, AlertCircle } from "lucide-react"
import { apiService, type TerminalSession, type TerminalOutputEvent } from "@/lib/api"

interface TerminalLine {
  id: string
  content: string
  type: "input" | "output" | "error" | "system"
  timestamp: Date
  commandId?: string
}

export default function NodeTerminalPage() {
  const { nodeId } = useParams<{ nodeId: string }>()
  const [input, setInput] = useState("")
  const [lines, setLines] = useState<TerminalLine[]>([])
  const [session, setSession] = useState<TerminalSession | null>(null)
  const [connectionStatus, setConnectionStatus] = useState<'connected' | 'disconnected' | 'connecting' | 'creating'>('creating')
  const [eventSource, setEventSource] = useState<EventSource | null>(null)
  const [isExecutingCommand, setIsExecutingCommand] = useState(false)
  const scrollAreaRef = useRef<HTMLDivElement>(null)

  const addLine = (content: string, type: TerminalLine["type"], commandId?: string) => {
    const line: TerminalLine = {
      id: Date.now().toString() + Math.random(),
      content,
      type,
      timestamp: new Date(),
      commandId
    }
    setLines(prev => [...prev, line])
  }

  const scrollToBottom = () => {
    if (scrollAreaRef.current) {
      const scrollContainer = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]')
      if (scrollContainer) {
        scrollContainer.scrollTop = scrollContainer.scrollHeight
      }
    }
  }

  useEffect(() => {
    scrollToBottom()
  }, [lines])

  useEffect(() => {
    if (!nodeId) return

    const initializeTerminal = async () => {
      try {
        setConnectionStatus('creating')
        addLine("Creating terminal session...", "system")

        // Create terminal session
        const newSession = await apiService.createTerminalSession({
          agent_id: nodeId,
          shell: "bash",
          working_dir: "~"
        })

        setSession(newSession)
        addLine(`Terminal session created: ${newSession.session_id}`, "system")
        addLine(`Shell: ${newSession.shell} | Working Directory: ${newSession.working_dir}`, "system")

        // Connect to terminal stream
        const stream = apiService.connectToTerminalStream(newSession.session_id)
        if (stream) {
          setEventSource(stream)
          setConnectionStatus('connecting')

          stream.onopen = () => {
            setConnectionStatus('connected')
            addLine("Connected to terminal stream", "system")
          }

          stream.onerror = () => {
            setConnectionStatus('disconnected')
            addLine("Error: Terminal stream disconnected", "error")
          }

          stream.onmessage = (event) => {
            try {
              const data: TerminalOutputEvent = JSON.parse(event.data)
              
              if (data.output) {
                addLine(data.output, data.error ? "error" : "output", data.command_id)
              }

              if (data.is_final) {
                setIsExecutingCommand(false)
                if (data.exit_code !== undefined) {
                  addLine(`Command completed with exit code: ${data.exit_code}`, "system", data.command_id)
                }
              }
            } catch (error) {
              console.error('Error parsing terminal output:', error)
            }
          }

          // Listen for specific event types
          stream.addEventListener('terminal_output', (event) => {
            try {
              const data: TerminalOutputEvent = JSON.parse(event.data)
              if (data.output) {
                addLine(data.output, data.error ? "error" : "output", data.command_id)
              }
            } catch (error) {
              console.error('Error parsing terminal output event:', error)
            }
          })

        } else {
          setConnectionStatus('disconnected')
          addLine("Failed to connect to terminal stream", "error")
        }

      } catch (error) {
        console.error('Failed to initialize terminal:', error)
        setConnectionStatus('disconnected')
        addLine(`Failed to create terminal session: ${error instanceof Error ? error.message : 'Unknown error'}`, "error")
      }
    }

    initializeTerminal()

    return () => {
      if (eventSource) {
        eventSource.close()
      }
      if (session) {
        apiService.closeTerminalSession(session.session_id).catch(console.error)
      }
    }
  }, [nodeId])

  const executeCommand = async (command: string) => {
    if (!session || !command.trim()) return

    try {
      setIsExecutingCommand(true)
      
      // Add the input line
      addLine(`$ ${command}`, "input")

      // Execute the command
      const result = await apiService.executeTerminalCommand(session.session_id, command)
      
      // The output will come through SSE, so we don't need to handle it here
      console.log('Command execution started:', result)

    } catch (error) {
      console.error('Failed to execute command:', error)
      addLine(`Error executing command: ${error instanceof Error ? error.message : 'Unknown error'}`, "error")
      setIsExecutingCommand(false)
    }
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !isExecutingCommand && session) {
      e.preventDefault()
      executeCommand(input)
      setInput("")
    }
  }

  const handleClearTerminal = () => {
    setLines([])
    addLine("Terminal cleared", "system")
  }

  const handleReconnect = () => {
    if (session && eventSource) {
      eventSource.close()
      const stream = apiService.connectToTerminalStream(session.session_id)
      if (stream) {
        setEventSource(stream)
        setConnectionStatus('connecting')
        stream.onopen = () => setConnectionStatus('connected')
        stream.onerror = () => setConnectionStatus('disconnected')
      }
    }
  }

  const getLineColor = (type: string) => {
    switch (type) {
      case "input":
        return "text-blue-400"
      case "error":
        return "text-red-400"
      case "system":
        return "text-yellow-400"
      default:
        return "text-green-400"
    }
  }

  const getConnectionIcon = () => {
    switch (connectionStatus) {
      case 'connected':
        return <Wifi className="h-4 w-4 text-green-500" />
      case 'connecting':
      case 'creating':
        return <AlertCircle className="h-4 w-4 text-yellow-500 animate-pulse" />
      default:
        return <WifiOff className="h-4 w-4 text-red-500" />
    }
  }

  const getStatusText = () => {
    switch (connectionStatus) {
      case 'connected':
        return 'Connected'
      case 'connecting':
        return 'Connecting...'
      case 'creating':
        return 'Creating session...'
      default:
        return 'Disconnected'
    }
  }

  return (
    <div className="flex-1 space-y-4 p-4 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <div className="flex items-center space-x-4">
          <Button variant="outline" size="sm" asChild>
            <Link to={`/nodes/${nodeId}`}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <h2 className="text-3xl font-bold tracking-tight">Terminal</h2>
        </div>
        <div className="flex items-center space-x-2">
          <Badge variant="outline">Node: {nodeId}</Badge>
          {session && (
            <Badge variant="outline">Session: {session.session_id.slice(-8)}</Badge>
          )}
        </div>
      </div>

      <Card className="h-[calc(100vh-200px)]">
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Terminal className="h-5 w-5" />
              <span>Interactive Terminal</span>
            </div>
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2">
                {getConnectionIcon()}
                <span className="text-sm">{getStatusText()}</span>
              </div>
              <div className="flex items-center space-x-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleClearTerminal}
                  disabled={connectionStatus !== 'connected'}
                >
                  <Square className="h-3 w-3 mr-1" />
                  Clear
                </Button>
                {connectionStatus === 'disconnected' && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleReconnect}
                  >
                    <Play className="h-3 w-3 mr-1" />
                    Reconnect
                  </Button>
                )}
              </div>
            </div>
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col h-full">
          <ScrollArea ref={scrollAreaRef} className="flex-1 mb-4">
            <div className="bg-black text-green-400 p-4 rounded font-mono text-sm min-h-[400px]">
              {lines.map((line) => (
                <div 
                  key={line.id} 
                  className={`mb-1 ${getLineColor(line.type)}`}
                >
                  {line.content.split('\n').map((subline, index) => (
                    <div key={index}>{subline || '\u00A0'}</div>
                  ))}
                </div>
              ))}
              {connectionStatus === 'connected' && (
                <div className="flex items-center text-blue-400">
                  <span className="mr-2">$</span>
                  <Input
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    onKeyPress={handleKeyPress}
                    className="flex-1 bg-transparent border-none text-green-400 font-mono focus-visible:ring-0 focus-visible:ring-offset-0 p-0"
                    placeholder={isExecutingCommand ? "Executing command..." : "Enter command..."}
                    disabled={isExecutingCommand || connectionStatus !== 'connected'}
                    autoFocus
                  />
                  {isExecutingCommand && (
                    <span className="ml-2 text-yellow-400 animate-pulse">●</span>
                  )}
                </div>
              )}
              {connectionStatus !== 'connected' && (
                <div className="text-red-400">
                  Terminal not connected. Please check the connection.
                </div>
              )}
            </div>
          </ScrollArea>
        </CardContent>
      </Card>
    </div>
  )
}
