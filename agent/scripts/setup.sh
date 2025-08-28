#!/bin/bash

# Nodelink Agent Clean Install Script
# This script performs a clean installation by removing any existing installation
# and then installing the specified version of the Nodelink Agent.
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
VERSION="__VERSION_PLACEHOLDER__"

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

# Get installed agent version
get_installed_version() {
    local version="unknown"
    
    # Try to get version from the binary
    if [[ -f "$INSTALL_DIR/nodelink-agent" ]]; then
        # Try to extract version from the binary (capture both stdout and stderr)
        version=$("$INSTALL_DIR/nodelink-agent" --version 2>&1 | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1 || echo "unknown")
    fi
    
    echo "$version"
}

# Stop and disable services
stop_service() {
    log "Stopping and disabling services..."
    
    # Stop agent service if it exists and is running
    if systemctl is-active --quiet nodelink-agent.service 2>/dev/null; then
        log "Stopping nodelink-agent service..."
        systemctl stop nodelink-agent.service
    fi
    
    # Disable agent service if it exists
    if systemctl is-enabled --quiet nodelink-agent.service 2>/dev/null; then
        log "Disabling nodelink-agent service..."
        systemctl disable nodelink-agent.service
    fi
    
    # Stop updater service if it exists and is running
    if systemctl is-active --quiet nodelink-updater.service 2>/dev/null; then
        log "Stopping nodelink-updater service..."
        systemctl stop nodelink-updater.service
    fi
    
    # Disable updater service if it exists
    if systemctl is-enabled --quiet nodelink-updater.service 2>/dev/null; then
        log "Disabling nodelink-updater service..."
        systemctl disable nodelink-updater.service
    fi
}

# Remove service files
remove_service_file() {
    log "Removing existing systemd service files..."
    
    if [[ -f "/etc/systemd/system/nodelink-agent.service" ]]; then
        log "Removing nodelink-agent service file..."
        rm -f "/etc/systemd/system/nodelink-agent.service"
    fi
    
    if [[ -f "/etc/systemd/system/nodelink-updater.service" ]]; then
        log "Removing nodelink-updater service file..."
        rm -f "/etc/systemd/system/nodelink-updater.service"
    fi
    
    systemctl daemon-reload
}

# Remove existing binaries
remove_binary() {
    log "Removing existing binaries..."
    
    if [[ -f "$INSTALL_DIR/nodelink-agent" ]]; then
        log "Removing nodelink-agent binary..."
        rm -f "$INSTALL_DIR/nodelink-agent"
        
        # Remove backup files if they exist
        if [[ -f "$INSTALL_DIR/nodelink-agent.backup" ]]; then
            rm -f "$INSTALL_DIR/nodelink-agent.backup"
        fi
    fi
    
    if [[ -f "$INSTALL_DIR/nodelink-updater.sh" ]]; then
        log "Removing nodelink-updater script..."
        rm -f "$INSTALL_DIR/nodelink-updater.sh"
    fi
    
    # Also remove old filename if it exists
    if [[ -f "$INSTALL_DIR/updater.sh" ]]; then
        log "Removing old updater script..."
        rm -f "$INSTALL_DIR/updater.sh"
    fi
}

# Clean uninstall of existing installation
clean_uninstall() {
    local installed_version
    installed_version=$(get_installed_version)
    
    if [[ "$installed_version" != "unknown" || -f "$INSTALL_DIR/nodelink-agent" || -f "$INSTALL_DIR/nodelink-updater.sh" || -f "$INSTALL_DIR/updater.sh" || -f "/etc/systemd/system/nodelink-agent.service" || -f "/etc/systemd/system/nodelink-updater.service" ]]; then
        log "Existing Nodelink Agent installation detected (version: $installed_version)"
        log "Performing clean uninstall before installing $VERSION..."
        
        stop_service
        remove_service_file
        remove_binary
        
        log "Clean uninstall completed"
    else
        log "No existing installation detected"
    fi
}

