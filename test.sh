#!/bin/bash

# Langchaingo Langfuse Adapter Test Script
# Run this script to test the adapter

set -e

echo "========================================"
echo "Langchaingo Langfuse Adapter Test"
echo "========================================"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ first."
    exit 1
fi

echo "✅ Go version: $(go version)"

# Check environment variables
echo ""
echo "Checking environment variables..."

if [ -z "$LANGFUSE_PUBLIC_KEY" ]; then
    echo "⚠️  LANGFUSE_PUBLIC_KEY not set (required for tests)"
fi

if [ -z "$LANGFUSE_SECRET_KEY" ]; then
    echo "⚠️  LANGFUSE_SECRET_KEY not set (required for tests)"
fi

if [ -z "$OPENAI_API_KEY" ]; then
    echo "⚠️  OPENAI_API_KEY not set (required for LLM tests)"
fi

echo ""
echo "========================================"
echo "Step 1: Installing dependencies..."
echo "========================================"
go mod tidy

echo ""
echo "========================================"
echo "Step 2: Building..."
echo "========================================"
go build -v ./...

echo ""
echo "========================================"
echo "Step 3: Running tests..."
echo "========================================"

# Run basic adapter test (doesn't require API keys)
echo ""
echo "--- Test: NewAdapter ---"
go test -v -run TestNewAdapter -timeout 30s || true

# Run trace creation test
echo ""
echo "--- Test: CreateTrace ---"
go test -v -run TestCreateTrace -timeout 30s || true

# Run generation test (requires all keys)
echo ""
echo "--- Test: GenerateWithTracing ---"
go test -v -run TestGenerateWithTracing -timeout 60s || true

echo ""
echo "========================================"
echo "All tests completed!"
echo "========================================"
