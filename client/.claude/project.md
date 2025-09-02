# Nodelink Client - Claude Configuration

## Project Overview
The client is a modern React/TypeScript web dashboard that provides real-time monitoring and control of the distributed task execution system.

## Key Technologies
- **Framework**: React 18 with TypeScript
- **Build Tool**: Vite for fast development and building
- **Styling**: Tailwind CSS with shadcn/ui components
- **State Management**: React hooks and context
- **HTTP Client**: Fetch API with custom hooks
- **Real-time**: Server-Sent Events for live updates

## Project Structure
```
client/
├── src/
│   ├── components/             # Reusable UI components
│   │   ├── ui/                 # shadcn/ui component library
│   │   └── app-sidebar.tsx     # Main application sidebar
│   ├── pages/                  # Page components
│   │   ├── dashboard.tsx       # Main dashboard
│   │   ├── nodes.tsx           # Agent nodes overview
│   │   ├── node-detail.tsx     # Individual agent details
│   │   ├── node-command.tsx    # Command execution interface
│   │   └── node-terminal.tsx   # Terminal interface
│   ├── hooks/                  # Custom React hooks
│   ├── lib/                    # Utility functions and API client
│   └── assets/                 # Static assets
├── public/                     # Public static files
└── config/                     # Configuration files
```

## Development Guidelines
- Use TypeScript for all new components and utilities
- Follow React best practices with hooks and functional components
- Use shadcn/ui components for consistent design
- Implement responsive design with Tailwind CSS
- Handle real-time updates with Server-Sent Events
- Use custom hooks for API calls and state management

## Key Components
- **DashboardLayout**: Main application layout with sidebar
- **Agent Cards**: Display agent status and basic information
- **Terminal Interface**: Real-time terminal session management
- **Metrics Dashboard**: Real-time system metrics visualization
- **Command Interface**: Remote command execution with results

## API Integration
```typescript
// Custom hooks for API integration
const { agents, isLoading } = useAgents();
const { executeCommand } = useCommandExecution();
const { terminalSession } = useTerminalSession();
```

## Common Tasks
- Add new dashboard widgets for monitoring
- Implement new pages for additional features
- Create reusable UI components
- Add real-time data visualization
- Implement user interaction features
- Handle error states and loading states gracefully

## Build and Deployment
- Development: `npm run dev`
- Build: `npm run build`
- Preview: `npm run preview`
- Type checking: `npm run type-check`
