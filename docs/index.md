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
- **GraphQL** - Read-only query endpoint with JSON and browser form access
- **CLI Client** - Terminal interface with TUI mode
- **PWA Support** - Cached static assets and an offline fallback page
- **Translations** - Exact stored translations by language, platform, section, and page
- **Interface languages** - Embedded English, Spanish, French, German, Italian, and Arabic catalogues
- **Guest Preferences** - Cookie-backed theme, language, and consent settings, with cross-device sync and reset
- **Owner Tokens** - Anonymous primary and linked credentials for protected integrations
- **MCP Streamable HTTP** - Bearer-authenticated API tools at `/api/v1/server/mcp` with `/api/mcp` and `/mcp` aliases

## Documentation

- [Installation](installation.md) - How to install and run
- [Configuration](configuration.md) - All configuration options
- [API Reference](api.md) - REST API, Swagger, GraphQL
- [CLI Reference](cli.md) - Command-line client
- [Server Operations](admin.md) - Configuration and command-line operations
- [Security](security.md) - Security boundaries and reporting
- [Integrations](integrations.md) - Public protocols and integrations
- [Translating](translating.md) - Translation status and contribution guidance
- [Development](development.md) - Contributing guide

## Links

- [Repository](https://github.com/casapps/casman)
- [API Documentation](/server/docs/swagger)
- [GraphQL query form](/server/docs/graphql)

## License

MIT - See [LICENSE.md](https://github.com/casapps/casman/blob/main/LICENSE.md)