# Download and install updater script
install_updater() {
    log "Installing Nodelink Updater..."
    
    # Download the versioned updater script
    local updater_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/nodelink-updater.sh"
    local temp_updater="/tmp/nodelink-updater.sh"
    
    log "Downloading updater script from $updater_url..."
    if ! curl -L -o "$temp_updater" "$updater_url"; then
        error "Failed to download updater script from $updater_url"
    fi
    
    # Install updater script
    cp "$temp_updater" "$INSTALL_DIR/nodelink-updater.sh"
    chmod +x "$INSTALL_DIR/nodelink-updater.sh"
    chown root:root "$INSTALL_DIR/nodelink-updater.sh"
    
    # Clean up temp file
    rm -f "$temp_updater"
    
    log "Updater script installed successfully"
}

# Install systemd services
install_service() {
    log "Installing systemd services..."
    
    log "Creating agent service file"
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

    log "Creating updater service file"
    # Create updater service file
    cat > "/etc/systemd/system/nodelink-updater.service" << EOF
[Unit]
Description=Nodelink Agent Updater
Documentation=https://github.com/mooncorn/nodelink
After=network.target
Wants=network.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/nodelink-updater.sh
Restart=always
RestartSec=60
StartLimitInterval=300s
StartLimitBurst=3

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=false
ProtectHome=true
ReadWritePaths=/usr/local/bin /var/log/nodelink /tmp

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nodelink-updater

[Install]
WantedBy=multi-user.target
EOF
    
    # Reload systemd
    systemctl daemon-reload
}

# Start and enable services
start_service() {
    log "Starting and enabling services..."
    
    # Enable and start agent service
    systemctl enable nodelink-agent.service
    log "Starting nodelink-agent service..."
    systemctl start nodelink-agent.service
    
    # Enable and start updater service
    systemctl enable nodelink-updater.service
    log "Starting nodelink-updater service..."
    systemctl start nodelink-updater.service
    
    # Check status
    sleep 2
    if systemctl is-active --quiet nodelink-agent.service; then
        log "Nodelink Agent service started successfully"
    else
        warn "Nodelink Agent service failed to start. Check logs with: journalctl -u nodelink-agent.service"
    fi
    
    if systemctl is-active --quiet nodelink-updater.service; then
        log "Nodelink Updater service started successfully"
    else
        warn "Nodelink Updater service failed to start. Check logs with: journalctl -u nodelink-updater.service"
    fi
}

# Show status
show_status() {
    log "Service Status:"
    echo
    log "Agent Service:"
    systemctl status nodelink-agent.service --no-pager -l
    echo
    log "Updater Service:"
    systemctl status nodelink-updater.service --no-pager -l
}

# Main setup function
main() {
    log "Starting Nodelink Agent clean install..."
    log "This will uninstall any existing installation and install version $VERSION"
    
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
    
    # Perform clean uninstall of any existing installation
    clean_uninstall
    
    # Create user and directories
    create_user
    create_directories
    
    # Download and install
    local installed_version
    installed_version=$(download_agent "$platform" "$VERSION" "$download_url")
    
    # Install updater
    install_updater
    
    # Install and start services
    install_service
    start_service
    
    log "Clean install completed successfully!"
    log "Agent ID: $AGENT_ID"
    log "Version: $installed_version"
    echo
    log "You can check the logs with:"
    log "  journalctl -u nodelink-agent.service -f     # Agent logs"
    log "  journalctl -u nodelink-updater.service -f   # Updater logs"
    echo
    show_status
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Nodelink Agent Clean Install Script"
        echo
        echo "This script performs a clean installation by:"
        echo "  1. Detecting any existing Nodelink Agent installation"
        echo "  2. Safely removing the existing installation (service, binary)"
        echo "  3. Installing the new version from scratch"
        echo
        echo "Environment Variables (required):"
        echo "  AGENT_ID        - Unique identifier"
        echo "  AGENT_TOKEN     - Authentication token"
        echo
        echo "Usage:"
        echo "  sudo AGENT_ID=my-agent AGENT_TOKEN=secret ./setup.sh"
        echo
        echo "Note: This script will preserve user data and logs but replace"
        echo "      the agent binary and service configuration."
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac
