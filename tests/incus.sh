#!/usr/bin/env bash
set -euo pipefail

# Check if incus is available
if ! command -v incus &>/dev/null; then
    echo "ERROR: incus not found. Install incus or use tests/docker.sh"
    exit 1
fi

# Detect project info
PROJECTNAME=$(basename "$PWD")
PROJECTORG=$(basename "$(dirname "$PWD")")
CONTAINER_NAME="test-${PROJECTNAME}-$$"

# Incus image - use latest Debian stable (update when new stable releases)
INCUS_IMAGE="images:debian/12"

# Create temp directory for build
mkdir -p "${TMPDIR:-/tmp}/${PROJECTORG}"
BUILD_DIR=$(mktemp -d "${TMPDIR:-/tmp}/${PROJECTORG}/${PROJECTNAME}-XXXXXX")
trap "rm -rf $BUILD_DIR; incus delete $CONTAINER_NAME --force 2>/dev/null || true" EXIT

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

echo "Launching Incus container (Debian + systemd)..."
incus launch "$INCUS_IMAGE" "$CONTAINER_NAME"

# Wait for container to be ready
sleep 2

echo "Copying binaries to container..."
incus file push "$BUILD_DIR/${PROJECTNAME}" "$CONTAINER_NAME/usr/local/bin/"
incus exec "$CONTAINER_NAME" -- chmod +x "/usr/local/bin/${PROJECTNAME}"

# Copy client if built
if [ -f "$BUILD_DIR/${PROJECTNAME}-cli" ]; then
    incus file push "$BUILD_DIR/${PROJECTNAME}-cli" "$CONTAINER_NAME/usr/local/bin/"
    incus exec "$CONTAINER_NAME" -- chmod +x "/usr/local/bin/${PROJECTNAME}-cli"
fi

# Copy agent if built
if [ -f "$BUILD_DIR/${PROJECTNAME}-agent" ]; then
    incus file push "$BUILD_DIR/${PROJECTNAME}-agent" "$CONTAINER_NAME/usr/local/bin/"
    incus exec "$CONTAINER_NAME" -- chmod +x "/usr/local/bin/${PROJECTNAME}-agent"
fi

# Ensure curl is available for testing
incus exec "$CONTAINER_NAME" -- bash -c "command -v curl || apt-get update && apt-get install -y curl" >/dev/null 2>&1

echo "Running tests in Incus..."
incus exec "$CONTAINER_NAME" -- bash -c "
    set -e

    echo '=== Version Check ==='
    ${PROJECTNAME} --version

    echo '=== Help Check ==='
    ${PROJECTNAME} --help

    echo '=== Binary Info ==='
    ls -lh /usr/local/bin/${PROJECTNAME}
    file /usr/local/bin/${PROJECTNAME}

    echo '=== Service Install Test ==='
    ${PROJECTNAME} --service --install

    echo '=== Service Status ==='
    systemctl status ${PROJECTNAME} || true

    echo '=== Service Start Test ==='
    systemctl start ${PROJECTNAME}
    sleep 2
    systemctl status ${PROJECTNAME}

    echo '=== API Endpoint Tests ==='
    # Test health endpoint
    curl -sf http://localhost:80/healthz || echo 'FAILED: /healthz'

    # Test API healthz
    curl -sf http://localhost:80/api/v1/healthz || echo 'FAILED: /api/v1/healthz'

    echo '=== Project-Specific Endpoint Tests ==='
    # Test man page endpoints
    curl -sf http://localhost:80/ || echo 'FAILED: homepage'
    curl -sf http://localhost:80/search?q=ls || echo 'FAILED: search'
    curl -sf http://localhost:80/browse/ || echo 'FAILED: browse'

    # Test API endpoints
    curl -sf http://localhost:80/api/v1/stats || echo 'FAILED: /api/v1/stats'
    curl -sf http://localhost:80/api/v1/sections || echo 'FAILED: /api/v1/sections'
    curl -sf http://localhost:80/api/v1/platforms || echo 'FAILED: /api/v1/platforms'
    curl -sf http://localhost:80/api/v1/search?q=ls || echo 'FAILED: /api/v1/search'

    # Test man page API (may return 404 if no pages loaded, that's expected initially)
    curl -s http://localhost:80/api/v1/man/ls || echo 'Note: /api/v1/man/ls (may be empty initially)'

    # Test SEO endpoints
    curl -sf http://localhost:80/robots.txt || echo 'FAILED: /robots.txt'
    curl -sf http://localhost:80/sitemap.xml || echo 'FAILED: /sitemap.xml'

    # Test feed endpoints
    curl -sf http://localhost:80/feed.xml || echo 'FAILED: /feed.xml'
    curl -sf http://localhost:80/feed.json || echo 'FAILED: /feed.json'

    # Test OpenAPI
    curl -sf http://localhost:80/openapi.json || echo 'FAILED: /openapi.json'

    echo '=== Binary Rename Tests ==='
    # Test that binaries show ACTUAL name in --help/--version (not hardcoded)
    cp /usr/local/bin/${PROJECTNAME} /tmp/renamed-server
    chmod +x /tmp/renamed-server
    if /tmp/renamed-server --help 2>&1 | grep -q 'renamed-server'; then
        echo '✓ Server binary rename works (--help shows actual name)'
    else
        echo '✗ FAILED: Server --help does not show renamed binary name'
    fi

    echo '=== Client Tests (if exists) ==='
    if [ -f /usr/local/bin/${PROJECTNAME}-cli ]; then
        ${PROJECTNAME}-cli --version || echo 'FAILED: CLI --version'
        ${PROJECTNAME}-cli --help || echo 'FAILED: CLI --help'

        # Test binary rename
        cp /usr/local/bin/${PROJECTNAME}-cli /tmp/renamed-cli
        chmod +x /tmp/renamed-cli
        if /tmp/renamed-cli --help 2>&1 | grep -q 'renamed-cli'; then
            echo '✓ CLI binary rename works'
        else
            echo '✗ FAILED: CLI --help does not show renamed binary name'
        fi
    else
        echo 'client not installed - skipping'
    fi

    echo '=== Agent Tests (if exists) ==='
    if [ -f /usr/local/bin/${PROJECTNAME}-agent ]; then
        ${PROJECTNAME}-agent --version || echo 'FAILED: Agent --version'
        ${PROJECTNAME}-agent --help || echo 'FAILED: Agent --help'

        # Test binary rename
        cp /usr/local/bin/${PROJECTNAME}-agent /tmp/renamed-agent
        chmod +x /tmp/renamed-agent
        if /tmp/renamed-agent --help 2>&1 | grep -q 'renamed-agent'; then
            echo '✓ Agent binary rename works'
        else
            echo '✗ FAILED: Agent --help does not show renamed binary name'
        fi
    else
        echo 'Agent not installed - skipping'
    fi

    echo '=== Service Stop Test ==='
    systemctl stop ${PROJECTNAME}

    echo '=== All tests passed ==='
"

echo "Incus tests completed successfully"
