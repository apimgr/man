# CLI Reference

## Installation

The CLI client is released as a separate binary:

```bash
# Download CLI
curl -LO https://github.com/casapps/casman/releases/latest/download/casman-cli-linux-amd64
chmod +x casman-cli-linux-amd64
mv casman-cli-linux-amd64 /usr/local/bin/casman-cli
```

## Usage

```bash
# Configure the server on first use. The valid address is saved to cli.yml
casman-cli --server https://your-server.example.com health

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
| `--server URL` | Server URL for this invocation; saves it only when the config has no valid server |
| `--token TOKEN` | API token |
| `--config PATH` | Select a config file or profile |
| `--no-pager` | Disable pager |
| `-h, --help` | Show help |
| `-v, --version` | Show version |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `CASMAN_SERVER_PRIMARY` | Default server URL |
| `CASMAN_SERVER_API_VERSION` | API version prefix, such as `v1` |
| `CASMAN_SERVER_TIMEOUT` | Per-request timeout; accepts Go durations or integer seconds |
| `CASMAN_SERVER_RETRY` | Number of retries after a failed request |
| `CASMAN_SERVER_RETRY_DELAY` | Delay between retries |
| `CASMAN_TOKEN` | Default API token |
| `CASMAN_TOKEN_FILE` | Read the API token from a file |

## Configuration

On Unix-like systems, the CLI uses `~/.config/casapps/casman/cli.yml`. On
Windows, it uses `%APPDATA%\\casapps\\casman\\cli.yml`. The default config
directory and file are created with user-only permissions where supported.
Custom config files are secured as files, while their parent directories are
left under the caller's control. The CLI preserves other YAML sections when it
updates a server or token value.

Server selection is `--server`, `CASMAN_SERVER_PRIMARY`, `server.primary` in
the config file, then the compiled official site when one is configured. If no
server is available, the CLI prints setup instructions. Token selection is
`--token`, `CASMAN_TOKEN`, then `auth.token`.

`--config` accepts an absolute path or a profile name relative to the default
config directory. A profile with no extension resolves to `.yml` unless an
existing `.yml` or `.yaml` file is found. Use `--config prod.yaml` to select a
specific YAML file.

The `server` section supports `primary`, `api_version` (default `v1`),
`timeout` (default `30s`), `retry` (default `3`), and `retry_delay` (default
`1s`). The `auth` section supports `token` and `token_file`; relative token
file paths are resolved beside the selected config file. A token passed with
`--token` takes precedence over `CASMAN_TOKEN`, which takes precedence over
the token file and then `auth.token`. Environment variables override the
matching config values.

## TUI Mode

After configuring a server, launch interactive mode by running `casman-cli`
without command arguments:

- Browse man pages by section
- Full-text search
- Keyboard navigation
- Vim-like keybindings
