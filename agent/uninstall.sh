#!/bin/bash

# Nodelink Agent Uninstall Script
set -euo pipefail

# Configuration (should match deploy.sh defaults)
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-/etc/nodelink}"
LOG_DIR="${LOG_DIR:-/var/log/nodelink}"
DATA_DIR="${DATA_DIR:-/var/lib/nodelink}"
SERVICE_USER="${SERVICE_USER:-nodelink}"

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

# Stop and disable services
stop_services() {
    log "Stopping and disabling services..."
    
    # Stop services if they exist and are running
    if systemctl is-active --quiet nodelink-agent.service 2>/dev/null; then
        log "Stopping nodelink-agent service..."
        systemctl stop nodelink-agent.service
    fi
    
    if systemctl is-active --quiet nodelink-updater.service 2>/dev/null; then
        log "Stopping nodelink-updater service..."
        systemctl stop nodelink-updater.service
    fi
    
    # Disable services if they exist
    if systemctl is-enabled --quiet nodelink-agent.service 2>/dev/null; then
        log "Disabling nodelink-agent service..."
        systemctl disable nodelink-agent.service
    fi
    
    if systemctl is-enabled --quiet nodelink-updater.service 2>/dev/null; then
        log "Disabling nodelink-updater service..."
        systemctl disable nodelink-updater.service
    fi
}

# Remove service files
remove_service_files() {
    log "Removing systemd service files..."
    
    if [[ -f "/etc/systemd/system/nodelink-agent.service" ]]; then
        rm -f "/etc/systemd/system/nodelink-agent.service"
        log "Removed nodelink-agent.service"
    fi
    
    if [[ -f "/etc/systemd/system/nodelink-updater.service" ]]; then
        rm -f "/etc/systemd/system/nodelink-updater.service"
        log "Removed nodelink-updater.service"
    fi
    
    # Reload systemd to reflect changes
    systemctl daemon-reload
}

# Remove binaries
remove_binaries() {
    log "Removing binaries..."
    
    if [[ -f "$INSTALL_DIR/nodelink-agent" ]]; then
        rm -f "$INSTALL_DIR/nodelink-agent"
        log "Removed nodelink-agent binary"
    fi
    
    if [[ -f "$INSTALL_DIR/nodelink-updater" ]]; then
        rm -f "$INSTALL_DIR/nodelink-updater"
        log "Removed nodelink-updater binary"
    fi
    
    # Remove backup files if they exist
    if [[ -f "$INSTALL_DIR/nodelink-agent.backup" ]]; then
        rm -f "$INSTALL_DIR/nodelink-agent.backup"
        log "Removed nodelink-agent backup"
    fi
}

# Remove configuration and data directories
remove_directories() {
    log "Removing configuration and data directories..."
    
    # Remove configuration directory
    if [[ -d "$CONFIG_DIR" ]]; then
        rm -rf "$CONFIG_DIR"
        log "Removed configuration directory: $CONFIG_DIR"
    fi
    
    # Ask user about log and data directories
    read -p "Remove log directory ($LOG_DIR)? This will delete all agent logs. [y/N]: " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [[ -d "$LOG_DIR" ]]; then
            rm -rf "$LOG_DIR"
            log "Removed log directory: $LOG_DIR"
        fi
    else
        log "Keeping log directory: $LOG_DIR"
    fi
    
    read -p "Remove data directory ($DATA_DIR)? This will delete all agent data. [y/N]: " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [[ -d "$DATA_DIR" ]]; then
            rm -rf "$DATA_DIR"
            log "Removed data directory: $DATA_DIR"
        fi
    else
        log "Keeping data directory: $DATA_DIR"
    fi
}

# Remove system user
remove_user() {
    if id "$SERVICE_USER" &>/dev/null; then
        read -p "Remove system user '$SERVICE_USER'? [y/N]: " -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log "Removing system user: $SERVICE_USER"
            userdel "$SERVICE_USER" 2>/dev/null || warn "Failed to remove user $SERVICE_USER (may not exist or have dependencies)"
        else
            log "Keeping system user: $SERVICE_USER"
        fi
    else
        log "System user $SERVICE_USER does not exist"
    fi
}

# Show uninstall summary
show_summary() {
    log "Uninstall Summary:"
    echo
    log "The following components have been removed:"
    log "  ✓ Nodelink Agent and Updater services stopped and disabled"
    log "  ✓ Service files removed from /etc/systemd/system/"
    log "  ✓ Binary files removed from $INSTALL_DIR"
    log "  ✓ Configuration directory removed: $CONFIG_DIR"
    echo
    log "Remaining items (if kept):"
    if [[ -d "$LOG_DIR" ]]; then
        log "  - Log directory: $LOG_DIR"
    fi
    if [[ -d "$DATA_DIR" ]]; then
        log "  - Data directory: $DATA_DIR"
    fi
    if id "$SERVICE_USER" &>/dev/null; then
        log "  - System user: $SERVICE_USER"
    fi
    echo
}

# Main uninstall function
main() {
    log "Starting Nodelink Agent uninstall..."
    echo
    
    # Confirmation prompt
    warn "This will completely remove the Nodelink Agent from this system."
    read -p "Are you sure you want to continue? [y/N]: " -r
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log "Uninstall cancelled by user"
        exit 0
    fi
    
    echo
    check_root
    
    # Perform uninstall steps
    stop_services
    remove_service_files
    remove_binaries
    remove_directories
    remove_user
    
    show_summary
    log "Nodelink Agent uninstall completed!"
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Nodelink Agent Uninstall Script"
        echo
        echo "This script removes all components installed by the Nodelink Agent deployment script."
        echo
        echo "Components removed:"
        echo "  - Systemd services (nodelink-agent, nodelink-updater)"
        echo "  - Binary files (nodelink-agent, nodelink-updater)"
        echo "  - Configuration directory ($CONFIG_DIR)"
        echo "  - Optionally: log directory, data directory, and system user"
        echo
        echo "Environment Variables (optional):"
        echo "  INSTALL_DIR     - Installation directory (default: /usr/local/bin)"
        echo "  CONFIG_DIR      - Configuration directory (default: /etc/nodelink)"
        echo "  LOG_DIR         - Log directory (default: /var/log/nodelink)"
        echo "  DATA_DIR        - Data directory (default: /var/lib/nodelink)"
        echo "  SERVICE_USER    - System user for services (default: nodelink)"
        echo
        echo "Usage:"
        echo "  sudo ./uninstall.sh"
        exit 0
        ;;
    --force)
        # Skip confirmation prompt
        export FORCE_UNINSTALL=1
        main "$@"
        ;;
    *)
        main "$@"
        ;;
esac
