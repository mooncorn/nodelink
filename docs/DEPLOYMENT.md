# Nodelink Agent Deployment Guide

This guide covers everything you need to know about deploying and managing Nodelink agents, from releasing new versions to setting up agents on target machines.

## Table of Contents

1. [Release Management](#release-management)
2. [Agent Setup](#agent-setup)
3. [Configuration Management](#configuration-management)
4. [Monitoring and Maintenance](#monitoring-and-maintenance)
5. [Troubleshooting](#troubleshooting)
6. [Security Best Practices](#security-best-practices)

---

## Release Management

### How to Release a New Version

#### Prerequisites

- Push access to the GitHub repository
- Git configured with your credentials
- Local development environment set up

#### Step-by-Step Release Process

1. **Prepare the Release**
   ```bash
   # Ensure you're on the main branch and up to date
   git checkout main
   git pull origin main
   
   # Verify everything builds correctly
   cd agent
   go build -ldflags "-X main.Version=v1.2.3" -o bin/nodelink-agent ./cmd/agent
   go build -ldflags "-X main.Version=v1.2.3" -o bin/nodelink-updater ./cmd/updater
   
   # Run tests
   go test ./...
   ```

2. **Create and Push a Git Tag**
   ```bash
   # Create a new tag (use semantic versioning)
   git tag -a v1.2.3 -m "Release v1.2.3: Add new features and bug fixes"
   
   # Push the tag to trigger the release workflow
   git push origin v1.2.3
   ```

3. **Monitor the Release Process**
   - Go to GitHub Actions: `https://github.com/mooncorn/nodelink/actions`
   - Watch the "Build and Release Agent" workflow
   - Verify that all builds complete successfully
   - Check that release assets are uploaded

4. **Verify the Release**
   - Go to GitHub Releases: `https://github.com/mooncorn/nodelink/releases`
   - Confirm the new release is published with all assets:
     - `nodelink-agent_linux_amd64.tar.gz`
     - `nodelink-agent_linux_arm64.tar.gz`
     - `deploy.sh` (version-specific deployment script)
     - `uninstall.sh` (version-specific uninstall script)
   - Test download one of the assets to ensure they're accessible
   - Verify the deployment script installs the correct version

#### Manual Release (if needed)

If the automated workflow fails, you can create a release manually:

```bash
# Build for all platforms
cd agent
VERSION=v1.2.3
LDFLAGS="-ldflags \"-X main.Version=${VERSION}\""

mkdir -p dist

# Linux AMD64
GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/nodelink-agent-linux-amd64 ./cmd/agent
GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/nodelink-updater-linux-amd64 ./cmd/updater
tar -czf dist/nodelink-agent_linux_amd64.tar.gz -C dist nodelink-agent-linux-amd64 nodelink-updater-linux-amd64

# Linux ARM64
GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o dist/nodelink-agent-linux-arm64 ./cmd/agent
GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o dist/nodelink-updater-linux-arm64 ./cmd/updater
tar -czf dist/nodelink-agent_linux_arm64.tar.gz -C dist nodelink-agent-linux-arm64 nodelink-updater-linux-arm64
```

Then manually create the GitHub release and upload the files.

#### Release Notes Template

When creating releases, use this template for consistent release notes:

```markdown
## Nodelink Agent v1.2.3

### 🚀 New Features
- Feature 1 description
- Feature 2 description

### 🐛 Bug Fixes
- Fix 1 description
- Fix 2 description

### 🔧 Improvements
- Improvement 1 description
- Improvement 2 description

### 📦 Installation

Download the appropriate binary for your platform and follow the [deployment guide](https://github.com/mooncorn/nodelink/blob/main/docs/DEPLOYMENT.md).

**Quick Install:**
```bash
export AGENT_ID="your-agent-id"
export AGENT_TOKEN="your-agent-token"

# Download version-specific deployment script (installs exactly v1.2.3)
curl -L https://github.com/mooncorn/nodelink/releases/download/v1.2.3/deploy.sh -o deploy.sh
chmod +x deploy.sh
sudo ./deploy.sh
```

**Note:** This deployment script will install exactly the specified version. The agent's built-in updater will handle future upgrades automatically.

### 🔄 Automatic Updates
Existing agents with the updater service will automatically check for updates and upgrade to this version within 30 minutes.
```

---

## Agent Setup

### Prerequisites

- Linux system (Ubuntu 18.04+, CentOS 7+, or equivalent)
- Root or sudo access
- Internet connectivity for downloading releases
- Network access to your Nodelink server

### Method 1: Automatic Deployment (Recommended)

The automatic deployment script handles everything for you:

1. **Prepare Environment Variables**
   ```bash
   # Required configuration
   export AGENT_ID="production-web-01"        # Unique identifier
   export AGENT_TOKEN="your-secure-token"     # Authentication token
   export SERVER_ADDRESS="nodelink.example.com:9090"  # Server address

   # Optional configuration
   export REPO_OWNER="mooncorn"               # GitHub repository owner
   export REPO_NAME="nodelink"                # GitHub repository name
   export INSTALL_DIR="/usr/local/bin"        # Installation directory
   export CONFIG_DIR="/etc/nodelink"          # Configuration directory
   export SERVICE_USER="nodelink"             # System user for services
   export GITHUB_TOKEN="ghp_xxxxx"            # GitHub token (for private repos)
   ```

2. **Download and Run Deployment Script**
   ```bash
   # Download deployment script for a specific version (recommended)
   curl -L https://github.com/mooncorn/nodelink/releases/download/v1.2.3/deploy.sh -o deploy.sh

   # Alternative: Download from latest release (gets the newest version)
   # curl -L https://github.com/mooncorn/nodelink/releases/latest/download/deploy.sh -o deploy.sh

   # Make it executable
   chmod +x deploy.sh

   # Run the deployment
   sudo ./deploy.sh
   ```
   
   **Important:** Using a specific version ensures predictable deployments. The script will install exactly that version, and the built-in updater will handle future upgrades.

3. **Verify Installation**
   ```bash
   # Check service status
   sudo systemctl status nodelink-agent
   sudo systemctl status nodelink-updater

   # View logs
   sudo journalctl -u nodelink-agent -f
   ```

### Method 2: Manual Installation

For more control over the installation process:

1. **Download the Agent**
   ```bash
   # Determine your architecture
   ARCH=$(uname -m)
   case "$ARCH" in
       x86_64) PLATFORM="linux_amd64" ;;
       aarch64) PLATFORM="linux_arm64" ;;
       *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
   esac

   # Download the latest release
   wget https://github.com/mooncorn/nodelink/releases/latest/download/nodelink-agent_${PLATFORM}.tar.gz

   # Extract the archive
   tar -xzf nodelink-agent_${PLATFORM}.tar.gz
   ```

2. **Install Binaries**
   ```bash
   # Install to system directory
   sudo cp nodelink-agent-* /usr/local/bin/nodelink-agent
   sudo cp nodelink-updater-* /usr/local/bin/nodelink-updater
   sudo chmod +x /usr/local/bin/nodelink-agent /usr/local/bin/nodelink-updater
   ```

3. **Create System User**
   ```bash
   # Create dedicated user for the service
   sudo useradd --system --no-create-home --shell /bin/false nodelink
   ```

4. **Create Directories**
   ```bash
   # Create necessary directories
   sudo mkdir -p /etc/nodelink /var/log/nodelink /var/lib/nodelink
   sudo chown nodelink:nodelink /var/log/nodelink /var/lib/nodelink
   sudo chmod 755 /etc/nodelink /var/log/nodelink /var/lib/nodelink
   ```

5. **Create Configuration**
   ```bash
   # Create environment file
   sudo tee /etc/nodelink/agent.env << EOF
   AGENT_ID=your-agent-id
   AGENT_TOKEN=your-agent-token
   SERVER_ADDRESS=your-server:9090
   AGENT_VERSION=v1.2.3
   EOF

   sudo chmod 600 /etc/nodelink/agent.env
   sudo chown root:root /etc/nodelink/agent.env
   ```

6. **Install Systemd Services**
   ```bash
   # Download service files from the repository
   sudo curl -L https://raw.githubusercontent.com/mooncorn/nodelink/main/agent/nodelink-agent.service \
       -o /etc/systemd/system/nodelink-agent.service
   
   sudo curl -L https://raw.githubusercontent.com/mooncorn/nodelink/main/agent/nodelink-updater.service \
       -o /etc/systemd/system/nodelink-updater.service

   # Reload systemd and enable services
   sudo systemctl daemon-reload
   sudo systemctl enable nodelink-agent.service
   sudo systemctl enable nodelink-updater.service
   ```

7. **Start Services**
   ```bash
   # Start the services
   sudo systemctl start nodelink-agent.service
   sudo systemctl start nodelink-updater.service

   # Verify they're running
   sudo systemctl status nodelink-agent
   sudo systemctl status nodelink-updater
   ```

### Method 3: Docker Deployment

For containerized environments:

1. **Create Docker Compose File**
   ```yaml
   # docker-compose.yml
   version: '3.8'
   
   services:
     nodelink-agent:
       image: nodelink/agent:latest
       environment:
         - AGENT_ID=docker-agent-01
         - AGENT_TOKEN=your-secure-token
         - SERVER_ADDRESS=nodelink.example.com:9090
       restart: unless-stopped
       volumes:
         - agent-data:/var/lib/nodelink
         - agent-logs:/var/log/nodelink
       networks:
         - nodelink
   
   volumes:
     agent-data:
     agent-logs:
   
   networks:
     nodelink:
   ```

2. **Deploy**
   ```bash
   docker-compose up -d
   ```

---

## Configuration Management

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AGENT_ID` | Yes | - | Unique identifier for the agent |
| `AGENT_TOKEN` | Yes | - | Authentication token |
| `SERVER_ADDRESS` | Yes | - | Server address (host:port) |
| `AGENT_VERSION` | No | auto-detected | Current agent version |
| `GITHUB_TOKEN` | No | - | GitHub token for API requests |

### Configuration File Locations

- **Environment file**: `/etc/nodelink/agent.env`
- **Service files**: `/etc/systemd/system/nodelink-*.service`
- **Log directory**: `/var/log/nodelink/`
- **Data directory**: `/var/lib/nodelink/`
- **Binary location**: `/usr/local/bin/nodelink-*`

### Updating Configuration

1. **Modify environment file**:
   ```bash
   sudo nano /etc/nodelink/agent.env
   ```

2. **Restart services**:
   ```bash
   sudo systemctl restart nodelink-agent
   sudo systemctl restart nodelink-updater
   ```

---

## Monitoring and Maintenance

### Service Management

```bash
# Check service status
sudo systemctl status nodelink-agent
sudo systemctl status nodelink-updater

# Start/stop services
sudo systemctl start nodelink-agent
sudo systemctl stop nodelink-agent

# Restart services
sudo systemctl restart nodelink-agent
sudo systemctl restart nodelink-updater

# Enable/disable auto-start
sudo systemctl enable nodelink-agent
sudo systemctl disable nodelink-agent
```

### Log Management

```bash
# View live logs
sudo journalctl -u nodelink-agent -f
sudo journalctl -u nodelink-updater -f

# View recent logs
sudo journalctl -u nodelink-agent --since "1 hour ago"
sudo journalctl -u nodelink-updater --since "1 hour ago"

# View logs with specific priority
sudo journalctl -u nodelink-agent -p err

# Export logs
sudo journalctl -u nodelink-agent --since "2024-01-01" > agent-logs.txt
```

### Health Checks

Create a simple health check script:

```bash
#!/bin/bash
# /usr/local/bin/nodelink-health-check.sh

echo "=== Nodelink Agent Health Check ==="
echo "Date: $(date)"
echo

# Check service status
echo "Service Status:"
systemctl is-active --quiet nodelink-agent && echo "✓ Agent: Running" || echo "✗ Agent: Stopped"
systemctl is-active --quiet nodelink-updater && echo "✓ Updater: Running" || echo "✗ Updater: Stopped"
echo

# Check recent errors
echo "Recent Errors (last 1 hour):"
ERROR_COUNT=$(journalctl -u nodelink-agent -u nodelink-updater --since "1 hour ago" -p err --no-pager | wc -l)
if [ "$ERROR_COUNT" -eq 0 ]; then
    echo "✓ No errors found"
else
    echo "✗ Found $ERROR_COUNT errors"
    journalctl -u nodelink-agent -u nodelink-updater --since "1 hour ago" -p err --no-pager | tail -10
fi
echo

# Check disk space
echo "Disk Usage:"
df -h /var/log/nodelink /var/lib/nodelink | grep -v Filesystem

# Check connectivity (if server is reachable)
echo
echo "Connectivity:"
SERVER_HOST=$(grep SERVER_ADDRESS /etc/nodelink/agent.env | cut -d'=' -f2 | cut -d':' -f1)
if [ -n "$SERVER_HOST" ]; then
    if ping -c 1 "$SERVER_HOST" >/dev/null 2>&1; then
        echo "✓ Server reachable: $SERVER_HOST"
    else
        echo "✗ Server unreachable: $SERVER_HOST"
    fi
fi
```

### Automated Monitoring

Set up a cron job for regular health checks:

```bash
# Add to crontab (sudo crontab -e)
# Run health check every 30 minutes
*/30 * * * * /usr/local/bin/nodelink-health-check.sh >> /var/log/nodelink/health-check.log 2>&1
```

---

## Troubleshooting

### Common Issues and Solutions

#### Agent Won't Start

1. **Check service status**:
   ```bash
   sudo systemctl status nodelink-agent
   sudo journalctl -u nodelink-agent --no-pager
   ```

2. **Common causes**:
   - Missing or incorrect environment variables
   - Network connectivity issues
   - Binary permissions problems
   - Port conflicts

3. **Solutions**:
   ```bash
   # Check configuration
   sudo cat /etc/nodelink/agent.env
   
   # Test connectivity
   telnet your-server.com 9090
   
   # Check binary permissions
   ls -la /usr/local/bin/nodelink-*
   
   # Restart with debug output
   sudo systemctl stop nodelink-agent
   sudo -u nodelink /usr/local/bin/nodelink-agent -agent_id=test -agent_token=test -address=your-server:9090
   ```

#### Connection Issues

1. **Verify server address**:
   ```bash
   # Test DNS resolution
   nslookup your-server.com
   
   # Test port connectivity
   nc -zv your-server.com 9090
   ```

2. **Check firewall**:
   ```bash
   # Ubuntu/Debian
   sudo ufw status
   
   # CentOS/RHEL
   sudo firewall-cmd --list-all
   ```

#### Updater Not Working

1. **Check updater logs**:
   ```bash
   sudo journalctl -u nodelink-updater -f
   ```

2. **Test GitHub connectivity**:
   ```bash
   curl -I https://api.github.com/repos/mooncorn/nodelink/releases/latest
   ```

3. **Manual update**:
   ```bash
   # Stop services
   sudo systemctl stop nodelink-agent nodelink-updater
   
   # Download latest version manually
   # (follow manual installation steps)
   
   # Start services
   sudo systemctl start nodelink-agent nodelink-updater
   ```

#### Performance Issues

1. **Check resource usage**:
   ```bash
   # CPU and memory usage
   top -p $(pgrep nodelink)
   
   # Disk I/O
   iotop -p $(pgrep nodelink)
   ```

2. **Adjust service limits** (edit service files):
   ```ini
   [Service]
   LimitNOFILE=65536
   LimitNPROC=4096
   ```

---

## Security Best Practices

### Agent Security

1. **Use Strong Tokens**:
   - Generate cryptographically strong tokens (32+ characters)
   - Use different tokens for each agent
   - Rotate tokens regularly

2. **Limit Network Access**:
   ```bash
   # Allow only necessary outbound connections
   sudo ufw allow out 9090/tcp
   sudo ufw deny out 80,443/tcp
   ```

3. **File Permissions**:
   ```bash
   # Secure configuration files
   sudo chmod 600 /etc/nodelink/agent.env
   sudo chown root:root /etc/nodelink/agent.env
   
   # Secure binary files
   sudo chmod 755 /usr/local/bin/nodelink-*
   sudo chown root:root /usr/local/bin/nodelink-*
   ```

4. **Service Security**:
   - Services run as non-privileged user (`nodelink`)
   - Limited filesystem access (`ProtectSystem=strict`)
   - No new privileges (`NoNewPrivileges=true`)

### Token Management

1. **Generate Secure Tokens**:
   ```bash
   # Generate a random token
   openssl rand -hex 32
   
   # Or use UUID
   uuidgen
   ```

2. **Store Tokens Securely**:
   - Use configuration management tools (Ansible, Chef, Puppet)
   - Consider using secrets management (HashiCorp Vault, AWS Secrets Manager)
   - Never commit tokens to version control

3. **Token Rotation**:
   ```bash
   # Update token in configuration
   sudo sed -i 's/AGENT_TOKEN=.*/AGENT_TOKEN=new-token-here/' /etc/nodelink/agent.env
   
   # Restart agent
   sudo systemctl restart nodelink-agent
   ```

### Monitoring Security

1. **Log Analysis**:
   ```bash
   # Monitor for authentication failures
   sudo journalctl -u nodelink-agent | grep -i "auth\|fail\|error"
   
   # Monitor for suspicious activity
   sudo journalctl -u nodelink-agent | grep -i "refused\|denied\|invalid"
   ```

2. **Automated Alerts**:
   ```bash
   # Add to monitoring script
   if journalctl -u nodelink-agent --since "5 minutes ago" | grep -q "authentication failed"; then
       echo "ALERT: Authentication failure detected" | mail -s "Nodelink Security Alert" admin@example.com
   fi
   ```

### Network Security

1. **TLS Configuration**:
   - Ensure server uses TLS encryption
   - Validate server certificates
   - Use mutual TLS if required

2. **Firewall Rules**:
   ```bash
   # Minimal firewall rules
   sudo ufw default deny incoming
   sudo ufw default deny outgoing
   sudo ufw allow out 9090/tcp  # Nodelink server
   sudo ufw allow out 53/tcp    # DNS
   sudo ufw allow out 53/udp    # DNS
   sudo ufw allow out 443/tcp   # HTTPS (for updates)
   sudo ufw enable
   ```

---

## Appendix

### Useful Commands Reference

```bash
# Service management
sudo systemctl {start|stop|restart|status} nodelink-agent
sudo systemctl {enable|disable} nodelink-agent

# Log viewing
sudo journalctl -u nodelink-agent -f
sudo journalctl -u nodelink-agent --since "1 hour ago"
sudo journalctl -u nodelink-agent -p err

# Configuration
sudo nano /etc/nodelink/agent.env
sudo systemctl daemon-reload

# Health checks
sudo systemctl is-active nodelink-agent
sudo systemctl is-enabled nodelink-agent

# Manual operations
/usr/local/bin/nodelink-agent -version
sudo -u nodelink /usr/local/bin/nodelink-agent -help
```

### File Locations Reference

```
/usr/local/bin/nodelink-agent       # Agent binary
/usr/local/bin/nodelink-updater     # Updater binary
/etc/nodelink/agent.env             # Environment configuration
/etc/systemd/system/nodelink-*.service  # Service definitions
/var/log/nodelink/                  # Log directory
/var/lib/nodelink/                  # Data directory
```

### Support and Resources

- **GitHub Repository**: https://github.com/mooncorn/nodelink
- **Issues**: https://github.com/mooncorn/nodelink/issues
- **Releases**: https://github.com/mooncorn/nodelink/releases
- **Documentation**: https://github.com/mooncorn/nodelink/docs/
