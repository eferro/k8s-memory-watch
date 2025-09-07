# k8s-memory-watch

A modern Go application for monitoring memory usage of pods and jobs in Kubernetes clusters, designed to proactively detect potential memory issues.

## Features

- **Memory Monitoring**: Track memory usage across pods and jobs
- **Proactive Alerts**: Detect potential memory issues before they become critical  
- **Kubernetes Native**: Built specifically for Kubernetes environments
- **Modern Go**: Uses Go 1.22+ features and current best practices
- **Structured Logging**: JSON-based structured logging with configurable levels
- **Graceful Shutdown**: Proper handling of termination signals

## Quick Start

### Prerequisites

- Go 1.22+ 
- Access to a Kubernetes cluster
- kubectl configured (for local development)

### Installation

```bash
# Clone the repository
git clone https://github.com/eduardoferro/k8s-memory-watch.git
cd k8s-memory-watch

# Set up development environment
make local-setup

# Build the application
make build

# Run the application
make up
```

## Installation

You can install from prebuilt binaries or build from source.

### Install from Releases (recommended)

1. Go to the Releases page:
   - `https://github.com/eferro/k8s-memory-watch/releases`
2. Download the binary for your platform:
   - `k8s-memory-watch-linux-amd64` or `k8s-memory-watch-linux-arm64`
   - `k8s-memory-watch-darwin-amd64` or `k8s-memory-watch-darwin-arm64`
   - `k8s-memory-watch-windows-amd64.exe`
3. Verify checksums with `SHA256SUMS` (optional but recommended)
4. Make it executable and move to PATH (Linux/macOS):

```bash
chmod +x k8s-memory-watch
sudo mv k8s-memory-watch /usr/local/bin/
```

### Install via script

```bash
curl -fsSL https://raw.githubusercontent.com/eferro/k8s-memory-watch/main/install.sh | bash
```

## Releases

- Releases are created automatically from tags (semantic versioning) by GitHub Actions.
- Version bump follows Conventional Commits.
- Multi-platform artifacts are published alongside `SHA256SUMS`.
- Releases page: `https://github.com/eferro/k8s-memory-watch/releases`

### Development

```bash
# Run with hot reload during development
make dev

# Run all validation checks
make validate

# Format code according to Go standards
make reformat

# Run tests with coverage
make test-unit
```

## Configuration

### Command Line Interface (Recommended)

Use command line flags for the best experience:

```bash
# Show all available options
./build/k8s-memory-watch --help

# Monitor specific namespace
./build/k8s-memory-watch --namespace=production

# Monitor all namespaces explicitly  
./build/k8s-memory-watch --all-namespaces

# Custom configuration
./build/k8s-memory-watch \
    --namespace=kube-system \
    --check-interval=1m \
    --memory-threshold=2048 \
    --memory-warning=75.0 \
    --log-level=debug
```

**Available Command Line Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--namespace` | string | Monitor specific namespace |
| `--all-namespaces` | bool | Monitor all namespaces explicitly |
| `--kubeconfig` | string | Path to kubeconfig file |
| `--in-cluster` | bool | Use in-cluster configuration |
| `--check-interval` | duration | Check interval (e.g., 30s, 1m) |
| `--memory-threshold` | int | Memory threshold in MB |
| `--memory-warning` | float | Memory warning percentage |
| `--log-level` | string | Log level (debug, info, warn, error) |
| `--help` | bool | Show help message |

### Environment Variables (Legacy)

Command line flags take precedence over environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `NAMESPACE` | (all namespaces) | Kubernetes namespace to monitor |
| `ALL_NAMESPACES` | `true` | Monitor all namespaces |
| `KUBECONFIG` | | Path to kubeconfig file (for out-of-cluster) |
| `IN_CLUSTER` | `false` | Whether running inside Kubernetes cluster |
| `CHECK_INTERVAL` | `30s` | How often to check memory usage |
| `MEMORY_THRESHOLD_MB` | `1024` | Memory threshold in MB |
| `MEMORY_WARNING_PERCENT` | `80.0` | Warning threshold as percentage |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Log format (json, text) |

## Project Structure

```
├── cmd/k8s-memory-watch/    # Application entry point
├── internal/               # Private application code
│   ├── config/            # Configuration management
│   ├── k8s/               # Kubernetes client and operations
│   └── monitor/           # Memory monitoring logic
├── pkg/metrics/           # Public packages (metrics, etc.)
├── test/integration/      # Integration tests
├── docs/                  # Documentation
├── build/                 # Build artifacts
└── tmp/                   # Temporary files (air hot reload)
```

## Available Make Targets

### Development
- `make local-setup` - Install development tools (golangci-lint, air, etc.)
- `make dev` - Run with hot reload
- `make reformat` - Format code with gofmt and goimports

### Building
- `make build` - Build the application
- `make build-linux` - Cross-compile for Linux
- `make build-docker` - Build Docker image

### Testing & Quality
- `make test-unit` - Run unit tests with coverage
- `make test-e2e` - Run integration tests
- `make check-format` - Check code formatting
- `make check-style` - Run golangci-lint
- `make check-typing` - Run go vet
- `make validate` - Run all checks

### Dependencies
- `make update` - Update Go modules
- `make add-package package=github.com/example/pkg` - Add new dependency

### Operations
- `make up` - Build and run application
- `make down` - Stop application
- `make clean` - Clean build artifacts

## Modern Go Tooling Used

- **golangci-lint**: Comprehensive linter with 40+ linters
- **air**: Live reload for Go applications during development
- **goimports**: Automatic import management
- **go vet**: Built-in static analysis
- **go modules**: Dependency management (go.mod/go.sum)
- **slog**: Structured logging (Go 1.21+ standard library)
- **context**: Proper cancellation and timeouts

## Docker Support

```bash
# Build Docker image
make build-docker

# Run with Docker
make docker-run

# Use docker-compose
make docker-compose-up
```

## Development Workflow

1. **Setup**: `make local-setup` (one-time)
2. **Develop**: `make dev` (hot reload)  
3. **Test**: `make validate` (run all checks)
4. **Build**: `make build`
5. **Deploy**: `make build-docker` + deploy

## Architecture

- **Modular Design**: Clear separation between internal packages
- **Dependency Injection**: Configuration and clients passed explicitly
- **Error Handling**: Go 1.13+ error wrapping patterns
- **Graceful Shutdown**: Context-based cancellation
- **Observability**: Structured logging and metrics ready

## Next Steps

- Implement Kubernetes client connection
- Add memory monitoring logic
- Set up metrics collection (Prometheus)
- Add health check endpoints
- Implement alerting mechanisms
