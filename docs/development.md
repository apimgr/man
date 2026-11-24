# Development Guide

## Prerequisites

- Docker (required for building)
- Make
- Git
- Git LFS (for the embedded man-page catalog)

## Clone

```bash
git clone https://github.com/casapps/casman
cd casman
git lfs install
git lfs pull
```

`src/server/data/manpages.db` must be a real SQLite database before building or
running the application. The checked-in file is currently only a 134-byte Git
LFS pointer, and its 241,754,112-byte object is missing from the GitHub LFS
store. The repository does not yet provide a complete reproducible catalog
builder: `scripts/loader --download` fetches a limited handpicked set, and the
no-input loader mode creates sample pages. Restoring the old pointer with
`git lfs pull` therefore cannot currently supply the catalog. See the catalog
item in `TODO.AI.md`; do not replace the production catalog with sample data.

## Build

All builds use containers - no local Go installation required.

```bash
# Quick development build
make dev

# Local build with version info
make local

# Full release build (all 8 platforms)
make build
```

## Run

```bash
# Test the dev build in Docker
docker run --rm -v /path/to/build:/app alpine:latest /app/casman --help
```

## Testing

```bash
# Run tests
make test

# Run the basic Docker binary-validation tier
./tests/run_tests.sh

# Run full service-lifecycle tests in Debian/systemd and Alpine/OpenRC (default)
./tests/incus.sh

# Run one init system only
./tests/incus.sh systemd
./tests/incus.sh openrc

# Fail if Tor or I2P cannot bootstrap
REQUIRE_OVERLAY=1 ./tests/incus.sh all

# Run browser end-to-end tests (on demand)
./tests/e2e.sh
```

The browser end-to-end harness builds the application in Docker and runs it
against a headless Chromium browser in an isolated Docker network. It requires
Docker, Python 3, and Git, and pulls the configured Go and browser images
before each run. This check is on demand and is not part of `make test`.
`tests/run_tests.sh` runs the basic Docker binary-validation tier.
`tests/incus.sh` requires both Docker and Incus. Its default `all` mode runs
the full service lifecycle in Debian Trixie with systemd and Alpine 3.24 with
OpenRC, sequentially; pass `systemd` or `openrc` to run one mode. Each mode
checks service installation and enablement, start, restart, stop/start,
container restart persistence, disable, re-install, and uninstall. Overlay
checks cover Tor and opt-in I2P. The containers need outbound network access
for Tor/I2P bootstrap; by default, an unavailable overlay is reported as a
skip while its health contract is still checked. Set `REQUIRE_OVERLAY=1` to
make unsuccessful bootstrap fail the test.
`make test` runs `go vet` and the Go unit suite in the Go container, then
enforces at least 60% total coverage. The Docker tier builds the server and
CLI through Make and exercises them in Alpine. The Incus tier uses Docker for
the build and the Debian/systemd plus Alpine/OpenRC containers for service
checks.

## Project Structure

```
src/                    # Go source code
├── main.go             # Server entry point
├── config/             # Configuration
├── server/             # HTTP server
├── client/             # CLI client
└── ...

docker/                 # Docker files
├── Dockerfile          # Multi-stage build
└── docker-compose.yml  # Production compose

docs/                   # Documentation (MkDocs)
tests/                  # Test scripts
scripts/                # Production scripts
```

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing`)
3. Make changes
4. Run tests (`make test`)
5. Submit pull request

## Code Style

- Follow Go standard formatting (`gofmt`)
- Comments above code, never inline
- Add tests for new features
- Update documentation
