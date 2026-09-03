# CLI Reference

## Installation

The CLI client is included with the server binary or available separately:

```bash
# Download CLI
curl -LO https://github.com/casapps/casman/releases/latest/download/casman-cli-linux-amd64
chmod +x casman-cli-linux-amd64
mv casman-cli-linux-amd64 /usr/local/bin/casman-cli
```

## Usage

```bash
# Interactive TUI mode
casman-cli

# View man page
casman-cli man ls
casman-cli man 5 passwd

# Search
casman-cli search "file permission"

# Other commands
casman-cli whatis chmod
casman-cli apropos permission
casman-cli stats
```

## Commands

| Command | Description |
|---------|-------------|
| `man [SECTION] NAME` | View man page |
| `search QUERY` | Search man pages |
| `whatis NAME` | One-line description |
| `apropos QUERY` | Search descriptions |
| `stats` | Database statistics |
| `sections` | List sections |
| `platforms` | List platforms |
| `health` | Check server health |

## Global Options

| Option | Description |
|--------|-------------|
| `--server URL` | Server URL (default: http://localhost:64580) |
| `--token TOKEN` | API token |
| `--no-pager` | Disable pager |
| `-h, --help` | Show help |
| `-v, --version` | Show version |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `CASMAN_SERVER` | Default server URL |
| `CASMAN_TOKEN` | Default API token |

## TUI Mode

Launch interactive mode by running `casman-cli` without arguments:

- Browse man pages by section
- Full-text search
- Keyboard navigation
- Vim-like keybindings
