#!/bin/bash

# Nodelink Agent Uninstall Script
set -euo pipefail

# Configuration (should match deploy.sh defaults)
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-/etc/nodelink}"
LOG_DIR="${LOG_DIR:-/var/log/nodelink}"
DATA_DIR="${DATA_DIR:-/var/lib/nodelink}"
SERVICE_USER="${SERVICE_USER:-nodelink}"

# Script version info (will be replaced during build)
SCRIPT_VERSION="${SCRIPT_VERSION:-__VERSION_PLACEHOLDER__}"

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

# Show version information
show_version_info() {
    local installed_version
    installed_version=$(get_installed_version)
    
    # Check if script version was properly set during build
    if [[ "$SCRIPT_VERSION" == "__VERSION_PLACEHOLDER__" ]]; then
        warn "Script version not set. This script should be downloaded from a specific release."
        SCRIPT_VERSION="unknown"
    fi
    
    log "Uninstall Script Version: $SCRIPT_VERSION"
    log "Installed Agent Version: $installed_version"
    
    if [[ "$installed_version" != "unknown" && "$SCRIPT_VERSION" != "unknown" && "$installed_version" != "$SCRIPT_VERSION" ]]; then
        warn "Version mismatch detected between installed agent ($installed_version) and uninstall script ($SCRIPT_VERSION)"
        warn "This may indicate you're using a different version of the uninstall script"
    fi
}

# Stop and disable service
stop_service() {
    log "Stopping and disabling service..."
    
    # Stop service if it exists and is running
    if systemctl is-active --quiet nodelink-agent.service 2>/dev/null; then
        log "Stopping nodelink-agent service..."
        systemctl stop nodelink-agent.service
    fi
    
    # Disable service if it exists
    if systemctl is-enabled --quiet nodelink-agent.service 2>/dev/null; then
        log "Disabling nodelink-agent service..."
        systemctl disable nodelink-agent.service
    fi
}

# Remove service file
remove_service_file() {
    log "Removing systemd service file..."
    
    if [[ -f "/etc/systemd/system/nodelink-agent.service" ]]; then
        rm -f "/etc/systemd/system/nodelink-agent.service"
        log "Removed nodelink-agent.service"
    fi
    
    # Reload systemd to reflect changes
    systemctl daemon-reload
}

# Remove binary
remove_binary() {
    log "Removing binary..."
    
    if [[ -f "$INSTALL_DIR/nodelink-agent" ]]; then
        rm -f "$INSTALL_DIR/nodelink-agent"
        log "Removed nodelink-agent binary"
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
    if [[ "${FORCE_UNINSTALL:-}" == "1" ]]; then
        # Force mode: remove directories without asking
        if [[ -d "$LOG_DIR" ]]; then
            rm -rf "$LOG_DIR"
            log "Removed log directory: $LOG_DIR"
        fi
        if [[ -d "$DATA_DIR" ]]; then
            rm -rf "$DATA_DIR"
            log "Removed data directory: $DATA_DIR"
        fi
    else
        # Interactive mode: ask user
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
    fi
}

# Remove system user
remove_user() {
    if id "$SERVICE_USER" &>/dev/null; then
        if [[ "${FORCE_UNINSTALL:-}" == "1" ]]; then
            # Force mode: remove user without asking
            log "Removing system user: $SERVICE_USER"
            userdel "$SERVICE_USER" 2>/dev/null || warn "Failed to remove user $SERVICE_USER (may not exist or have dependencies)"
        else
            # Interactive mode: ask user
            read -p "Remove system user '$SERVICE_USER'? [y/N]: " -r
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                log "Removing system user: $SERVICE_USER"
                userdel "$SERVICE_USER" 2>/dev/null || warn "Failed to remove user $SERVICE_USER (may not exist or have dependencies)"
            else
                log "Keeping system user: $SERVICE_USER"
            fi
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
    log "  ✓ Nodelink Agent service stopped and disabled"
    log "  ✓ Service file removed from /etc/systemd/system/"
    log "  ✓ Binary file removed from $INSTALL_DIR"
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
    
    # Show version information
    show_version_info
    echo
    
    # Confirmation prompt
    if [[ "${FORCE_UNINSTALL:-}" != "1" ]]; then
        warn "This will completely remove the Nodelink Agent from this system."
        read -p "Are you sure you want to continue? [y/N]: " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log "Uninstall cancelled by user"
            exit 0
        fi
    else
        log "Force uninstall mode: skipping confirmation prompts"
    fi
    
    echo
    check_root
    
    # Perform uninstall steps
    stop_service
    remove_service_file
    remove_binary
    remove_directories
    remove_user
    
    show_summary
    log "Nodelink Agent uninstall completed!"
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Nodelink Agent Uninstall Script (version: $SCRIPT_VERSION)"
        echo
        echo "This script removes all components installed by the Nodelink Agent deployment script."
        echo
        echo "Components removed:"
        echo "  - Systemd service (nodelink-agent)"
        echo "  - Binary file (nodelink-agent)"
        echo "  - Configuration directory ($CONFIG_DIR)"
        echo "  - Optionally: log directory, data directory, and system user"
        echo
        echo "Usage:"
        echo "  sudo ./uninstall.sh                    # Interactive mode (default)"
        echo "  sudo ./uninstall.sh --force            # Force mode - skip all prompts"
        echo
        echo "Force mode behavior:"
        echo "  - Skips confirmation prompts"
        echo "  - Automatically removes log and data directories"
        echo "  - Automatically removes system user"
        echo
        echo "Note: For best compatibility, use the uninstall script from the same"
        echo "      release as your installed agent version."
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
