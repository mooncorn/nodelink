# NodeLink React Frontend

A modern React frontend for managing remote nodes built with Vite, TypeScript, and TailwindCSS.

## Features

- **Node Management**: View and manage remote nodes with real-time status updates
- **Task Execution**: Execute shell commands, Docker operations, and system tasks
- **Real-time Updates**: WebSocket integration for live task status and node updates
- **Responsive Design**: Clean, minimalistic UI that works on desktop and mobile
- **TypeScript**: Full type safety throughout the application

## Technology Stack

- **React 19** - UI framework
- **TypeScript** - Type safety
- **Vite** - Build tool and dev server
- **TailwindCSS** - Utility-first CSS framework
- **React Router** - Client-side routing
- **Socket.IO Client** - Real-time communication
- **Axios** - HTTP client
- **Lucide React** - Icons

## Getting Started

### Prerequisites

- Node.js 18+ and npm
- NodeLink server running on `https://localhost:8443`

### Installation

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

The application will be available at `http://localhost:3000`.

## Project Structure

```
src/
├── components/          # Reusable UI components
│   └── Navbar.tsx      # Navigation bar
├── context/            # React context providers
│   └── AppContext.tsx  # Global application state
├── pages/              # Page components
│   ├── NodesPage.tsx   # Node list page
│   └── NodeDetailPage.tsx # Individual node management
├── services/           # API and service layers
│   ├── api.ts         # REST API service
│   └── socket.ts      # WebSocket service
├── types/              # TypeScript type definitions
│   └── index.ts       # Shared types
├── App.tsx            # Main application component
└── main.tsx           # Application entry point
```

## Pages

### `/nodes`

- Displays a grid of all registered nodes
- Shows node status, system information, and capabilities
- Click on any node to navigate to its detail page

### `/nodes/:id`

- Detailed node management interface
- Execute shell commands with configurable working directory and timeout
- Manage Docker containers (run, start, stop, delete, list)
- System operations (health check, system info)
- Real-time task execution and results
- View recent task history

## Real-time Features

The application uses WebSocket connections to provide real-time updates for:

- Node connection status changes
- Task creation, updates, and completion
- Live task output streaming
- Connection status indicators

## API Integration

The frontend communicates with the NodeLink server through:

- **REST API** for CRUD operations on nodes and tasks
- **WebSocket** for real-time updates and live data

## Development

### Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint

### Configuration

The application is configured to connect to the NodeLink server at `https://localhost:8443`. Update the `API_BASE_URL` and `SOCKET_URL` constants in the service files to point to your server.

## Browser Compatibility

Modern browsers with ES2015+ support. For HTTPS with self-signed certificates, you may need to accept the certificate in your browser first.
