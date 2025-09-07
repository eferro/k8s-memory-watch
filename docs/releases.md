# Releases and Versioning Guide

This document explains the complete release and versioning strategy for k8s-memory-watch.

## 📦 Release Strategy Overview

The project uses **Semantic Versioning (SemVer)** with automated releases through GitHub Actions:

- **Major version** (`x.0.0`): Breaking changes or major new features
- **Minor version** (`x.y.0`): New features, backwards compatible
- **Patch version** (`x.y.z`): Bug fixes, backwards compatible

## 🚀 Automated Release Process

### Conventional Commits

We use conventional commit messages to automatically determine version bumps:

```bash
# Patch version (x.y.Z) - Bug fixes
fix: resolve memory leak in monitor loop
fix(api): handle nil pointer in response
chore: update dependencies

# Minor version (x.Y.0) - New features
feat: add CSV output format
feat(cli): add --watch-interval flag

# Major version (X.0.0) - Breaking changes
feat!: change CLI flag structure
feat(api)!: restructure response format
# or include "BREAKING CHANGE:" in commit body
```

### Automatic Versioning Workflow

1. **Push to `main`**: Triggers automatic version detection
2. **Commit Analysis**: Scans commit messages for conventional commit patterns
3. **Version Calculation**: Determines appropriate version bump
4. **Tag Creation**: Creates and pushes new version tag
5. **Release Build**: Automatically triggers release workflow

## 🔧 Manual Release Process

### Option 1: Using GitHub Interface

1. Go to **Actions** tab in GitHub repository
2. Select "**Auto Version**" workflow
3. Click "**Run workflow**"
4. Choose version type: `patch`, `minor`, or `major`
5. Click "**Run workflow**" button

### Option 2: Using Make Commands

```bash
# Create and push a specific version tag
make tag-release VERSION=v1.2.3

# View current version info
make version

# Generate changelog
make changelog
```

### Option 3: Manual Git Tags

```bash
# Create annotated tag
git tag -a v1.2.3 -m "Release v1.2.3"

# Push tag to trigger release
git push origin v1.2.3
```

## 🏗️ Build Artifacts

Each release automatically creates:

### Binaries
- **Linux**: `amd64`, `arm64`
- **macOS**: `amd64` (Intel), `arm64` (Apple Silicon)  
- **Windows**: `amd64`

### Archives
- **Linux/macOS**: `.tar.gz` format
- **Windows**: `.zip` format

### Docker Images
- Multi-architecture images pushed to GitHub Container Registry
- Tags: `latest`, `v1.2.3`, `latest-amd64`, `v1.2.3-arm64`, etc.

### Additional Assets
- **Checksums**: `SHA256SUMS` file for verification
- **Changelog**: Auto-generated from commit messages

## 🧪 Testing Releases

### Local Development Builds

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Test version info
./build/k8s-memory-watch --version
```

### Snapshot Releases (Testing)

```bash
# Install goreleaser
make install-goreleaser

# Create local snapshot (no publishing)
make release-snapshot

# Build artifacts locally
make release-local
```

## 📋 Pre-Release Checklist

Before creating a release:

- [ ] All tests pass: `make validate`
- [ ] Security scan clean: `make security-scan`
- [ ] Documentation updated
- [ ] CHANGELOG entries added (if manual)
- [ ] Version bump justified (major/minor/patch)

## 🚨 Emergency Releases

For critical bug fixes:

1. Create hotfix branch from `main`
2. Make minimal fix with conventional commit: `fix: critical security vulnerability`
3. Open PR and merge quickly
4. Auto-versioning will create patch release
5. Or manually trigger with higher urgency

## 🔍 Release Monitoring

### GitHub Actions Status
- Monitor workflow status in **Actions** tab
- Check for failed builds or uploads
- Review artifact generation

### Release Verification
```bash
# Download and test latest release
curl -sfL https://raw.githubusercontent.com/eduardoferro/k8s-memory-watch/main/install.sh | sh

# Verify installation
k8s-memory-watch --version
```

## 📊 Version Management

### Skipping Releases

Add `[skip-release]` to commit message to prevent automatic versioning:

```bash
git commit -m "docs: update README [skip-release]"
```

### Version Override

Force specific version through workflow dispatch or manual tagging.

### Rollback Strategy

If a release has issues:

1. **Immediate**: Delete problematic tag and re-release with patch
2. **Planned**: Create new release with revert commits
3. **Emergency**: Update install script to point to last known good version

## 🛠️ Development Workflow

### Feature Development
```bash
# Create feature branch
git checkout -b feature/new-monitoring-option

# Use conventional commits
git commit -m "feat: add memory warning thresholds"

# Push and create PR
git push origin feature/new-monitoring-option
```

### Release Branch (if needed)
```bash
# For complex releases, create release branch
git checkout -b release/v2.0.0

# Finalize features, update docs, fix bugs
git commit -m "chore: prepare v2.0.0 release"

# Merge to main when ready
```

## 🔐 Security Considerations

- All release artifacts are signed with GitHub's signing key
- Docker images include security labels and vulnerability scanning
- Checksums provided for integrity verification
- Dependencies are regularly updated and scanned

## 🌟 Best Practices

1. **Keep commits atomic**: One logical change per commit
2. **Write clear commit messages**: Follow conventional commit format
3. **Test before merging**: Ensure CI passes
4. **Monitor releases**: Check that artifacts are created successfully
5. **Communicate changes**: Update documentation and inform users
6. **Version compatibility**: Maintain backwards compatibility when possible

This automated release system ensures consistent, reliable, and secure distribution of k8s-memory-watch across all supported platforms.
