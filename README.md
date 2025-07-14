# Node Manager - Remote Server Management PoC

A simple proof-of-concept web application for managing remote servers (nodes) with secure WebSocket communication.

## Features

- **Secure Communication**: HTTPS/WSS using self-signed SSL certificates
- **Real-time Output**: Live command execution with streaming output
- **Node Management**: Dynamic node registration and discovery
- **Web Interface**: Simple HTML frontend for node selection and command execution
- **Multi-node Support**: Connect multiple nodes simultaneously

## Architecture

```
┌─────────────────┐    WSS     ┌─────────────────┐    WSS     ┌─────────────────┐
│    Frontend     │ ◄───────── │  Main Server    │ ◄───────── │   Node Agent    │
│   (Browser)     │            │   (Express +    │            │   (Node.js)     │
│                 │            │   WebSocket)    │            │                 │
└─────────────────┘            └─────────────────┘            └─────────────────┘
```

## Prerequisites

- Node.js (v14 or higher)
- npm
- `mkcert` (for SSL certificate generation)

## Setup Instructions

### 1. Install mkcert (if not already installed)

```bash
# macOS
brew install mkcert

# Linux
sudo apt install libnss3-tools
curl -JLO "https://dl.filippo.io/mkcert/latest?for=linux/amd64"
chmod +x mkcert-v*-linux-amd64
sudo mv mkcert-v*-linux-amd64 /usr/local/bin/mkcert

# Windows
choco install mkcert
```

### 2. Generate SSL Certificates

```bash
# Navigate to server/certs directory
cd server/certs

# Generate certificates for localhost
mkcert localhost

# Install the local CA (if not done before)
mkcert -install
```

This will create:
- `localhost.pem` (certificate)
- `localhost-key.pem` (private key)

### 3. Install Dependencies

```bash
# Install server dependencies
cd server
npm install

# Install node agent dependencies
cd ../node-agent
npm install
```

## Running the Application

### Option 1: Using Helper Scripts (Recommended)

```bash
# Terminal 1 - Start the server
./start-server.sh

# Terminal 2 - Start node1
./start-node.sh node1

# Terminal 3 - Start node2
./start-node.sh node2
```

### Option 2: Manual Start

#### 1. Start the Main Server

```bash
cd server
node index.js
```

The server will start on `https://localhost:8443`

#### 2. Start Node Agents

Open separate terminal windows for each node:

```bash
# Terminal 1 - Start node1
cd node-agent
node index.js node1

# Terminal 2 - Start node2
cd node-agent
node index.js node2
```

### 3. Open the Web Interface

Navigate to `https://localhost:8443` in your browser. You may need to accept the self-signed certificate warning.

## Usage

1. **Select a Node**: Choose from the dropdown list of registered nodes
2. **Enter Command**: Type a command to execute (e.g., `ls -l`, `dir`, `whoami`)
3. **Execute**: Click "Execute Command" or press Enter
4. **View Output**: Real-time command output appears in the terminal-style output area

## Example Commands

### Unix/Linux/macOS:
- `ls -l` - List files with details
- `pwd` - Show current directory
- `whoami` - Show current user
- `uname -a` - System information
- `ps aux` - Process list

### Windows:
- `dir` - List files
- `echo %cd%` - Show current directory
- `whoami` - Show current user
- `systeminfo` - System information
- `tasklist` - Process list

## Project Structure

```
nodelink/
├── server/
│   ├── index.js          # Main server (Express + WebSocket)
│   ├── package.json      # Server dependencies
│   └── certs/
│       ├── localhost.pem     # SSL certificate
│       └── localhost-key.pem # SSL private key
├── frontend/
│   ├── index.html        # Web interface
│   └── script.js         # Frontend JavaScript
├── node-agent/
│   ├── index.js          # Node agent implementation
│   └── package.json      # Agent dependencies
└── README.md
```

## Security Notes

⚠️ **This is a proof-of-concept application with simplified security:**

- Uses self-signed SSL certificates (not production-ready)
- Hardcoded node authentication tokens
- No user authentication or authorization
- Minimal input validation
- No rate limiting or DoS protection

**Do not use in production environments without proper security measures!**

## Configuration

### Node Authentication

Node tokens are currently hardcoded in `server/index.js`:

```javascript
const validNodes = {
  "node1": "token-for-node1",
  "node2": "token-for-node2"
};
```

### Adding New Nodes

1. Add the node ID and token to the `validNodes` object in `server/index.js`
2. Update the token mapping in `node-agent/index.js`
3. Restart the server

## Troubleshooting

### SSL Certificate Issues

If you encounter SSL certificate errors:

1. Ensure `mkcert` is installed and the local CA is installed:
   ```bash
   mkcert -install
   ```

2. Regenerate certificates:
   ```bash
   cd server/certs
   rm localhost.pem localhost-key.pem
   mkcert localhost
   ```

### Node Connection Issues

- Check that the server is running on port 8443
- Verify node tokens match between server and agent
- Ensure no firewall is blocking the connection

### Browser Connection Issues

- Try accessing `https://localhost:8443` directly
- Accept the self-signed certificate warning
- Clear browser cache if needed

## Extending the Application

### Adding Features

- **User Authentication**: Add login/logout functionality
- **Node Management**: Add/remove nodes dynamically
- **Command History**: Store and display previous commands
- **File Upload/Download**: Transfer files between nodes
- **Real-time Monitoring**: Display node status and metrics

### Production Considerations

- Use proper SSL certificates from a trusted CA
- Implement robust authentication and authorization
- Add input validation and sanitization
- Use environment variables for configuration
- Add logging and monitoring
- Implement proper error handling
- Add rate limiting and security headers

## License

This project is open source and available under the [MIT License](LICENSE).