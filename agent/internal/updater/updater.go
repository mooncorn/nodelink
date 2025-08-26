package updater

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Config holds the updater configuration
type Config struct {
	CheckInterval   time.Duration
	AgentBinaryPath string
	RepoOwner       string
	RepoName        string
	CurrentVersion  string
	GitHubToken     string
	DryRun          bool
}

// Updater manages automatic updates of the agent binary
type Updater struct {
	config Config
	client *http.Client
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
	Prerelease bool `json:"prerelease"`
	Draft      bool `json:"draft"`
}

// New creates a new updater instance
func New(config Config) *Updater {
	return &Updater{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start begins the update checking loop
func (u *Updater) Start() error {
	ticker := time.NewTicker(u.config.CheckInterval)
	defer ticker.Stop()

	// Check immediately on start
	if err := u.checkForUpdates(); err != nil {
		log.Printf("Initial update check failed: %v", err)
	}

	for range ticker.C {
		if err := u.checkForUpdates(); err != nil {
			log.Printf("Update check failed: %v", err)
		}
	}

	return nil
}

// checkForUpdates checks GitHub for new releases
func (u *Updater) checkForUpdates() error {
	log.Println("Checking for updates...")

	release, err := u.getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	// Normalize version strings for comparison (remove 'v' prefix if present)
	currentVersion := strings.TrimPrefix(u.config.CurrentVersion, "v")
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	if latestVersion == currentVersion {
		log.Printf("Already running latest version: %s", u.config.CurrentVersion)
		return nil
	}

	log.Printf("New version available: %s (current: %s)", release.TagName, u.config.CurrentVersion)

	if u.config.DryRun {
		log.Println("Dry run mode: would update but not actually updating")
		return nil
	}

	return u.performUpdate(release)
}

// getLatestRelease fetches the latest release from GitHub
func (u *Updater) getLatestRelease() (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", u.config.RepoOwner, u.config.RepoName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if u.config.GitHubToken != "" {
		req.Header.Set("Authorization", "token "+u.config.GitHubToken)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	// Skip drafts and prereleases
	if release.Draft || release.Prerelease {
		return nil, fmt.Errorf("latest release is draft or prerelease")
	}

	return &release, nil
}

// performUpdate downloads and installs the new version
func (u *Updater) performUpdate(release *GitHubRelease) error {
	log.Printf("Updating to version %s", release.TagName)

	// Find the appropriate asset for current platform
	assetName := u.getAssetName()
	var downloadURL string

	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no suitable asset found for platform %s_%s", runtime.GOOS, runtime.GOARCH)
	}

	// Download the new binary
	tempFile, err := u.downloadBinary(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	defer os.Remove(tempFile)

	// Extract if it's a tar.gz file
	binaryPath := tempFile
	if strings.HasSuffix(assetName, ".tar.gz") {
		extractedPath, err := u.extractBinary(tempFile)
		if err != nil {
			return fmt.Errorf("failed to extract binary: %w", err)
		}
		defer os.Remove(extractedPath)
		binaryPath = extractedPath
	}

	// Backup current binary
	backupPath := u.config.AgentBinaryPath + ".backup"
	if err := u.copyFile(u.config.AgentBinaryPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Replace the binary
	if err := u.copyFile(binaryPath, u.config.AgentBinaryPath); err != nil {
		// Restore backup on failure
		u.copyFile(backupPath, u.config.AgentBinaryPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Make sure the new binary is executable
	if err := os.Chmod(u.config.AgentBinaryPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	log.Printf("Successfully updated to version %s", release.TagName)

	// Restart the agent service
	if err := u.restartAgent(); err != nil {
		log.Printf("Failed to restart agent: %v", err)
		return err
	}

	// Clean up backup
	os.Remove(backupPath)

	return nil
}

// getAssetName returns the expected asset name for the current platform
func (u *Updater) getAssetName() string {
	if runtime.GOOS != "linux" {
		log.Fatalf("Unsupported operating system: %s. Only Linux is supported.", runtime.GOOS)
	}
	return fmt.Sprintf("nodelink-agent_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
}

// downloadBinary downloads a binary from the given URL
func (u *Updater) downloadBinary(url string) (string, error) {
	resp, err := u.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "nodelink-agent-*")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// Copy content to temporary file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

// extractBinary extracts the binary from a tar.gz file
func (u *Updater) extractBinary(tarGzPath string) (string, error) {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Look for the agent binary (assuming it's named "agent" or "nodelink-agent")
		if header.Typeflag == tar.TypeReg && (header.Name == "agent" || header.Name == "nodelink-agent" || strings.HasSuffix(header.Name, "/agent") || strings.HasSuffix(header.Name, "/nodelink-agent")) {
			// Create temporary file for extracted binary
			tempFile, err := os.CreateTemp("", "nodelink-agent-extracted-*")
			if err != nil {
				return "", err
			}
			defer tempFile.Close()

			// Copy binary content
			_, err = io.Copy(tempFile, tr)
			if err != nil {
				os.Remove(tempFile.Name())
				return "", err
			}

			return tempFile.Name(), nil
		}
	}

	return "", fmt.Errorf("agent binary not found in archive")
}

// copyFile copies a file from src to dst
func (u *Updater) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// restartAgent restarts the agent service using systemctl
func (u *Updater) restartAgent() error {
	log.Println("Restarting agent service...")

	if runtime.GOOS != "linux" {
		return fmt.Errorf("unsupported operating system: %s. Only Linux is supported", runtime.GOOS)
	}

	// Try to restart using systemctl
	cmd := exec.Command("systemctl", "restart", "nodelink-agent")
	if err := cmd.Run(); err != nil {
		// If systemctl fails, try to kill the current process and let systemd restart it
		log.Printf("systemctl restart failed, trying to kill current process: %v", err)

		// Send SIGTERM to current process (ourselves and the agent)
		if err := syscall.Kill(os.Getppid(), syscall.SIGTERM); err != nil {
			log.Printf("Failed to send SIGTERM to parent process: %v", err)
		}

		// Exit this updater process so systemd can restart the service
		os.Exit(0)
	}

	return nil
}
