#!/bin/bash
set -e

REPO="futureCreator/vcoding"
BINARY_NAME="vcoding"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       error "Unsupported OS: $(uname -s)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        arm64|aarch64)  echo "arm64" ;;
        *)              error "Unsupported architecture: $(uname -m)" ;;
    esac
}

get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$version" ]; then
        error "Failed to fetch latest version. Check your internet connection."
    fi
    echo "$version"
}

download_binary() {
    local os="$1"
    local arch="$2"
    local version="$3"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${BINARY_NAME}-${os}-${arch}"
    
    info "Downloading ${BINARY_NAME} ${version} for ${os}/${arch}..."
    
    if ! curl -fsSL "$download_url" -o "$BINARY_NAME"; then
        error "Failed to download binary from ${download_url}"
    fi
    
    chmod +x "$BINARY_NAME"
}

install_binary() {
    local install_dir
    local binary_path
    
    if [ -w "/usr/local/bin" ]; then
        install_dir="/usr/local/bin"
    elif [ -d "$HOME/.local/bin" ]; then
        install_dir="$HOME/.local/bin"
    else
        install_dir="$HOME/.local/bin"
        mkdir -p "$install_dir"
    fi
    
    binary_path="${install_dir}/${BINARY_NAME}"
    
    mv "$BINARY_NAME" "$binary_path"
    
    info "Installed ${BINARY_NAME} to ${binary_path}"
    
    if [[ ":$PATH:" != *":${install_dir}:"* ]]; then
        warn "${install_dir} is not in your PATH"
        echo ""
        echo "Add the following to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo ""
        echo "    export PATH=\"\${HOME}/.local/bin:\${PATH}\""
        echo ""
    fi
}

main() {
    echo "vcoding Installer"
    echo "================="
    echo ""
    
    local os arch version
    
    os=$(detect_os)
    arch=$(detect_arch)
    
    info "Detected: ${os}/${arch}"
    
    version=$(get_latest_version)
    info "Latest version: ${version}"
    
    download_binary "$os" "$arch" "$version"
    install_binary
    
    echo ""
    info "Installation complete!"
    echo ""
    echo "Next steps:"
    echo "  1. Run 'vcoding init' to initialize configuration"
    echo "  2. Set your OpenRouter API key: export OPENROUTER_API_KEY=your-key"
    echo "  3. Run 'vcoding doctor' to verify your setup"
    echo ""
}

main "$@"
