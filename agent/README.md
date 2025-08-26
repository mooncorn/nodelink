# Nodelink Agent

The Nodelink Agent connects to the Nodelink server and executes commands, manages terminal sessions, and reports system metrics.

## Features

- **Automatic Updates**: Built-in updater that checks GitHub releases and updates the agent automatically
- **Linux Support**: Designed specifically for Linux systems
- **Systemd Integration**: Native systemd service management
- **Secure Authentication**: Token-based authentication with the server
- **Real-time Communication**: gRPC-based bidirectional streaming
- **Resource Monitoring**: System metrics collection and reporting

## Quick Start

### Prerequisites

- A running Nodelink server
- Agent credentials (ID and token)
- Linux system with systemd

### Automatic Deployment

The easiest way to deploy the agent is using the deployment script:

```bash
# Set your configuration
export AGENT_ID="your-unique-agent-id"
export AGENT_TOKEN="your-secret-token"
export SERVER_ADDRESS="your-server.example.com:9090"

# Download and run the deployment script
curl -L https://github.com/mooncorn/nodelink/releases/latest/download/deploy.sh -o deploy.sh
chmod +x deploy.sh
sudo ./deploy.sh
```

For detailed setup instructions, see [SETUP.md](SETUP.md).

For complete deployment documentation, see [../docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md).

This script will:
1. Download the latest agent release for Linux
2. Create a system user (`nodelink`)
3. Install the binaries to `/usr/local/bin/`
4. Set up systemd services for both agent and updater
5. Configure and start the services

### Manual Installation

1. **Download the Agent**
   ```bash
   # For Linux x64
   wget https://github.com/mooncorn/nodelink/releases/latest/download/nodelink-agent_linux_amd64.tar.gz
   tar -xzf nodelink-agent_linux_amd64.tar.gz
   ```

2. **Install Binaries**
   ```bash
   sudo cp nodelink-agent-linux-amd64 /usr/local/bin/nodelink-agent
   sudo cp nodelink-updater-linux-amd64 /usr/local/bin/nodelink-updater
   sudo chmod +x /usr/local/bin/nodelink-agent /usr/local/bin/nodelink-updater
   ```

3. **Create Configuration**
   ```bash
   sudo mkdir -p /etc/nodelink
   sudo tee /etc/nodelink/agent.env << EOF
   AGENT_ID=your-agent-id
   AGENT_TOKEN=your-agent-token
   SERVER_ADDRESS=your-server:9090
   AGENT_VERSION=v1.0.0
   EOF
   sudo chmod 600 /etc/nodelink/agent.env
   ```

4. **Set up Systemd Services** (see service files in the repository)

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `AGENT_ID` | Unique identifier for this agent | Required |
| `AGENT_TOKEN` | Authentication token | Required |
| `SERVER_ADDRESS` | Server address (host:port) | Required |
| `AGENT_VERSION` | Current agent version (for updater) | Required |

### Command Line Options

**Agent (`nodelink-agent`)**:
- `-agent_id`: Agent ID (overrides env var)
- `-agent_token`: Agent token (overrides env var)
- `-address`: Server address (overrides env var)
- `-version`: Print version and exit

**Updater (`nodelink-updater`)**:
- `-check-interval`: Update check interval (default: 30m)
- `-agent-binary`: Path to agent binary (default: /usr/local/bin/nodelink-agent)
- `-repo-owner`: GitHub repo owner (default: mooncorn)
- `-repo-name`: GitHub repo name (default: nodelink)
- `-current-version`: Current version (required)
- `-github-token`: GitHub token for API requests (optional)
- `-dry-run`: Only check for updates, don't apply them

## Service Management

### Systemd Commands

