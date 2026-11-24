# casman

[![Build](https://github.com/casapps/casman/actions/workflows/ci.yml/badge.svg)](https://github.com/casapps/casman/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/casapps/casman)](https://github.com/casapps/casman/releases)
[![License](https://img.shields.io/github/license/casapps/casman)](LICENSE.md)

## About

casman is a self-contained man page application with an embedded collection from BSD, macOS, Linux, Void, and other Unix-like systems. It provides a responsive browser interface, read-only REST and GraphQL endpoints, and a terminal client.

## Features

- **Embedded Man Pages** - A cross-platform collection compiled into the server binary
- **Multi-Platform** - BSD (FreeBSD, OpenBSD, NetBSD), macOS, Linux, Void, Alpine, POSIX
- **All Sections** - Sections 1-9, n (New/Tcl), x (X Window System)
- **Multiple Formats** - HTML, plain text, Markdown, JSON, raw source
- **Search** - Full-text search with autocomplete and filters
- **Compare** - Side-by-side comparison across platforms
- **TLDR Summaries** - Auto-generated concise examples
- **Whatis/Apropos** - Traditional man page utilities
- **Bookmarks** - Save favorites locally (no account required)
- **History** - Track recently viewed pages
- **Export** - Download as PDF, EPUB, or standalone HTML
- **Update Feeds** - Atom and JSON feeds
- **PWA Support** - Installable with cached static assets and an offline fallback page
- **Theme** - Dark by default, with light and system-following modes
- **Guest Preferences** - Cookie-backed theme, language, and consent settings, with reset and cross-device sync
- **Translations** - View exact translated man page rows when the database contains them
- **API Documentation** - Server-rendered OpenAPI route list and GraphQL query form

The catalog asset is currently unavailable: `src/server/data/manpages.db` is a
134-byte pointer to a Git LFS object that is missing from the public LFS store.
The current daily release does not include a database asset, and this checkout
does not yet have a complete reproducible catalog builder. Until the asset is
restored or regenerated, the advertised page catalog cannot be served. See the
[development guide](docs/development.md) and [alignment backlog](TODO.AI.md).

## Production

### Docker (Recommended)

```bash
docker run -d \
  --name casman \
  -p 64580:80 \
  -v ./volumes/config:/config:z \
  -v ./volumes/data:/data:z \
  ghcr.io/casapps/casman:latest
```

### Docker Compose

```bash
curl -O https://raw.githubusercontent.com/casapps/casman/main/docker/docker-compose.yml
docker compose up -d
```

### Binary

```bash
# Download latest release
curl -LO https://github.com/casapps/casman/releases/latest/download/casman-linux-amd64
chmod +x casman-linux-amd64
sudo mv casman-linux-amd64 /usr/local/bin/casman

# Run
casman

# Or install as service
sudo casman --service --install
```

## Client

The `casman-cli` client provides terminal access to man pages.

```bash
# View man page
casman-cli man ls

# Search
casman-cli search "file permission"

# Interactive TUI
casman-cli
```

## Configuration

Configuration file: `/etc/casapps/casman/server.yml` when running as root,
`/config/server.yml` in a container, or the user's OS configuration directory
when running unprivileged. Use `--config` to select another configuration
directory.

```yaml
server:
  port: 64580
  fqdn: casman.example.com
  ssl:
    enabled: false
    letsencrypt:
      enabled: false
      email: admin@example.com

database:
  driver: sqlite
  path: "{data_dir}/db/server.db"
```

## API

| Endpoint | Description |
|----------|-------------|
| `GET /man/{name}` | Get man page (HTML) |
| `GET /man/{section}/{name}` | Get man page by section |
| `GET /man/{os}/{section}/{name}` | Get man page for specific OS |
| `GET /man/{lang}/{os}/{section}/{name}` | Get an exact stored translation |
| `GET /search?q={term}` | Search man pages |
| `GET /whatis/{name}` | One-line description |
| `GET /apropos?q={term}` | Search descriptions |
| `GET /compare/{name}` | Compare across platforms |
| `GET /api/v1/*` | JSON API endpoints |
| `GET /openapi` | Server-rendered OpenAPI route documentation |
| `GET /graphql` | GraphQL query form; POST queries to the same path |
| `GET /server/preferences` | Edit browser theme, language, and consent settings |
| `GET /server/preferences/sync` | Export or import browser preferences |

Full API documentation: [`/openapi`](http://localhost:64580/openapi). The
service has no admin web UI or user accounts; configure and maintain it through
`server.yml` and the command line.

---

## Development

### Prerequisites

- Docker (required for building)
- Make

### Build

```bash
# Development build (quick, no version info)
make dev

# Local build (with version info)
make local

# Full release build (all 8 platforms)
make build

# Run tests
make test

# Run the basic Docker binary-validation tier
./tests/run_tests.sh

# Run full service-lifecycle tests on both Incus init systems (default)
./tests/incus.sh

# Or select one init system
./tests/incus.sh systemd
./tests/incus.sh openrc

# Require Tor and I2P to bootstrap; otherwise an unavailable overlay is skipped
REQUIRE_OVERLAY=1 ./tests/incus.sh all

# Run browser end-to-end tests (on demand; requires Docker, Python 3, and Git)
./tests/e2e.sh
```

The browser end-to-end harness runs the application and headless Chromium in
an isolated Docker network. It pulls its configured Go and browser images
before each run and is separate from `make test`.

`tests/run_tests.sh` runs the basic Docker tier. `tests/incus.sh` requires
Docker and Incus. Its default `all` mode runs the full service lifecycle in a
Debian Trixie systemd container and an Alpine 3.24 OpenRC container, one after
the other. Pass `systemd` or `openrc` to run only one. The service checks cover
installation, enable/start, restart, stop/start, container restart persistence,
disable, re-install, and uninstall. The containers also check the Tor and
opt-in I2P overlay contracts. They need outbound network access for overlay
bootstrap; without it, the test reports a skip while still checking the
contract. Set `REQUIRE_OVERLAY=1` to make unsuccessful Tor or I2P bootstrap a
test failure.
`make test` runs Go vet and unit tests in the Go container and enforces the
60% coverage threshold. The Docker tier builds the server and CLI with Make,
then checks their binaries and HTTP behavior in Alpine. The Incus tier also
needs Docker for the build and runs service checks in Debian/systemd and
Alpine/OpenRC containers by default.

### Container-Only Development

All development uses containers. Never run Go directly on your machine.

```bash
# Start development environment
make dev

# The binary is built inside Docker and output to a temp directory
```

## Disclaimer

This software is provided "as is" without warranty of any kind. Use it at your
own risk.

- **No warranty:** The authors are not responsible for damages, data loss, or
  issues arising from use of this software.
- **Not professional advice:** This software does not provide legal, financial,
  medical, or other professional advice.
- **Third-party services:** External services have their own terms of service.
- **Security:** No software can be guaranteed free of vulnerabilities.
- **Production use:** Evaluate the software for your environment before
  deploying it.

By using this software, you acknowledge that you have read and understood this
disclaimer.

## License

MIT License - see [LICENSE.md](LICENSE.md)
