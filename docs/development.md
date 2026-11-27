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
directories by running the catalog script inside the Go toolchain container:

```bash
docker run --rm --name "casman-$(head -c6 /dev/urandom | od -An -tx1 | tr -d ' \n' | cut -c1-8)" \\
  -v "$PWD:/workspace" -w /workspace casjaysdev/go:latest \\
  bash scripts/build_catalog.sh
```

This imports the repository's source snapshot,
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
downloaders are separate from the catalog rebuild: they do not refresh the tracked
database or establish the provenance of its existing pages.

## Build

All builds use containers - no local Go installation required.

```bash
# Debug binaries for all supported platforms
make build

# Production binaries, source archive, and checksums (does not publish)
make release

# Build the local container image, run the complete gate, or inspect targets
make docker
make test
make help
make clean
```

## Run

```bash
# Run the Docker binary-validation tier or Incus service-lifecycle tier
./tests/run_tests.sh
./tests/incus.sh
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
`make test` runs vet, static analysis, vulnerability checks, shell checks, and
the Go unit suite in the Go container, enforces at least 60% total coverage,
then runs the basic binary checks. The Docker tier builds the host-platform
server and CLI debug binaries directly in the Go toolchain container and
exercises them in Alpine. Runtime test data lives under a
project-scoped OS temporary directory that the script removes when it exits.
The Incus tier uses Docker for the build and
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
