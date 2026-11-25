# Development Guide

## Prerequisites

- Docker (required for building)
- Make
- Git

## Clone

```bash
git clone https://github.com/casapps/casman
cd casman
```

`src/server/data/manpages.db` must be a real SQLite database before building
or running the application. Rebuild it from the tracked, pre-rendered platform
directories with:

```bash
make catalog
```

This runs `scripts/build_catalog.sh`, imports the repository's source snapshot,
normalizes catalog timestamps from `SOURCE_DATE_EPOCH` (default `0`), compacts
the SQLite database with `VACUUM`, and then verifies its integrity. This
normalizes timestamps and removes unused database pages. Two consecutive
builds of the current tracked snapshot produced byte-identical databases. The
command does not download or refresh upstream content. The snapshot is not
verified as complete or latest stable because upstream source-version and
license provenance is missing. The integrity check confirms that SQLite can
read the generated database; it does not establish catalog coverage or
freshness.
No trustworthy per-page source dates are available, so catalog timestamps use
`SOURCE_DATE_EPOCH` for repeatable builds and default to `0` when it is unset.

The source inventory at `scripts/catalog-sources.tsv` is the downloader version
and provenance record. It pins Linux man-pages 6.20,
FreeBSD 15.1, OpenBSD 8.0, and NetBSD 11.1, and identifies Apple OSS inputs by
immutable commit. Linux's SHA-256 is checked against the [signed upstream sums
file](https://kernel.org/pub/linux/docs/man-pages/sha256sums.asc); BSD archive
checksums are checked against the upstream checksum files. The Apple commits
are not mapped to one macOS point release. The manifest also marks the IDEA.md
sources that remain unpinned, unimported, or unresolved for licensing. These
downloaders are separate from `make catalog`: they do not refresh the tracked
database or establish the provenance of its existing pages.

## Build

All builds use containers - no local Go installation required.

```bash
# Quick development build
make dev

# Local build with version info
make local

# Full release build (all 8 platforms)
make build

# Rebuild the embedded catalog from tracked pre-rendered platform data
make catalog
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
CLI through Make and exercises them in Alpine. Its binaries, staged releases,
and runtime test data live under a project-scoped OS temporary directory that
the script removes when it exits. The Incus tier uses Docker for the build and
the Debian/systemd plus Alpine/OpenRC containers for service checks.

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
