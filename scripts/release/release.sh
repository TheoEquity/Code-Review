#!/bin/bash
set -e

# TheoEquity Code Review 多平台编译脚本
# 用法：./scripts/release/release.sh v1.0

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}/../.."

VERSION="${1:-}"
if [ -z "${VERSION}" ]; then
    echo "Usage: ./scripts/release/release.sh <version>"
    echo "Example: ./scripts/release/release.sh v1.0"
    exit 1
fi

# 创建输出目录
DIST_DIR="${SCRIPT_DIR}/dist"
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

info() {
    echo -e "\033[0;32m[INFO]\033[0m $1"
}

# 构建前端
info "=== 构建前端 ==="
cd pages
npm install --silent 2>/dev/null || true
npm run build
cd ..

# 复制前端资源到 internal/static
info "=== 复制前端资源 ==="
rm -rf internal/static/dist
cp -r pages/dist internal/static/dist

# 编译多平台二进制
PLATFORMS=(
    "linux-amd64"
    "linux-arm64"
    "darwin-amd64"
    "darwin-arm64"
)

for platform in "${PLATFORMS[@]}"; do
    os="${platform%%-*}"
    arch="${platform##*-}"
    
    info "=== 编译 ${platform} ==="
    
    GOOS="${os}" GOARCH="${arch}" CGO_ENABLED=0 go build \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        -o "${DIST_DIR}/opencodereview-${os}-${arch}" \
        ./cmd/opencodereview
    
    info "完成：${DIST_DIR}/opencodereview-${os}-${arch}"
done

# 输出校验和
info "=== 生成校验和 ==="
cd "${DIST_DIR}"
sha256sum opencodereview-* > SHA256SUMS
cat SHA256SUMS

info ""
info "=== 编译完成 ==="
info "输出目录：${DIST_DIR}"
info ""
info "下一步："
info "1. 上传 ${DIST_DIR}/* 到 GitHub Release ${VERSION}"
info "2. 标签提交：git tag -a ${VERSION} -m 'Release ${VERSION}'"
info "3. 推送标签：git push origin ${VERSION}"
