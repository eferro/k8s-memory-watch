#!/bin/bash
set -e

# k8s-memory-watch installation script
# Usage: curl -sfL https://raw.githubusercontent.com/eduardoferro/k8s-memory-watch/main/install.sh | sh

REPO="eduardoferro/k8s-memory-watch"
BINARY_NAME="k8s-memory-watch"
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $OS in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        *)
            log_error "Unsupported OS: $OS"
            exit 1
            ;;
    esac
    
    case $ARCH in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    log_info "Detected platform: ${OS}-${ARCH}"
}

# Get latest release version
get_latest_version() {
    log_info "Getting latest release version..."
    VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        log_error "Failed to get latest version"
        exit 1
    fi
    
    log_info "Latest version: $VERSION"
}

# Download and install binary
install_binary() {
    ARCHIVE_NAME="${BINARY_NAME}_${VERSION#v}_${OS}_${ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"
    
    log_info "Downloading $ARCHIVE_NAME..."
    
    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    
    # Download archive
    if ! curl -sL "$DOWNLOAD_URL" -o "$ARCHIVE_NAME"; then
        log_error "Failed to download $DOWNLOAD_URL"
        exit 1
    fi
    
    # Extract archive
    log_info "Extracting archive..."
    tar -xzf "$ARCHIVE_NAME"
    
    # Make binary executable
    chmod +x "$BINARY_NAME"
    
    # Check if we can write to install directory
    if [ ! -w "$INSTALL_DIR" ]; then
        log_warn "Cannot write to $INSTALL_DIR, trying with sudo..."
        sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
    else
        mv "$BINARY_NAME" "$INSTALL_DIR/"
    fi
    
    # Clean up
    cd - > /dev/null
    rm -rf "$TMP_DIR"
    
    log_info "Successfully installed $BINARY_NAME to $INSTALL_DIR"
}

# Verify installation
verify_installation() {
    log_info "Verifying installation..."
    
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        VERSION_OUTPUT=$("$BINARY_NAME" --version)
        log_info "Installation successful!"
        echo "$VERSION_OUTPUT"
    else
        log_error "Installation failed. $BINARY_NAME not found in PATH"
        log_info "Please add $INSTALL_DIR to your PATH or move the binary to a directory in your PATH"
        exit 1
    fi
}

# Usage information
show_usage() {
    log_info "Usage examples:"
    echo "  # Monitor all namespaces"
    echo "  $BINARY_NAME --all-namespaces"
    echo ""
    echo "  # Monitor specific namespace"
    echo "  $BINARY_NAME --namespace=production"
    echo ""
    echo "  # Continuous monitoring with interval"
    echo "  $BINARY_NAME --all-namespaces --check-interval=30s"
    echo ""
    echo "  # Export to CSV"
    echo "  $BINARY_NAME --all-namespaces --output=csv > memory-report.csv"
    echo ""
    echo "For more options, run: $BINARY_NAME --help"
}

# Main installation process
main() {
    log_info "Starting k8s-memory-watch installation..."
    
    # Check prerequisites
    if ! command -v curl >/dev/null 2>&1; then
        log_error "curl is required but not installed"
        exit 1
    fi
    
    if ! command -v tar >/dev/null 2>&1; then
        log_error "tar is required but not installed"
        exit 1
    fi
    
    # Install process
    detect_platform
    get_latest_version
    install_binary
    verify_installation
    show_usage
    
    log_info "Installation completed successfully! 🎉"
}

# Run main function
main "$@"
