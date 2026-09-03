# CASMAN

Universal man pages from BSD, macOS, Linux, and more - embedded in a single binary.

## Quick Start

```bash
# Docker
docker run -p 64580:80 ghcr.io/casapps/casman:latest

# Binary
./casman-linux-amd64
```

## Features

- **Embedded Man Pages** - All man pages compiled into the binary
- **Multi-Platform** - BSD (FreeBSD, OpenBSD, NetBSD), macOS, Linux
- **All Sections** - Sections 1-9, n (New/Tcl), x (X Window System)
- **Multiple Formats** - HTML, plain text, Markdown, JSON
- **Full-Text Search** - Search across all pages with highlighting
- **Compare** - Side-by-side comparison across platforms
- **REST API** - Programmatic access to all man pages
- **GraphQL** - Complex queries and relationships
- **CLI Client** - Terminal interface with TUI mode
- **PWA Support** - Installable, offline-capable

## Documentation

- [Installation](installation.md) - How to install and run
- [Configuration](configuration.md) - All configuration options
- [API Reference](api.md) - REST API, Swagger, GraphQL
- [CLI Reference](cli.md) - Command-line client
- [Admin Panel](admin.md) - WebUI administration
- [Development](development.md) - Contributing guide

## Links

- [Repository](https://github.com/casapps/casman)
- [API Documentation](/openapi) (Swagger UI)
- [GraphQL Playground](/graphql)

## License

MIT - See [LICENSE.md](https://github.com/casapps/casman/blob/main/LICENSE.md)
