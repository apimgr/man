# casman

[![Build](https://github.com/casapps/casman/actions/workflows/build.yml/badge.svg)](https://github.com/casapps/casman/actions/workflows/build.yml)
[![Release](https://img.shields.io/github/v/release/casapps/casman)](https://github.com/casapps/casman/releases)
[![License](https://img.shields.io/github/license/casapps/casman)](LICENSE.md)

## About

casman is a universal man page application with ALL man pages from BSD, macOS, Linux, Void, and other Unix-like systems embedded into a single binary. Like man.cgi but completely self-contained with no external dependencies.

## Official Site

https://casman.example.com

## Features

- **Embedded Man Pages** - All man pages compiled into a single binary
- **Multi-Platform** - BSD (FreeBSD, OpenBSD, NetBSD), macOS, Linux, Void, Alpine, POSIX
- **All Sections** - Sections 1-9, n (New/Tcl), x (X Window System)
- **Multiple Formats** - HTML, plain text, Markdown, JSON, raw source
- **Search** - Full-text search with autocomplete and filters
- **Compare** - Side-by-side comparison across platforms
- **TLDR Summaries** - Auto-generated concise examples
- **Whatis/Apropos** - Traditional man page utilities
- **Translations** - Support for localized man pages (de, fr, ja, zh, etc.)
- **Bookmarks** - Save favorites locally (no account required)
- **History** - Track recently viewed pages
- **Export** - Download as PDF, EPUB, MOBI
- **RSS Feeds** - Subscribe to updates
- **PWA Support** - Installable, offline-capable
- **Dark/Light Theme** - System preference detection

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

Configuration file: `/etc/casapps/casman/server.yml` (or `~/.config/casapps/casman/server.yml`)

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
| `GET /search?q={term}` | Search man pages |
| `GET /whatis/{name}` | One-line description |
| `GET /apropos?q={term}` | Search descriptions |
| `GET /compare/{name}` | Compare across platforms |
| `GET /api/v1/*` | JSON API endpoints |

Full API documentation: `/openapi`

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
```

### Container-Only Development

All development uses containers. Never run Go directly on your machine.

```bash
# Start development environment
make dev

# The binary is built inside Docker and output to a temp directory
```

## License

MIT License - see [LICENSE.md](LICENSE.md)
