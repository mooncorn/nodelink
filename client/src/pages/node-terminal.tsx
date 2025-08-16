import { useState, useEffect, useRef } from "react"
import { useParams, Link } from "react-router-dom"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Terminal, ArrowLeft } from "lucide-react"

interface TerminalLine {
  id: string
  content: string
  type: "input" | "output" | "error"
  timestamp: Date
}

export default function NodeTerminalPage() {
  const { id } = useParams<{ id: string }>()
  const [input, setInput] = useState("")
  const [lines, setLines] = useState<TerminalLine[]>([
    {
      id: "1",
      content: "Ubuntu 22.04.3 LTS",
      type: "output",
      timestamp: new Date()
    },
    {
      id: "2", 
      content: "Welcome to the NodeLink terminal for node " + id,
      type: "output",
      timestamp: new Date()
    },
    {
      id: "3",
      content: "Type 'help' for available commands",
      type: "output", 
      timestamp: new Date()
    }
  ])
  const [currentPath] = useState("~/")
  const scrollAreaRef = useRef<HTMLDivElement>(null)

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

  const executeCommand = (command: string) => {
    // Add the input line
    const inputLine: TerminalLine = {
      id: Date.now().toString(),
      content: `ubuntu@${id}:${currentPath}$ ${command}`,
      type: "input",
      timestamp: new Date()
    }
    
    setLines(prev => [...prev, inputLine])

    // Simulate command processing
    setTimeout(() => {
      let output = ""
      let type: "output" | "error" = "output"

      switch (command.toLowerCase().trim()) {
        case "help":
          output = `Available commands:
  help     - Show this help message
  ls       - List directory contents  
  pwd      - Print working directory
  whoami   - Show current user
  date     - Show current date and time
  clear    - Clear the terminal
  uname -a - Show system information`
          break
        case "ls":
          output = `Documents  Downloads  Pictures  Videos
Desktop    Music     Public    Templates`
          break
        case "pwd":
          output = `/home/ubuntu${currentPath.slice(1)}`
          break
        case "whoami":
          output = "ubuntu"
          break
        case "date":
          output = new Date().toString()
          break
        case "clear":
          setLines([])
          return
        case "uname -a":
          output = "Linux prod-server-01 5.15.0-78-generic #85-Ubuntu SMP Fri Jul 7 15:25:09 UTC 2023 x86_64 x86_64 x86_64 GNU/Linux"
          break
        default:
          if (command.trim()) {
            output = `Command '${command}' not found. Type 'help' for available commands.`
            type = "error"
          }
      }

      if (output) {
        const outputLine: TerminalLine = {
          id: (Date.now() + 1).toString(),
          content: output,
          type: type,
          timestamp: new Date()
        }
        setLines(prev => [...prev, outputLine])
      }
    }, 100 + Math.random() * 500)
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault()
      executeCommand(input)
      setInput("")
    }
  }

  const getLineColor = (type: string) => {
    switch (type) {
      case "input":
        return "text-blue-400"
      case "error":
        return "text-red-400"
      default:
        return "text-green-400"
    }
  }

  return (
    <div className="flex-1 space-y-4 p-4 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <div className="flex items-center space-x-4">
          <Button variant="outline" size="sm" asChild>
            <Link to={`/nodes/${id}`}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <h2 className="text-3xl font-bold tracking-tight">Terminal</h2>
        </div>
        <Badge variant="outline">Node: {id}</Badge>
      </div>

      <Card className="h-[calc(100vh-200px)]">
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Terminal className="h-5 w-5" />
            <span>Interactive Terminal</span>
            <Badge variant="secondary" className="ml-auto">
              Connected
            </Badge>
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
                    <div key={index}>{subline}</div>
                  ))}
                </div>
              ))}
              <div className="flex items-center text-blue-400">
                <span className="mr-2">ubuntu@{id}:{currentPath}$</span>
                <Input
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyPress={handleKeyPress}
                  className="flex-1 bg-transparent border-none text-green-400 font-mono focus-visible:ring-0 focus-visible:ring-offset-0 p-0"
                  placeholder="Enter command..."
                  autoFocus
                />
              </div>
            </div>
          </ScrollArea>
        </CardContent>
      </Card>
    </div>
  )
}