```bash
# Check service status
sudo systemctl status nodelink-agent
sudo systemctl status nodelink-updater

# View logs
sudo journalctl -u nodelink-agent -f
sudo journalctl -u nodelink-updater -f

# Restart services
sudo systemctl restart nodelink-agent
sudo systemctl restart nodelink-updater

# Stop services
sudo systemctl stop nodelink-agent
sudo systemctl stop nodelink-updater

# Disable services
sudo systemctl disable nodelink-agent
sudo systemctl disable nodelink-updater
```

## Automatic Updates

The updater service automatically:
1. Checks GitHub releases every 30 minutes
2. Downloads new versions when available
3. Backs up the current binary
4. Replaces the binary with the new version
5. Restarts the agent service

### Updater Security

- Runs as a non-privileged user
- Limited file system access
- Only downloads from official GitHub releases
- Verifies downloads before installation

## Building from Source

### Prerequisites

- Go 1.23 or later

### Build Commands

```bash
# Build agent
go build -ldflags "-X main.Version=v1.0.0" -o bin/nodelink-agent ./cmd/agent

# Build updater  
go build -ldflags "-X main.Version=v1.0.0" -o bin/nodelink-updater ./cmd/updater

# Build for Linux ARM64
GOOS=linux GOARCH=arm64 go build -ldflags "-X main.Version=v1.0.0" -o nodelink-agent-linux-arm64 ./cmd/agent

# Install locally (Linux)
sudo cp bin/nodelink-agent /usr/local/bin/
sudo cp bin/nodelink-updater /usr/local/bin/
sudo chmod +x /usr/local/bin/nodelink-agent /usr/local/bin/nodelink-updater

# Run tests
go test ./...
```

### Build Targets

The build process supports Linux platforms:
- Linux (amd64, arm64)

## Development

### Project Structure

```
agent/
├── cmd/
│   ├── agent/          # Agent main binary
│   └── updater/        # Updater main binary
├── internal/
│   └── updater/        # Updater implementation
├── pkg/                # Agent packages
│   ├── grpc/          # gRPC client
│   ├── command/       # Command execution
│   ├── terminal/      # Terminal management
│   └── metrics/       # Metrics collection
└── deploy.sh          # Deployment script
```

### Adding Features

1. Implement the feature in the appropriate package
2. Update the protocol buffers if needed
3. Test the changes
4. Update documentation

## Troubleshooting

### Common Issues

**Agent won't connect to server**:
- Check network connectivity
- Verify server address and port
- Check authentication credentials
- Review server logs

**Updates not working**:
- Check GitHub API rate limits
- Verify internet connectivity
- Check updater logs: `journalctl -u nodelink-updater -f`
- Ensure proper file permissions

**Service won't start**:
- Check systemd logs: `journalctl -u nodelink-agent -f`
- Verify binary permissions
- Check configuration file syntax
- Ensure all required environment variables are set

### Logs

All services log to systemd journal. Use these commands to view logs:

```bash
# Agent logs
sudo journalctl -u nodelink-agent -f --since "1 hour ago"

# Updater logs
sudo journalctl -u nodelink-updater -f --since "1 hour ago"

# Both services
sudo journalctl -u nodelink-agent -u nodelink-updater -f
```

### Debug Mode

To run the agent in debug mode:

```bash
# Stop the service
sudo systemctl stop nodelink-agent

# Run manually with debug output
sudo -u nodelink /usr/local/bin/nodelink-agent -agent_id=$AGENT_ID -agent_token=$AGENT_TOKEN -address=$SERVER_ADDRESS
```

## Security Considerations

- Store credentials securely
- Use strong, unique agent tokens
- Regularly rotate authentication tokens
- Monitor agent activity and logs
- Keep the agent updated
- Run services as non-privileged users
- Use firewall rules to restrict network access

## Support

For issues and questions:
- **Quick Setup**: [SETUP.md](SETUP.md)
- **Complete Documentation**: [../docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md)
- **Release Process**: [../docs/RELEASE.md](../docs/RELEASE.md)
- **GitHub Issues**: https://github.com/mooncorn/nodelink/issues
- **GitHub Releases**: https://github.com/mooncorn/nodelink/releases
