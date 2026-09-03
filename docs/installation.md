# Installation

## Docker (Recommended)

```bash
docker run -d \
  --name casman \
  -p 64580:80 \
  -v casman-config:/config \
  -v casman-data:/data \
  ghcr.io/casapps/casman:latest
```

## Docker Compose

```bash
curl -O https://raw.githubusercontent.com/casapps/casman/main/docker/docker-compose.yml
docker compose up -d
```

## Binary

Download from [releases](https://github.com/casapps/casman/releases):

```bash
# Linux AMD64
curl -LO https://github.com/casapps/casman/releases/latest/download/casman-linux-amd64
chmod +x casman-linux-amd64
./casman-linux-amd64
```

Available platforms:

- `casman-linux-amd64`
- `casman-linux-arm64`
- `casman-darwin-amd64`
- `casman-darwin-arm64`
- `casman-freebsd-amd64`
- `casman-freebsd-arm64`
- `casman-windows-amd64.exe`
- `casman-windows-arm64.exe`

## Systemd Service

```bash
sudo ./casman --service --install
sudo systemctl start casman
sudo systemctl enable casman
```

## Configuration

See [Configuration](configuration.md) for all options.
