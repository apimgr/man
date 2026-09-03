# API Reference

## REST API

Base URL: `/api/v1/`

### Man Page Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/man/{name}` | GET | Get man page by name |
| `/api/v1/man/{section}/{name}` | GET | Get man page by section and name |
| `/api/v1/man/{os}/{section}/{name}` | GET | Get man page for specific OS |
| `/api/v1/search?q={term}` | GET | Search man pages |
| `/api/v1/whatis/{name}` | GET | One-line description |
| `/api/v1/apropos?q={term}` | GET | Search descriptions |

### System Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/healthz` | GET | Health check |
| `/api/v1/stats` | GET | Database statistics |
| `/api/v1/sections` | GET | List sections |
| `/api/v1/platforms` | GET | List platforms |

### Search Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `q` | string | Search query |
| `section` | int | Filter by section (1-9) |
| `os` | string | Filter by OS |
| `page` | int | Page number (default: 1) |
| `limit` | int | Results per page (default: 25) |

## Swagger UI

Interactive API documentation: [/openapi](/openapi)

## GraphQL

GraphQL playground: [/graphql](/graphql)

### Example Query

```graphql
query {
  manpage(name: "ls", section: "1") {
    name
    section
    title
    synopsis
    description
  }
}
```
