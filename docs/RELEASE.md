# Release Checklist

Use this checklist when releasing a new version of the Nodelink agent.

## Pre-Release

### Code Quality
- [ ] All tests pass (`go test ./...`)
- [ ] Code builds without errors for all platforms
- [ ] No linter warnings (`golangci-lint run`)
- [ ] Version number follows semantic versioning (e.g., v1.2.3)

### Testing
- [ ] Manual testing on target platforms
- [ ] Integration testing with server
- [ ] Updater functionality tested
- [ ] Service installation tested

### Documentation
- [ ] CHANGELOG.md updated with new features/fixes
- [ ] README.md updated if needed
- [ ] API documentation updated if applicable

## Release Process

### 1. Version Preparation
```bash
# Ensure clean working directory
git status
git stash  # if needed

# Checkout main and pull latest
git checkout main
git pull origin main

# Set version number
export VERSION="v1.2.3"
```

### 2. Build Verification
```bash
cd agent

# Test local build
go build -ldflags "-X main.Version=${VERSION}" -o bin/nodelink-agent ./cmd/agent
go build -ldflags "-X main.Version=${VERSION}" -o bin/nodelink-updater ./cmd/updater

# Verify version
./bin/nodelink-agent -version
./bin/nodelink-updater --help

# Test all platforms build
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=${VERSION}" -o /tmp/agent-amd64 ./cmd/agent
GOOS=linux GOARCH=arm64 go build -ldflags "-X main.Version=${VERSION}" -o /tmp/agent-arm64 ./cmd/agent

# Cleanup
rm -f /tmp/agent-*
```

### 3. Create and Push Tag
```bash
# Create annotated tag
git tag -a ${VERSION} -m "Release ${VERSION}

Features:
- Feature 1
- Feature 2

Bug Fixes:
- Fix 1
- Fix 2

Breaking Changes:
- None
"

# Push tag (this triggers the release workflow)
git push origin ${VERSION}
```

### 4. Monitor Release Workflow
- [ ] Go to [GitHub Actions](https://github.com/mooncorn/nodelink/actions)
- [ ] Verify "Build and Release Agent" workflow started
- [ ] Monitor build progress for both architectures
- [ ] Check for any build failures

### 5. Verify Release Artifacts
After the workflow completes:
- [ ] Go to [GitHub Releases](https://github.com/mooncorn/nodelink/releases)
- [ ] Verify new release is published
- [ ] Check all expected assets are present:
  - [ ] `nodelink-agent_linux_amd64.tar.gz`
  - [ ] `nodelink-agent_linux_arm64.tar.gz`
  - [ ] `deploy.sh`
- [ ] Download and test one archive:
  ```bash
  wget https://github.com/mooncorn/nodelink/releases/download/${VERSION}/nodelink-agent_linux_amd64.tar.gz
  tar -xzf nodelink-agent_linux_amd64.tar.gz
  ./nodelink-agent-linux-amd64 -version
  ```

## Post-Release

### 6. Update Documentation
- [ ] Update main README.md with new version references
- [ ] Update deployment documentation if needed
- [ ] Update any example configurations

### 7. Test Automatic Updates
- [ ] Set up a test agent with the previous version
- [ ] Verify the updater detects and installs the new version
- [ ] Check service restarts correctly
- [ ] Verify agent reports new version

### 8. Communication
- [ ] Announce release in team channels
- [ ] Update any deployment automation
- [ ] Notify users if breaking changes exist

### 9. Monitor Deployment
- [ ] Monitor agent logs for any issues
- [ ] Check error rates in production
- [ ] Verify automatic updates are working across fleet

## Rollback Plan

If critical issues are discovered:

### Immediate Rollback
```bash
# Create hotfix tag to previous stable version
git tag -d ${VERSION}  # Delete local tag
git push origin :refs/tags/${VERSION}  # Delete remote tag

# Create emergency release with previous version + critical fix
git checkout ${PREVIOUS_VERSION}
git checkout -b hotfix/${VERSION}-hotfix
# Apply minimal fix
git commit -m "Hotfix: Critical issue fix"
git tag -a ${VERSION}-hotfix -m "Hotfix release"
git push origin ${VERSION}-hotfix
```

### Update Communication
- [ ] Update release notes with known issues
- [ ] Mark release as pre-release if severe issues
- [ ] Communicate rollback to users
- [ ] Document issues for future prevention

## Release Automation

### GitHub Actions Workflow
The release process is automated via `.github/workflows/release-agent.yml`:

- Triggered by: Git tags matching `v*`
- Builds: Linux AMD64 and ARM64 binaries
- Creates: GitHub release with built artifacts
- Uploads: Deployment script

### Manual Trigger
If needed, you can manually trigger a release:
1. Go to GitHub Actions
2. Select "Build and Release Agent" workflow
3. Click "Run workflow"
4. Enter the tag name (e.g., v1.2.3)

## Version Strategy

### Semantic Versioning
- **Major (v2.0.0)**: Breaking changes, incompatible API changes
- **Minor (v1.1.0)**: New features, backward compatible
- **Patch (v1.0.1)**: Bug fixes, backward compatible

### Release Frequency
- **Patch releases**: As needed for critical bugs
- **Minor releases**: Monthly or when significant features are ready
- **Major releases**: Quarterly or when breaking changes are necessary

### Branch Strategy
- **main**: Always releasable, stable code
- **feature/***: New features, merged via PR
- **hotfix/***: Critical fixes, merged directly to main
- **release/***: Release preparation (if needed for complex releases)

## Troubleshooting Release Issues

### Build Failures
```bash
# Check GitHub Actions logs
# Common issues:
# - Go version mismatch
# - Dependency conflicts
# - Platform-specific build issues

# Test build locally
cd agent
go mod tidy
go mod verify
GOOS=linux GOARCH=amd64 go build ./cmd/agent
GOOS=linux GOARCH=arm64 go build ./cmd/agent
```

### Missing Artifacts
- Check workflow permissions
- Verify GITHUB_TOKEN has proper scope
- Check rate limits on GitHub API

### Failed Uploads
- Verify release was created successfully
- Check asset file paths in workflow
- Confirm no file size limits exceeded

## Emergency Procedures

### Critical Security Issue
1. **Immediate**: Remove vulnerable release
2. **Fast track**: Security patch release
3. **Communication**: Security advisory
4. **Documentation**: Update security practices

### Service Disruption
1. **Rollback**: To last known good version
2. **Hotfix**: Minimal fix for critical issue
3. **Testing**: Verify fix in staging
4. **Deploy**: Fast-track release process

---

## Checklist Template

Copy this template for each release:

```markdown
## Release ${VERSION} Checklist

### Pre-Release
- [ ] Tests pass
- [ ] Builds clean
- [ ] Documentation updated
- [ ] Version number set

### Release
- [ ] Tag created and pushed
- [ ] Workflow completed successfully
- [ ] Artifacts verified
- [ ] Release notes published

### Post-Release
- [ ] Automatic updates tested
- [ ] Documentation updated
- [ ] Team notified
- [ ] Monitoring checks passed

### Issues
- [ ] No critical issues reported
- [ ] Performance metrics normal
- [ ] Error rates acceptable

**Release Date**: $(date)
**Released By**: [Your Name]
**Rollback Plan**: [If needed]
```
