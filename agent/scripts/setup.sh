#!/bin/bash

# Nodelink Agent Setup Script
set -euo pipefail

# Configuration
REPO_OWNER="${REPO_OWNER:-mooncorn}"
REPO_NAME="${REPO_NAME:-nodelink}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-/etc/nodelink}"
LOG_DIR="${LOG_DIR:-/var/log/nodelink}"
DATA_DIR="${DATA_DIR:-/var/lib/nodelink}"
SERVICE_USER="${SERVICE_USER:-nodelink}"

# Version to install (will be replaced during build)
VERSION="${VERSION:-__VERSION_PLACEHOLDER__}"

# Required environment variables for the agent
AGENT_ID="${AGENT_ID:-}"
AGENT_TOKEN="${AGENT_TOKEN:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}" >&2
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}" >&2
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" >&2
    exit 1
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root (use sudo)"
    fi
}

# Validate required environment variables
validate_config() {
    if [[ -z "$AGENT_ID" ]]; then
        error "AGENT_ID environment variable is required"
    fi

    if [[ -z "$AGENT_TOKEN" ]]; then
        error "AGENT_TOKEN environment variable is required"
    fi

    log "Configuration validated"
}

# Detect system architecture
detect_architecture() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        armv7l) arch="arm" ;;
        i386|i686) arch="386" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac

    case "$os" in
        linux) os="linux" ;;
        *) error "Unsupported operating system: $os. Only Linux is supported." ;;
    esac

    echo "${os}_${arch}"
}

# Validate that version is set
validate_version() {
    if [[ "$VERSION" == "__VERSION_PLACEHOLDER__" ]]; then
        error "VERSION is not set. This script should be downloaded from a specific release."
    fi
    
    if [[ -z "$VERSION" ]]; then
        error "VERSION is not set. This script should be downloaded from a specific release."
    fi
    
    log "Installing Nodelink Agent version: $VERSION"
}

# Download and extract the agent
download_agent() {
    local platform="$1"
    local version="$2"
    local download_url="$3"

    log "Downloading Nodelink Agent $version for $platform..."

    local temp_dir
    temp_dir=$(mktemp -d)
    local archive_file="$temp_dir/nodelink-agent.tar.gz"

    # Download the archive
    if ! curl -L -o "$archive_file" "$download_url"; then
        error "Failed to download agent from $download_url"
    fi

    # Extract the archive
    log "Extracting agent..."
    tar -xzf "$archive_file" -C "$temp_dir"

    # Find and install agent binary
    local agent_binary=""

    # Look for the agent binary in the extracted files
    for file in "$temp_dir"/*; do
        if [[ -f "$file" ]]; then
            local basename=$(basename "$file")
            if [[ "$basename" == *"agent"* ]]; then
                agent_binary="$file"
                break
            fi
        fi
    done

    if [[ -z "$agent_binary" ]]; then
        error "Agent binary not found in downloaded archive"
    fi

    # Install binary
    log "Installing binary to $INSTALL_DIR..."
    cp "$agent_binary" "$INSTALL_DIR/nodelink-agent"
    chmod +x "$INSTALL_DIR/nodelink-agent"

    # Clean up
    rm -rf "$temp_dir"

    log "Binary installed successfully"
    echo "$version"
}

# Create system user for the service
create_user() {
    if ! id "$SERVICE_USER" &>/dev/null; then
        log "Creating system user: $SERVICE_USER"
        useradd --system --no-create-home --shell /bin/false "$SERVICE_USER"
    else
        log "User $SERVICE_USER already exists"
    fi
}

# Create necessary directories
create_directories() {
    log "Creating directories..."
    
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$LOG_DIR"
    mkdir -p "$DATA_DIR"
    
    chown "$SERVICE_USER:$SERVICE_USER" "$LOG_DIR" "$DATA_DIR"
    chmod 755 "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR"
}

# Check if services are running and stop them for reinstallation
handle_existing_services() {
    log "Checking for existing services..."
    
    if systemctl is-active --quiet nodelink-agent.service 2>/dev/null; then
        log "Stopping existing nodelink-agent service for reinstallation..."
        systemctl stop nodelink-agent.service
    fi
    
    if systemctl is-enabled --quiet nodelink-agent.service 2>/dev/null; then
        log "Disabling existing nodelink-agent service..."
        systemctl disable nodelink-agent.service
    fi
}

# Install systemd service
install_service() {
    log "Installing systemd service..."
    
    log "Creating service file"
    # Create agent service file
    cat > "/etc/systemd/system/nodelink-agent.service" << EOF
[Unit]
Description=Nodelink Agent
Documentation=https://github.com/mooncorn/nodelink
After=network.target
Wants=network.target

[Service]
Type=simple
User=nodelink
Group=nodelink
Environment=AGENT_ID=${AGENT_ID}
Environment=AGENT_TOKEN=${AGENT_TOKEN}
ExecStart=/usr/local/bin/nodelink-agent
Restart=always
RestartSec=5
StartLimitInterval=60s
StartLimitBurst=3

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/nodelink /var/lib/nodelink

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nodelink-agent

[Install]
WantedBy=multi-user.target
EOF
    
    # Reload systemd
    systemctl daemon-reload
}

# Start and enable service
start_service() {
    log "Starting and enabling service..."
    
    # Enable service
    systemctl enable nodelink-agent.service
    
    # Start service
    log "Starting nodelink-agent service..."
    systemctl start nodelink-agent.service
    
    # Check status
    sleep 2
    if systemctl is-active --quiet nodelink-agent.service; then
        log "Nodelink Agent service started successfully"
    else
        warn "Nodelink Agent service failed to start. Check logs with: journalctl -u nodelink-agent.service"
    fi
}

# Show status
show_status() {
    log "Service Status:"
    systemctl status nodelink-agent.service --no-pager -l
}

# Main setup function
main() {
    log "Starting Nodelink Agent setup..."
    
    check_root
    validate_config
    validate_version
    
    local platform
    platform=$(detect_architecture)
    log "Detected platform: $platform"
    
    # Build download URL for specific version
    local asset_name="nodelink-agent_${platform}.tar.gz"
    local download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/${asset_name}"
    
    log "Download URL: $download_url"
    
    # Handle existing services
    handle_existing_services
    
    # Create user and directories
    create_user
    create_directories
    
    # Download and install
    local installed_version
    installed_version=$(download_agent "$platform" "$VERSION" "$download_url")
    
    # Install and start service
    install_service
    start_service
    
    log "Setup completed successfully!"
    log "Agent ID: $AGENT_ID"
    log "Version: $installed_version"
    echo
    log "You can check the logs with:"
    log "  journalctl -u nodelink-agent.service -f"
    echo
    show_status
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Nodelink Agent Setup Script"
        echo
        echo "Environment Variables (required):"
        echo "  AGENT_ID        - Unique identifier"
        echo "  AGENT_TOKEN     - Authentication token"
        echo
        echo "Usage:"
        echo "  sudo AGENT_ID=my-agent AGENT_TOKEN=secret ./setup.sh"
        echo
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac
