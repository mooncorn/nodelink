#!/bin/bash

# Nodelink Agent Updater
# Automatically monitors for new releases and performs seamless updates
set -euo pipefail

# Version injected during build (only dynamic value)
VERSION="__VERSION_PLACEHOLDER__"

# Repository hardcoded (never changes for this project)
REPO_OWNER="mooncorn"
REPO_NAME="nodelink"

# Static configuration (could change on new release)
AGENT_BINARY_PATH="/usr/local/bin/nodelink-agent"
UPDATER_BINARY_PATH="/usr/local/bin/nodelink-updater.sh"
TEMP_DIR="/tmp/nodelink-updater"
LOG_FILE="/var/log/nodelink/updater.log"
CHECK_INTERVAL=180 # 3 minutes

# Service names
AGENT_SERVICE="nodelink-agent.service"
UPDATER_SERVICE="nodelink-updater.service"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging function with structured output
log_operation() {
    local level="$1"
    local message="$2"
    local timestamp=$(date +'%Y-%m-%d %H:%M:%S')
    local full_message="[$timestamp] [PID:$$] [$level] $message"
    
    # Log to file (ensure directory exists first)
    local log_dir=$(dirname "$LOG_FILE")
    if [[ ! -d "$log_dir" ]]; then
        mkdir -p "$log_dir" 2>/dev/null || true
        chown nodelink:nodelink "$log_dir" 2>/dev/null || true
    fi
    
    # Use a lock file for atomic logging to prevent message corruption
    local lock_file="${LOG_FILE}.lock"
    {
        flock -x 200
        echo "$full_message" >> "$LOG_FILE" 2>/dev/null || true
    } 200>"$lock_file"
    
    # Log to stderr with colors (never to stdout to avoid interfering with function returns)
    case "$level" in
        "INFO")
            echo -e "${GREEN}$full_message${NC}" >&2
            ;;
        "WARN")
            echo -e "${YELLOW}$full_message${NC}" >&2
            ;;
        "ERROR")
            echo -e "${RED}$full_message${NC}" >&2
            ;;
        *)
            echo "$full_message" >&2
            ;;
    esac
}

# Get the latest release version from GitHub API
get_latest_version() {
    local api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    local latest_version
    
    # Use curl to get the latest release info
    if latest_version=$(curl -s "$api_url" | grep '"tag_name":' | cut -d'"' -f4 2>/dev/null); then
        echo "$latest_version"
        return 0
    else
        return 1
    fi
}

