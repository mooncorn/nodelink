#!/bin/bash

# Nodelink Agent Deployment Script
set -euo pipefail

# Configuration
REPO_OWNER="${REPO_OWNER:-mooncorn}"
REPO_NAME="${REPO_NAME:-nodelink}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-/etc/nodelink}"
LOG_DIR="${LOG_DIR:-/var/log/nodelink}"
DATA_DIR="${DATA_DIR:-/var/lib/nodelink}"
SERVICE_USER="${SERVICE_USER:-nodelink}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"

# Required environment variables for the agent
AGENT_ID="${AGENT_ID:-}"
AGENT_TOKEN="${AGENT_TOKEN:-}"
SERVER_ADDRESS="${SERVER_ADDRESS:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
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

    if [[ -z "$SERVER_ADDRESS" ]]; then
        error "SERVER_ADDRESS environment variable is required"
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

# Get latest release information from GitHub
get_latest_release() {
    local api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    local curl_opts=("-s")

    if [[ -n "$GITHUB_TOKEN" ]]; then
        curl_opts+=("-H" "Authorization: token $GITHUB_TOKEN")
    fi

    local response
    response=$(curl "${curl_opts[@]}" "$api_url")

    if [[ $? -ne 0 ]]; then
        error "Failed to fetch release information from GitHub"
    fi

    echo "$response"
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

    # Find and install binaries
    local agent_binary=""
    local updater_binary=""

    # Look for the binaries in the extracted files
    for file in "$temp_dir"/*; do
        if [[ -f "$file" && -x "$file" ]]; then
            local basename=$(basename "$file")
            if [[ "$basename" == *"agent"* && "$basename" != *"updater"* ]]; then
                agent_binary="$file"
            elif [[ "$basename" == *"updater"* ]]; then
                updater_binary="$file"
            fi
        fi
    done

    if [[ -z "$agent_binary" ]]; then
        error "Agent binary not found in downloaded archive"
    fi

    if [[ -z "$updater_binary" ]]; then
        error "Updater binary not found in downloaded archive"
    fi

    # Install binaries
    log "Installing binaries to $INSTALL_DIR..."
    cp "$agent_binary" "$INSTALL_DIR/nodelink-agent"
    cp "$updater_binary" "$INSTALL_DIR/nodelink-updater"
    chmod +x "$INSTALL_DIR/nodelink-agent"
    chmod +x "$INSTALL_DIR/nodelink-updater"

    # Clean up
    rm -rf "$temp_dir"

    log "Binaries installed successfully"
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

# Create environment configuration
create_config() {
    local version="$1"
    
    log "Creating configuration file..."
    
    cat > "$CONFIG_DIR/agent.env" << EOF
# Nodelink Agent Configuration
AGENT_ID=$AGENT_ID
AGENT_TOKEN=$AGENT_TOKEN
SERVER_ADDRESS=$SERVER_ADDRESS
AGENT_VERSION=$version
EOF

    chmod 600 "$CONFIG_DIR/agent.env"
    chown root:root "$CONFIG_DIR/agent.env"
}

# Install systemd services
install_services() {
    log "Installing systemd services..."
    
    # Check if we're in a directory that has the service files
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local agent_service_file=""
    local updater_service_file=""
    
    # Look for service files in the same directory as the script
    if [[ -f "$script_dir/nodelink-agent.service" ]]; then
        agent_service_file="$script_dir/nodelink-agent.service"
    elif [[ -f "./nodelink-agent.service" ]]; then
        agent_service_file="./nodelink-agent.service"
    fi
    
    if [[ -f "$script_dir/nodelink-updater.service" ]]; then
        updater_service_file="$script_dir/nodelink-updater.service"
    elif [[ -f "./nodelink-updater.service" ]]; then
        updater_service_file="./nodelink-updater.service"
    fi
    
    # Use existing service files if available, otherwise create them
    if [[ -n "$agent_service_file" && -n "$updater_service_file" ]]; then
        log "Using existing service files"
        cp "$agent_service_file" "/etc/systemd/system/"
        cp "$updater_service_file" "/etc/systemd/system/"
    else
        log "Creating service files"
        # Create agent service file
        cat > "/etc/systemd/system/nodelink-agent.service" << 'EOF'
[Unit]
Description=Nodelink Agent
Documentation=https://github.com/mooncorn/nodelink
After=network.target
Wants=network.target

[Service]
Type=simple
User=nodelink
Group=nodelink
ExecStart=/usr/local/bin/nodelink-agent -agent_id=${AGENT_ID} -agent_token=${AGENT_TOKEN} -address=${SERVER_ADDRESS}
Restart=always
RestartSec=5
StartLimitInterval=60s
StartLimitBurst=3

# Environment file
EnvironmentFile=-/etc/nodelink/agent.env

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

        # Create updater service file
        cat > "/etc/systemd/system/nodelink-updater.service" << 'EOF'
[Unit]
Description=Nodelink Agent Updater
Documentation=https://github.com/mooncorn/nodelink
After=network.target
Wants=network.target

[Service]
Type=simple
User=nodelink
Group=nodelink
ExecStart=/usr/local/bin/nodelink-updater -current-version=${AGENT_VERSION} -check-interval=30m -agent-binary=/usr/local/bin/nodelink-agent -repo-owner=mooncorn -repo-name=nodelink
Restart=always
RestartSec=30
StartLimitInterval=300s
StartLimitBurst=5

# Environment file
EnvironmentFile=-/etc/nodelink/agent.env

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/usr/local/bin /var/log/nodelink /var/lib/nodelink

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nodelink-updater

[Install]
WantedBy=multi-user.target
EOF
    fi
    
    # Reload systemd
    systemctl daemon-reload
}

# Start and enable services
start_services() {
    log "Starting and enabling services..."
    
    # Enable services
    systemctl enable nodelink-agent.service
    systemctl enable nodelink-updater.service
    
    # Start services
    systemctl start nodelink-agent.service
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
    systemctl status nodelink-agent.service --no-pager -l
    echo
    systemctl status nodelink-updater.service --no-pager -l
}

# Main deployment function
main() {
    log "Starting Nodelink Agent deployment..."
    
    check_root
    validate_config
    
    local platform
    platform=$(detect_architecture)
    log "Detected platform: $platform"
    
    # Get latest release
    log "Fetching latest release information..."
    local release_info
    release_info=$(get_latest_release)
    
    local version
    version=$(echo "$release_info" | grep '"tag_name"' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')
    
    if [[ -z "$version" ]]; then
        error "Failed to extract version from release information"
    fi
    
    log "Latest version: $version"
    
    # Find download URL for our platform
    local asset_name="nodelink-agent_${platform}.tar.gz"
    local download_url
    download_url=$(echo "$release_info" | grep "browser_download_url.*$asset_name" | sed -E 's/.*"browser_download_url": "([^"]+)".*/\1/')
    
    if [[ -z "$download_url" ]]; then
        error "No download URL found for platform $platform (looking for $asset_name)"
    fi
    
    log "Download URL: $download_url"
    
    # Create user and directories
    create_user
    create_directories
    
    # Download and install
    local installed_version
    installed_version=$(download_agent "$platform" "$version" "$download_url")
    
    # Create configuration
    create_config "$installed_version"
    
    # Install and start services
    install_services
    start_services
    
    log "Deployment completed successfully!"
    log "Agent ID: $AGENT_ID"
    log "Server Address: $SERVER_ADDRESS"
    log "Version: $installed_version"
    echo
    log "You can check the logs with:"
    log "  journalctl -u nodelink-agent.service -f"
    log "  journalctl -u nodelink-updater.service -f"
    echo
    show_status
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Nodelink Agent Deployment Script"
        echo
        echo "Environment Variables (required):"
        echo "  AGENT_ID        - Unique identifier for this agent"
        echo "  AGENT_TOKEN     - Authentication token for the agent"
        echo "  SERVER_ADDRESS  - Address of the Nodelink server (host:port)"
        echo
        echo "Environment Variables (optional):"
        echo "  REPO_OWNER      - GitHub repository owner (default: mooncorn)"
        echo "  REPO_NAME       - GitHub repository name (default: nodelink)"
        echo "  INSTALL_DIR     - Installation directory (default: /usr/local/bin)"
        echo "  CONFIG_DIR      - Configuration directory (default: /etc/nodelink)"
        echo "  SERVICE_USER    - System user for services (default: nodelink)"
        echo "  GITHUB_TOKEN    - GitHub token for API requests (optional)"
        echo
        echo "Usage:"
        echo "  sudo AGENT_ID=my-agent AGENT_TOKEN=secret SERVER_ADDRESS=server:9090 ./deploy.sh"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac
