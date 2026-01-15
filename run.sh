#!/bin/bash

# SimpleArchiver 即用即走脚本
# 用法: bash <(curl -fsSL https://raw.githubusercontent.com/Lynricsy/SimpleArchiver/main/run.sh)

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 项目信息
REPO="Lynricsy/SimpleArchiver"
BINARY_NAME="simple-archiver"

# 打印带颜色的消息
info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[OK]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# 检测系统架构
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux*)   os="linux" ;;
        Darwin*)  os="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) os="windows" ;;
        *)        error "不支持的操作系统: $(uname -s)" ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)  arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        armv7l)        arch="arm" ;;
        i386|i686)     arch="386" ;;
        *)             error "不支持的架构: $(uname -m)" ;;
    esac

    echo "${os}_${arch}"
}

# 获取最新版本号
get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$version" ]; then
        echo ""
    else
        echo "$version"
    fi
}

# 下载并运行预编译版本
download_and_run() {
    local version="$1"
    local platform="$2"
    local tmp_dir
    local binary_ext=""
    
    if [[ "$platform" == windows_* ]]; then
        binary_ext=".exe"
    fi
    
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT
    
    local download_url="https://github.com/${REPO}/releases/download/${version}/${BINARY_NAME}_${platform}${binary_ext}"
    local binary_path="${tmp_dir}/${BINARY_NAME}${binary_ext}"
    
    info "正在下载 SimpleArchiver ${version} (${platform})..."
    
    if curl -fsSL -o "$binary_path" "$download_url" 2>/dev/null; then
        chmod +x "$binary_path"
        success "下载完成！正在启动..."
        echo ""
        exec "$binary_path"
    else
        return 1
    fi
}

# 使用 Go 运行
run_with_go() {
    if command -v go &> /dev/null; then
        info "使用 Go 运行 SimpleArchiver..."
        exec go run "github.com/${REPO}@latest"
    else
        return 1
    fi
}

# 克隆并编译运行
clone_and_run() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT
    
    info "正在克隆仓库并编译..."
    
    if git clone --depth 1 "https://github.com/${REPO}.git" "$tmp_dir/SimpleArchiver" 2>/dev/null; then
        cd "$tmp_dir/SimpleArchiver"
        if go build -o "${BINARY_NAME}" .; then
            success "编译完成！正在启动..."
            echo ""
            exec "./${BINARY_NAME}"
        else
            return 1
        fi
    else
        return 1
    fi
}

# 主函数
main() {
    echo -e "${CYAN}╔════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}     📦 ${GREEN}SimpleArchiver${NC} - 即用即走模式            ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    local platform
    platform=$(detect_platform)
    info "检测到系统: ${platform}"
    
    # 尝试方式1: 从 GitHub Releases 下载预编译版本
    local version
    version=$(get_latest_version)
    
    if [ -n "$version" ]; then
        info "发现最新版本: ${version}"
        if download_and_run "$version" "$platform"; then
            exit 0
        fi
        warn "预编译版本下载失败，尝试其他方式..."
    else
        warn "未找到预编译版本，尝试其他方式..."
    fi
    
    # 尝试方式2: 使用 go run 直接运行
    if command -v go &> /dev/null; then
        info "检测到 Go 环境"
        if run_with_go; then
            exit 0
        fi
        warn "go run 失败，尝试克隆编译..."
    fi
    
    # 尝试方式3: 克隆仓库并编译
    if command -v go &> /dev/null && command -v git &> /dev/null; then
        if clone_and_run; then
            exit 0
        fi
    fi
    
    # 所有方式都失败
    echo ""
    error "无法运行 SimpleArchiver。请确保安装了 Go 1.25+ 或手动下载预编译版本。

安装方法:
  1. 安装 Go: https://go.dev/dl/
  2. 运行: go install github.com/${REPO}@latest
  3. 执行: simple-archiver
"
}

main "$@"