# Compare versions (returns 0 if first version is newer, 1 if same/older)
version_is_newer() {
    local new_version="$1"
    local current_version="$2"
    
    # Remove 'v' prefix if present
    new_version=${new_version#v}
    current_version=${current_version#v}
    
    # Use sort -V for version comparison
    if [[ "$(printf '%s\n' "$new_version" "$current_version" | sort -V | head -n1)" != "$new_version" ]]; then
        return 0  # new_version is newer
    else
        return 1  # current_version is same or newer
    fi
}

# Check for updates by comparing versions
check_for_updates() {
    log_operation "INFO" "Checking for updates..."
    
    local latest_version
    if ! latest_version=$(get_latest_version); then
        log_operation "ERROR" "Failed to get latest version from GitHub API"
        return 1
    fi
    
    # Get actual agent binary version
    local agent_version="unknown"
    if [[ -f "$AGENT_BINARY_PATH" ]]; then
        agent_version=$("$AGENT_BINARY_PATH" --version 2>&1 | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1 || echo "unknown")
    fi
    
    log_operation "INFO" "Current updater version: $VERSION"
    log_operation "INFO" "Current agent version: $agent_version"
    log_operation "INFO" "Latest available version: $latest_version"
    
    # Check if either updater or agent needs updating
    local updater_needs_update=false
    local agent_needs_update=false
    
    if version_is_newer "$latest_version" "$VERSION"; then
        updater_needs_update=true
    fi
    
    if [[ "$agent_version" == "unknown" ]] || version_is_newer "$latest_version" "$agent_version"; then
        agent_needs_update=true
    fi
    
    if [[ "$updater_needs_update" == "true" ]] || [[ "$agent_needs_update" == "true" ]]; then
        log_operation "INFO" "Update available: $latest_version (updater: $updater_needs_update, agent: $agent_needs_update)"
        echo "$latest_version"
        return 0
    else
        log_operation "INFO" "Both updater and agent are up to date"
        return 1
    fi
}

# Download and validate binary
download_binary() {
    local version="$1"
    local binary_type="$2"  # "agent" or "updater"
    local output_path="$3"
    
    log_operation "INFO" "Downloading $binary_type binary version $version..."
    
    # Create temp directory
    mkdir -p "$TEMP_DIR"
    
    # Detect architecture
    local arch=$(uname -m)
    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) 
            log_operation "ERROR" "Unsupported architecture: $arch"
            return 1
            ;;
    esac
    
    local platform="linux_${arch}"
    
    if [[ "$binary_type" == "agent" ]]; then
        # Download agent binary
        local asset_name="nodelink-agent_${platform}.tar.gz"
        local download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${asset_name}"
        local archive_file="$TEMP_DIR/nodelink-agent.tar.gz"
        
        log_operation "INFO" "Downloading from: $download_url"
        
        # Download the archive
        if ! curl -L -o "$archive_file" "$download_url"; then
            log_operation "ERROR" "Failed to download agent from $download_url"
            return 1
        fi
        
        # Extract the archive
        tar -xzf "$archive_file" -C "$TEMP_DIR"
        
        # Find the agent binary
        local agent_binary=""
        for file in "$TEMP_DIR"/*; do
            if [[ -f "$file" ]]; then
                local basename=$(basename "$file")
                if [[ "$basename" == *"agent"* && "$basename" != *".tar.gz" ]]; then
                    agent_binary="$file"
                    break
                fi
            fi
        done
        
        if [[ -z "$agent_binary" ]]; then
            log_operation "ERROR" "Agent binary not found in downloaded archive"
            return 1
        fi
        
        # Move to output path
        mv "$agent_binary" "$output_path"
        
    elif [[ "$binary_type" == "updater" ]]; then
        # Download updater script
        local download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/nodelink-updater.sh"
        
        log_operation "INFO" "Downloading from: $download_url"
        
        if ! curl -L -o "$output_path" "$download_url"; then
            log_operation "ERROR" "Failed to download updater from $download_url"
            return 1
        fi
    fi
    
    # Validate downloaded file
    if [[ ! -f "$output_path" ]]; then
        log_operation "ERROR" "Downloaded $binary_type not found at $output_path"
        return 1
    fi
    
    # Set permissions
    chmod +x "$output_path"
    
    log_operation "INFO" "Successfully downloaded and validated $binary_type binary"
    return 0
}

# Update the updater itself (self-update)
update_updater() {
    local new_version="$1"
    
    log_operation "INFO" "Starting updater self-update to version $new_version..."
    
    local temp_updater="$TEMP_DIR/nodelink-updater.sh"
    
    # Download new updater script
    if ! download_binary "$new_version" "updater" "$temp_updater"; then
        log_operation "ERROR" "Failed to download new updater script"
        return 1
    fi
    
    # Atomically replace the updater script
    if ! mv "$temp_updater" "$UPDATER_BINARY_PATH"; then
        log_operation "ERROR" "Failed to replace updater script"
        return 1
    fi
    
    log_operation "INFO" "Updater script updated successfully"
    
    # Restart the updater service to pick up the new version
    log_operation "INFO" "Restarting updater service..."
    
    # Give a moment for log to be written before restart
    sleep 1
    
    systemctl restart "$UPDATER_SERVICE"
    
    # This process will be terminated by the restart, so log completion
    log_operation "INFO" "Updater service restart initiated - new version will continue the update process"
    
    return 0
}

# Update the agent binary
update_agent() {
    local new_version="$1"
    
    log_operation "INFO" "Starting agent update to version $new_version..."
    
    local temp_agent="$TEMP_DIR/nodelink-agent"
    
    # Download new agent binary
    if ! download_binary "$new_version" "agent" "$temp_agent"; then
        log_operation "ERROR" "Failed to download new agent binary"
        return 1
    fi
    
    # Stop agent service
    log_operation "INFO" "Stopping agent service..."
    if ! systemctl stop "$AGENT_SERVICE"; then
        log_operation "ERROR" "Failed to stop agent service"
        return 1
    fi
    
    # Create backup of current binary
    if [[ -f "$AGENT_BINARY_PATH" ]]; then
        cp "$AGENT_BINARY_PATH" "${AGENT_BINARY_PATH}.backup"
        log_operation "INFO" "Created backup of current agent binary"
    fi
    
    # Atomically replace agent binary
    if ! mv "$temp_agent" "$AGENT_BINARY_PATH"; then
        log_operation "ERROR" "Failed to replace agent binary"
        # Try to restore backup
        if [[ -f "${AGENT_BINARY_PATH}.backup" ]]; then
            mv "${AGENT_BINARY_PATH}.backup" "$AGENT_BINARY_PATH"
            log_operation "INFO" "Restored agent binary from backup"
        fi
        return 1
    fi
    
    # Set proper ownership and permissions
    chown root:root "$AGENT_BINARY_PATH"
    chmod +x "$AGENT_BINARY_PATH"
    
    # Start agent service
    log_operation "INFO" "Starting agent service..."
    if ! systemctl start "$AGENT_SERVICE"; then
        log_operation "ERROR" "Failed to start agent service after update"
        # Try to restore backup
        if [[ -f "${AGENT_BINARY_PATH}.backup" ]]; then
            mv "${AGENT_BINARY_PATH}.backup" "$AGENT_BINARY_PATH"
            systemctl start "$AGENT_SERVICE" || true
            log_operation "ERROR" "Restored agent binary from backup and attempted to restart service"
        fi
        return 1
    fi
    
    log_operation "INFO" "Agent binary updated successfully"
    return 0
}

# Verify the update was successful
verify_update() {
    local expected_version="$1"
    
    log_operation "INFO" "Verifying update to version $expected_version..."
    
    # Wait a moment for service to fully start
    sleep 5
    
    # Check if service is active
    if ! systemctl is-active --quiet "$AGENT_SERVICE"; then
        log_operation "ERROR" "Agent service is not active after update"
        return 1
    fi
    
    # Get the actual version from the agent binary
    local actual_version
    if [[ -f "$AGENT_BINARY_PATH" ]]; then
        actual_version=$("$AGENT_BINARY_PATH" --version 2>&1 | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1 || echo "unknown")
    else
        actual_version="unknown"
    fi
    
    if [[ "$actual_version" != "$expected_version" ]]; then
        log_operation "ERROR" "Version mismatch after update. Expected: $expected_version, Got: $actual_version"
        return 1
    fi
    
    log_operation "INFO" "Update verification successful - agent is running version $actual_version"
    return 0
}

# Cleanup temporary files
cleanup() {
    if [[ -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
    fi
    
    # Remove backup files older than 24 hours
    if [[ -f "${AGENT_BINARY_PATH}.backup" ]]; then
        if [[ $(find "${AGENT_BINARY_PATH}.backup" -mtime +1 2>/dev/null | wc -l) -gt 0 ]]; then
            rm -f "${AGENT_BINARY_PATH}.backup"
            log_operation "INFO" "Removed old backup file"
        fi
    fi
}

# Check if agent binary needs updating (independent of updater version)
agent_needs_update() {
    local target_version="$1"
    
    # Get actual agent binary version
    local agent_version="unknown"
    if [[ -f "$AGENT_BINARY_PATH" ]]; then
        agent_version=$("$AGENT_BINARY_PATH" --version 2>&1 | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1 || echo "unknown")
    fi
    
    log_operation "INFO" "Agent binary version: $agent_version"
    log_operation "INFO" "Target version: $target_version"
    
    if [[ "$agent_version" == "unknown" ]]; then
        return 0  # Update if we can't determine version
    fi
    
    if version_is_newer "$target_version" "$agent_version"; then
        return 0  # Agent needs update
    else
        return 1  # Agent is up to date
    fi
}

# Main update process
perform_update() {
    local new_version="$1"
    
    log_operation "INFO" "Starting update process to version $new_version"
    
    # First, always update the updater itself if needed
    if [[ "$VERSION" != "$new_version" ]]; then
        log_operation "INFO" "Updater needs update from $VERSION to $new_version"
        if ! update_updater "$new_version"; then
            log_operation "ERROR" "Updater self-update failed"
            return 1
        fi
        # Process will restart here, the new instance will continue
        return 0
    fi
    
    # If we reach here, updater is already up to date, check agent
    if agent_needs_update "$new_version"; then
        log_operation "INFO" "Agent binary needs update to $new_version"
        
        # Update the agent
        if ! update_agent "$new_version"; then
            log_operation "ERROR" "Agent update failed"
            return 1
        fi
        
        # Verify the update
        if ! verify_update "$new_version"; then
            log_operation "ERROR" "Update verification failed"
            return 1
        fi
        
        log_operation "INFO" "Agent update completed successfully"
    else
        log_operation "INFO" "Agent binary is already up to date"
    fi
    
    log_operation "INFO" "Update process completed successfully"
    return 0
}

# Main loop
main_loop() {
    log_operation "INFO" "Nodelink Agent Updater started (version: $VERSION)"
    log_operation "INFO" "Check interval: ${CHECK_INTERVAL} seconds"
    
    while true; do
        # Cleanup from previous runs
        cleanup
        
        # Check for updates
        local new_version
        if new_version=$(check_for_updates); then
            # New version available, perform update
            if perform_update "$new_version"; then
                log_operation "INFO" "Update completed successfully"
            else
                log_operation "ERROR" "Update failed - will retry on next check"
            fi
        fi
        
        # Wait for next check
        log_operation "INFO" "Sleeping for $CHECK_INTERVAL seconds until next check..."
        sleep "$CHECK_INTERVAL"
    done
}

# Handle script termination gracefully
trap 'log_operation "INFO" "Updater received termination signal, cleaning up..."; cleanup; exit 0' SIGTERM SIGINT

# Validate configuration
validate_configuration() {
    if [[ "$VERSION" == "__VERSION_PLACEHOLDER__" ]]; then
        log_operation "ERROR" "VERSION is not set. This script should be downloaded from a specific release."
        exit 1
    fi
    
    if [[ ! -f "$AGENT_BINARY_PATH" ]]; then
        log_operation "ERROR" "Agent binary not found at $AGENT_BINARY_PATH"
        exit 1
    fi
    
    # Check if running as root
    if [[ $EUID -ne 0 ]]; then
        log_operation "ERROR" "This script must be run as root"
        exit 1
    fi
}

# Main entry point
main() {
    validate_configuration
    main_loop
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Nodelink Agent Updater (version: $VERSION)"
        echo
        echo "Automatically monitors for new agent releases and performs seamless updates."
        echo
        echo "Configuration:"
        echo "  Repository: ${REPO_OWNER}/${REPO_NAME}"
        echo "  Agent Binary: $AGENT_BINARY_PATH"
        echo "  Check Interval: ${CHECK_INTERVAL} seconds"
        echo "  Log File: $LOG_FILE"
        echo
        echo "Usage:"
        echo "  $0                    # Start updater daemon"
        echo "  $0 --help            # Show this help"
        echo "  $0 --version         # Show version"
        echo
        echo "This script is designed to run as a systemd service."
        exit 0
        ;;
    --version|-v)
        echo "$VERSION"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac