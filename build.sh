#!/bin/bash
set -e

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "=================================="
echo "🏗️  MarkView Build Script"
echo "=================================="

# 1. Build Frontend
echo ""
echo "📦 Building Frontend..."
cd frontend

if [ ! -d "node_modules" ]; then
    echo "   Installing dependencies..."
    bun install
fi

echo "   Bundling assets..."
bun run build

# Verify dist exists
if [ ! -d "dist" ]; then
    echo "❌ Error: Frontend build failed (dist directory not found)"
    exit 1
fi

cd ..

# 2. Build Backend
echo ""
echo "🐹 Building Go Binary..."

# -s: Omit the symbol table and debug information
# -w: Omit the DWARF symbol table
go build -ldflags "-s -w" -o markview.exe main.go

echo ""
echo "✅ Build Complete!"
echo "   Output: $(pwd)/markview.exe"
ls -lh markview.exe
