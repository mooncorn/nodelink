# Quick Setup Guide

This is a quick reference for setting up a Nodelink agent. For detailed documentation, see [DEPLOYMENT.md](../docs/DEPLOYMENT.md).

## 🚀 Quick Install (Recommended)

```bash
# 1. Set your configuration
export AGENT_ID="your-unique-agent-id"
export AGENT_TOKEN="your-secure-token"
export SERVER_ADDRESS="your-server.example.com:9090"

# 2. Download and run deployment script
curl -L https://github.com/mooncorn/nodelink/releases/latest/download/deploy.sh -o deploy.sh
chmod +x deploy.sh
sudo ./deploy.sh

# 3. Verify installation
sudo systemctl status nodelink-agent
sudo journalctl -u nodelink-agent -f
```

## 📋 Prerequisites

- Linux system with systemd
- Root/sudo access
- Internet connectivity
- Network access to Nodelink server

## 🔧 Manual Setup

If you prefer manual installation:

```bash
# Download agent
wget https://github.com/mooncorn/nodelink/releases/latest/download/nodelink-agent_linux_amd64.tar.gz
tar -xzf nodelink-agent_linux_amd64.tar.gz

# Install binaries
sudo cp nodelink-agent-linux-amd64 /usr/local/bin/nodelink-agent
sudo cp nodelink-updater-linux-amd64 /usr/local/bin/nodelink-updater
sudo chmod +x /usr/local/bin/nodelink-*

# Create user and directories
sudo useradd --system --no-create-home --shell /bin/false nodelink
sudo mkdir -p /etc/nodelink /var/log/nodelink /var/lib/nodelink
sudo chown nodelink:nodelink /var/log/nodelink /var/lib/nodelink

# Create configuration
sudo tee /etc/nodelink/agent.env << EOF
AGENT_ID=your-agent-id
AGENT_TOKEN=your-token
SERVER_ADDRESS=your-server:9090
AGENT_VERSION=v1.0.0
EOF
sudo chmod 600 /etc/nodelink/agent.env

# Install systemd services (copy from repository)
sudo systemctl daemon-reload
sudo systemctl enable nodelink-agent nodelink-updater
sudo systemctl start nodelink-agent nodelink-updater
```

## 🏗️ Development Build

To build from source:

```bash
# Build agent
go build -ldflags "-X main.Version=dev" -o bin/nodelink-agent ./cmd/agent

# Build updater
go build -ldflags "-X main.Version=dev" -o bin/nodelink-updater ./cmd/updater

# Run locally
./bin/nodelink-agent -agent_id=dev-agent -agent_token=dev-token -address=localhost:9090
```

## 🔍 Troubleshooting

```bash
# Check service status
sudo systemctl status nodelink-agent
sudo systemctl status nodelink-updater

# View logs
sudo journalctl -u nodelink-agent -f
sudo journalctl -u nodelink-updater -f

# Test connectivity
telnet your-server.com 9090

# Manual restart
sudo systemctl restart nodelink-agent
```

## 📖 More Information

- **Full Documentation**: [../docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md)
- **GitHub Releases**: [https://github.com/mooncorn/nodelink/releases](https://github.com/mooncorn/nodelink/releases)
- **Service Files**: `nodelink-agent.service`, `nodelink-updater.service`
