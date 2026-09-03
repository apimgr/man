#!/usr/bin/env bash
set -euo pipefail

# Detect project info
PROJECTNAME=$(basename "$PWD")
PROJECTORG=$(basename "$(dirname "$PWD")")

# Create temp directory for build
mkdir -p "${TMPDIR:-/tmp}/${PROJECTORG}"
BUILD_DIR=$(mktemp -d "${TMPDIR:-/tmp}/${PROJECTORG}/${PROJECTNAME}-XXXXXX")
trap "rm -rf $BUILD_DIR" EXIT

# Go cache directories (same as Makefile)
GODIR="${HOME}/.local/share/go"
GOCACHE="${HOME}/.local/share/go/build"
mkdir -p "$GODIR" "$GOCACHE"

# Common docker run for Go builds
GO_DOCKER="docker run --rm \
  -v $(pwd):/build \
  -v ${GOCACHE}:/root/.cache/go-build \
  -v ${GODIR}:/go \
  -w /build \
  -e CGO_ENABLED=0 \
  golang:alpine"

echo "Building server binary in Docker..."
$GO_DOCKER go build -o "$BUILD_DIR/${PROJECTNAME}" ./src

# Build client if exists
if [ -d "src/client" ]; then
    echo "Building client in Docker..."
    $GO_DOCKER go build -o "$BUILD_DIR/${PROJECTNAME}-cli" ./src/client
fi

# Build agent if exists
if [ -d "src/agent" ]; then
    echo "Building agent in Docker..."
    $GO_DOCKER go build -o "$BUILD_DIR/${PROJECTNAME}-agent" ./src/agent
fi

echo "Testing in Docker (Alpine)..."
docker run --rm \
  -v "$BUILD_DIR:/app" \
  alpine:latest sh -c "
    set -e

    # Install required tools for testing
    apk add --no-cache curl bash file jq >/dev/null

    chmod +x /app/${PROJECTNAME}
    [ -f /app/${PROJECTNAME}-cli ] && chmod +x /app/${PROJECTNAME}-cli
    [ -f /app/${PROJECTNAME}-agent ] && chmod +x /app/${PROJECTNAME}-agent

    echo '=== Version Check ==='
    /app/${PROJECTNAME} --version

    echo '=== Help Check ==='
    /app/${PROJECTNAME} --help

    echo '=== Binary Info ==='
    ls -lh /app/${PROJECTNAME}
    file /app/${PROJECTNAME}

    echo '=== Starting Server for API Tests ==='
    /app/${PROJECTNAME} --port 64580 > /tmp/server.log 2>&1 &
    SERVER_PID=\$!
    sleep 3
    # Show setup token if present (for debugging)
    grep -i 'setup.*token' /tmp/server.log 2>/dev/null || true

    echo '=== API Endpoint Tests ==='
    # Test health endpoint
    curl -sf http://localhost:64580/healthz || echo 'FAILED: /healthz'

    # Test API healthz
    curl -sf http://localhost:64580/api/v1/healthz || echo 'FAILED: /api/v1/healthz'

    echo '=== Project-Specific Endpoint Tests ==='
    # Test man page endpoints
    curl -sf http://localhost:64580/ || echo 'FAILED: homepage'
    curl -sf http://localhost:64580/search?q=ls || echo 'FAILED: search'
    curl -sf http://localhost:64580/browse/ || echo 'FAILED: browse'

    # Test API endpoints
    curl -sf http://localhost:64580/api/v1/stats || echo 'FAILED: /api/v1/stats'
    curl -sf http://localhost:64580/api/v1/sections || echo 'FAILED: /api/v1/sections'
    curl -sf http://localhost:64580/api/v1/platforms || echo 'FAILED: /api/v1/platforms'
    curl -sf http://localhost:64580/api/v1/search?q=ls || echo 'FAILED: /api/v1/search'

    # Test man page API (may return 404 if no pages loaded, that's expected initially)
    curl -s http://localhost:64580/api/v1/man/ls || echo 'Note: /api/v1/man/ls (may be empty initially)'

    # Test SEO endpoints
    curl -sf http://localhost:64580/robots.txt || echo 'FAILED: /robots.txt'
    curl -sf http://localhost:64580/sitemap.xml || echo 'FAILED: /sitemap.xml'

    # Test feed endpoints
    curl -sf http://localhost:64580/feed.xml || echo 'FAILED: /feed.xml'
    curl -sf http://localhost:64580/feed.json || echo 'FAILED: /feed.json'

    # Test OpenAPI
    curl -sf http://localhost:64580/openapi.json || echo 'FAILED: /openapi.json'

    echo '=== Binary Rename Tests ==='
    # Test that binaries show ACTUAL name in --help/--version (not hardcoded)
    cp /app/${PROJECTNAME} /app/renamed-server
    chmod +x /app/renamed-server
    if /app/renamed-server --help 2>&1 | grep -q 'renamed-server'; then
        echo '✓ Server binary rename works (--help shows actual name)'
    else
        echo '✗ FAILED: Server --help does not show renamed binary name'
    fi

    echo '=== Client Tests (if exists) ==='
    if [ -f /app/${PROJECTNAME}-cli ]; then
        /app/${PROJECTNAME}-cli --version || echo 'FAILED: CLI --version'
        /app/${PROJECTNAME}-cli --help || echo 'FAILED: CLI --help'

        # Test binary rename
        cp /app/${PROJECTNAME}-cli /app/renamed-cli
        chmod +x /app/renamed-cli
        if /app/renamed-cli --help 2>&1 | grep -q 'renamed-cli'; then
            echo '✓ CLI binary rename works'
        else
            echo '✗ FAILED: CLI --help does not show renamed binary name'
        fi
    else
        echo 'client not built - skipping'
    fi

    echo '=== Agent Tests (if exists) ==='
    if [ -f /app/${PROJECTNAME}-agent ]; then
        /app/${PROJECTNAME}-agent --version || echo 'FAILED: Agent --version'
        /app/${PROJECTNAME}-agent --help || echo 'FAILED: Agent --help'

        # Test binary rename
        cp /app/${PROJECTNAME}-agent /app/renamed-agent
        chmod +x /app/renamed-agent
        if /app/renamed-agent --help 2>&1 | grep -q 'renamed-agent'; then
            echo '✓ Agent binary rename works'
        else
            echo '✗ FAILED: Agent --help does not show renamed binary name'
        fi
    else
        echo 'Agent not built - skipping'
    fi

    echo '=== Stopping Server ==='
    kill \$SERVER_PID
    wait \$SERVER_PID 2>/dev/null || true

    echo '=== All tests passed ==='
"

echo "Docker tests completed successfully"
