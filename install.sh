#!/bin/bash
set -e

# TheoEquity Code Review 一键安装脚本
# 用法：curl -sSL https://raw.githubusercontent.com/TheoEquity/Code-Review/main/install.sh | bash

REPO="TheoEquity/Code-Review"
BASE_URL="https://github.com/${REPO}/releases/latest/download"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

# 检测操作系统和架构
detect_os_arch() {
    local os arch

    case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
        linux*)
            os="linux"
            ;;
        darwin*)
            os="darwin"
            ;;
        *)
            error "不支持的操作系统：$(uname -s)"
            ;;
    esac

    case "$(uname -m | tr '[:upper:]' '[:lower:]')" in
        x86_64|amd64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        armv7l|armhf)
            arch="armv7"
            ;;
        *)
            error "不支持的架构：$(uname -m)"
            ;;
    esac

    echo "${os}-${arch}"
}

# 下载二进制
download_binary() {
    local os_arch="$1"
    local bin_name="opencodereview-${os_arch}"
    local download_url="${BASE_URL}/${bin_name}"
    
    info "检测平台：${os_arch}"
    info "下载地址：${download_url}"
    
    # 临时文件
    local tmp_file=$(mktemp)
    
    # 使用 curl 下载
    if command -v curl &> /dev/null; then
        curl -sSL "${download_url}" -o "${tmp_file}" || error "下载失败，请网络连接GitHub"
    else
        error "需要 curl 命令，请先安装：apt install curl 或 yum install curl"
    fi
    
    # 验证下载
    if [ ! -s "${tmp_file}" ]; then
        error "下载的文件为空，可能是Release不存在"
    fi
    
    # 移动到目标位置
    local install_dir="/usr/local/bin"
    if [ ! -w "${install_dir}" ]; then
        install_dir="${HOME}/.local/bin"
        mkdir -p "${install_dir}"
        info "没有 /usr/local/bin 写入权限，安装到 ${install_dir}"
        info "请将 ${install_dir} 加入 PATH: export PATH=${install_dir}:\$PATH"
    fi
    
    local target="${install_dir}/ocr"
    mv "${tmp_file}" "${target}"
    chmod +x "${target}"
    
    info "安装完成：${target}"
}

# 主函数
main() {
    info "=== TheoEquity Code Review 安装脚本 ==="
    
    local os_arch
    os_arch=$(detect_os_arch)
    
    download_binary "${os_arch}"
    
    # 验证安装
    if command -v ocr &> /dev/null; then
        info "验证安装成功"
        ocr version 2>/dev/null || ocr --version 2>/dev/null || true
    else
        warn "ocr 命令不在 PATH 中，请手动执行：ocr serve --addr :3030"
    fi
    
    info ""
    info "=== 安装完成 ==="
    info ""
    info "启动 Web 控制台："
    info "  ocr serve --addr :3030"
    info ""
    info "配置模型："
    info "  ocr config set llm.url https://your-llm-endpoint"
    info "  ocr config set llm.auth_token your-api-key"
    info "  ocr config set llm.model your-model-name"
    info ""
    info "访问："
    info "  http://localhost:3030"
    info ""
}

main "$@"
