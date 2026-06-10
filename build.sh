#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== Building OpenCodeReview ==="

# Step 1: Build frontend
echo "Building frontend..."
cd pages
npm install --silent 2>/dev/null || true
npm run build
cd ..

# Step 2: Copy frontend assets to internal/static
echo "Copying frontend assets..."
rm -rf internal/static/dist
cp -r pages/dist internal/static/dist

# Step 3: Build CLI binary
echo "Building CLI binary..."
go build -o ./dist/opencodereview ./cmd/opencodereview

echo ""
echo "=== Build complete ==="
echo "Binary: ./dist/opencodereview"
echo ""
echo "Usage:"
echo "  ./dist/opencodereview serve --addr :3030"
echo ""
