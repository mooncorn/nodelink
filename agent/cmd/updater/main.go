package main

import (
	"flag"
	"log"
	"time"

	"github.com/mooncorn/nodelink/agent/internal/updater"
)

func main() {
	var (
		checkInterval = flag.Duration("check-interval", 30*time.Minute, "Interval between update checks")
		agentBinary   = flag.String("agent-binary", "/usr/local/bin/nodelink-agent", "Path to the agent binary")
		repoOwner     = flag.String("repo-owner", "mooncorn", "GitHub repository owner")
		repoName      = flag.String("repo-name", "nodelink", "GitHub repository name")
		currentVer    = flag.String("current-version", "", "Current version of the agent")
		githubToken   = flag.String("github-token", "", "GitHub token for authenticated requests (optional)")
		dryRun        = flag.Bool("dry-run", false, "Only check for updates, don't download or restart")
	)
	flag.Parse()

	if *currentVer == "" {
		log.Fatal("Current version must be specified with -current-version")
	}

	config := updater.Config{
		CheckInterval:   *checkInterval,
		AgentBinaryPath: *agentBinary,
		RepoOwner:       *repoOwner,
		RepoName:        *repoName,
		CurrentVersion:  *currentVer,
		GitHubToken:     *githubToken,
		DryRun:          *dryRun,
	}

	u := updater.New(config)

	log.Printf("Starting updater for %s/%s, current version: %s", *repoOwner, *repoName, *currentVer)
	log.Printf("Checking for updates every %v", *checkInterval)

	if err := u.Start(); err != nil {
		log.Fatalf("Failed to start updater: %v", err)
	}

	// Keep the updater running
	select {}
}
