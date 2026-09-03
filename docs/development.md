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

# Run integration tests
./tests/run_tests.sh
```

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
